package ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// Limiter defines the interface for rate limiting
type Limiter interface {
	Allow(ctx context.Context, key string) (bool, error)
	Reset(ctx context.Context, key string) error
}

// RedisLimiter implements rate limiting using Redis
type RedisLimiter struct {
	client   *redis.Client
	rate     int           // requests per window
	window   time.Duration // time window
	keyPrefix string
}

// NewRedisLimiter creates a new Redis-based rate limiter
func NewRedisLimiter(client *redis.Client, rate int, window time.Duration) *RedisLimiter {
	return &RedisLimiter{
		client:    client,
		rate:      rate,
		window:    window,
		keyPrefix: "ratelimit:",
	}
}

// Allow checks if a request is allowed
func (l *RedisLimiter) Allow(ctx context.Context, key string) (bool, error) {
	redisKey := l.keyPrefix + key

	// Use Lua script for atomic increment and check
	script := redis.NewScript(`
		local key = KEYS[1]
		local limit = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])

		local current = redis.call('GET', key)
		if current == false then
			redis.call('SET', key, 1, 'EX', window)
			return 1
		end

		current = tonumber(current)
		if current < limit then
			redis.call('INCR', key)
			return 1
		end

		return 0
	`)

	result, err := script.Run(ctx, l.client, []string{redisKey}, l.rate, int(l.window.Seconds())).Int()
	if err != nil {
		return false, fmt.Errorf("redis error: %w", err)
	}

	return result == 1, nil
}

// Reset resets the rate limit for a key
func (l *RedisLimiter) Reset(ctx context.Context, key string) error {
	redisKey := l.keyPrefix + key
	return l.client.Del(ctx, redisKey).Err()
}

// GetRemaining returns remaining requests for a key
func (l *RedisLimiter) GetRemaining(ctx context.Context, key string) (int, error) {
	redisKey := l.keyPrefix + key

	current, err := l.client.Get(ctx, redisKey).Int()
	if err == redis.Nil {
		return l.rate, nil
	}
	if err != nil {
		return 0, err
	}

	remaining := l.rate - current
	if remaining < 0 {
		remaining = 0
	}

	return remaining, nil
}

// GetTTL returns time until reset for a key
func (l *RedisLimiter) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	redisKey := l.keyPrefix + key
	return l.client.TTL(ctx, redisKey).Result()
}

// KeyExtractor is a function that extracts the rate limit key from a request
type KeyExtractor func(*gin.Context) string

// IPKeyExtractor extracts client IP as the key
func IPKeyExtractor(c *gin.Context) string {
	return c.ClientIP()
}

// UserIDKeyExtractor extracts user ID from JWT claims as the key
func UserIDKeyExtractor(c *gin.Context) string {
	userID, exists := c.Get("user_id")
	if !exists {
		return c.ClientIP() // Fallback to IP
	}
	return fmt.Sprintf("user:%v", userID)
}

// APIKeyExtractor extracts API key from header as the key
func APIKeyExtractor(headerName string) KeyExtractor {
	return func(c *gin.Context) string {
		apiKey := c.GetHeader(headerName)
		if apiKey == "" {
			return c.ClientIP() // Fallback to IP
		}
		return fmt.Sprintf("apikey:%s", apiKey)
	}
}

// CompositeKeyExtractor combines multiple extractors
func CompositeKeyExtractor(extractors ...KeyExtractor) KeyExtractor {
	return func(c *gin.Context) string {
		key := ""
		for _, extractor := range extractors {
			part := extractor(c)
			if key == "" {
				key = part
			} else {
				key += ":" + part
			}
		}
		return key
	}
}

// RateLimitConfig defines rate limit configuration
type RateLimitConfig struct {
	Limiter      Limiter
	KeyExtractor KeyExtractor
	OnLimit      func(*gin.Context) // Optional custom handler when limit exceeded
}

// Middleware creates a rate limiting middleware
func Middleware(config RateLimitConfig) gin.HandlerFunc {
	if config.KeyExtractor == nil {
		config.KeyExtractor = IPKeyExtractor
	}

	return func(c *gin.Context) {
		key := config.KeyExtractor(c)

		allowed, err := config.Limiter.Allow(c.Request.Context(), key)
		if err != nil {
			// Log error but don't block request (fail open)
			c.Next()
			return
		}

		if !allowed {
			// Get remaining and retry info
			if redisLimiter, ok := config.Limiter.(*RedisLimiter); ok {
				remaining, _ := redisLimiter.GetRemaining(c.Request.Context(), key)
				ttl, _ := redisLimiter.GetTTL(c.Request.Context(), key)

				c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", redisLimiter.rate))
				c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
				c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(ttl).Unix()))
				c.Header("Retry-After", fmt.Sprintf("%d", int(ttl.Seconds())))
			}

			if config.OnLimit != nil {
				config.OnLimit(c)
				return
			}

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
				"message": "Too many requests. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// InMemoryLimiter implements a simple in-memory rate limiter (for development)
type InMemoryLimiter struct {
	rate      int
	window    time.Duration
	requests  map[string][]time.Time
}

// NewInMemoryLimiter creates a new in-memory rate limiter
func NewInMemoryLimiter(rate int, window time.Duration) *InMemoryLimiter {
	return &InMemoryLimiter{
		rate:     rate,
		window:   window,
		requests: make(map[string][]time.Time),
	}
}

// Allow checks if a request is allowed (in-memory implementation)
func (l *InMemoryLimiter) Allow(ctx context.Context, key string) (bool, error) {
	now := time.Now()
	cutoff := now.Add(-l.window)

	// Clean old requests
	requests := l.requests[key]
	validRequests := []time.Time{}
	for _, t := range requests {
		if t.After(cutoff) {
			validRequests = append(validRequests, t)
		}
	}

	// Check limit
	if len(validRequests) >= l.rate {
		l.requests[key] = validRequests
		return false, nil
	}

	// Add current request
	validRequests = append(validRequests, now)
	l.requests[key] = validRequests

	return true, nil
}

// Reset resets the rate limit for a key
func (l *InMemoryLimiter) Reset(ctx context.Context, key string) error {
	delete(l.requests, key)
	return nil
}
