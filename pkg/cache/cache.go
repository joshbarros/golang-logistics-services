package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	// ErrCacheMiss is returned when a key is not found in cache
	ErrCacheMiss = errors.New("cache miss")
	// ErrCacheUnavailable is returned when cache is not available
	ErrCacheUnavailable = errors.New("cache unavailable")
)

// Cache is the interface for cache operations
type Cache interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, keys ...string) (int64, error)
	Invalidate(ctx context.Context, pattern string) error
	Close() error
}

// RedisCache implements Cache using Redis
type RedisCache struct {
	client    *redis.Client
	keyPrefix string
}

// NewRedisCache creates a new Redis cache
func NewRedisCache(addr, password, keyPrefix string, db int) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{
		client:    client,
		keyPrefix: keyPrefix,
	}, nil
}

// Get retrieves a value from cache and unmarshals into dest
func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	fullKey := c.prefixKey(key)

	data, err := c.client.Get(ctx, fullKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return ErrCacheMiss
		}
		return fmt.Errorf("cache get error: %w", err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("cache unmarshal error: %w", err)
	}

	return nil
}

// Set stores a value in cache with expiration
func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	fullKey := c.prefixKey(key)

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache marshal error: %w", err)
	}

	if err := c.client.Set(ctx, fullKey, data, expiration).Err(); err != nil {
		return fmt.Errorf("cache set error: %w", err)
	}

	return nil
}

// Delete removes keys from cache
func (c *RedisCache) Delete(ctx context.Context, keys ...string) error {
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = c.prefixKey(key)
	}

	if err := c.client.Del(ctx, fullKeys...).Err(); err != nil {
		return fmt.Errorf("cache delete error: %w", err)
	}

	return nil
}

// Exists checks if keys exist in cache
func (c *RedisCache) Exists(ctx context.Context, keys ...string) (int64, error) {
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = c.prefixKey(key)
	}

	count, err := c.client.Exists(ctx, fullKeys...).Result()
	if err != nil {
		return 0, fmt.Errorf("cache exists error: %w", err)
	}

	return count, nil
}

// Invalidate removes all keys matching pattern
func (c *RedisCache) Invalidate(ctx context.Context, pattern string) error {
	fullPattern := c.prefixKey(pattern)

	iter := c.client.Scan(ctx, 0, fullPattern, 0).Iterator()
	var keys []string

	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("cache scan error: %w", err)
	}

	if len(keys) > 0 {
		if err := c.client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("cache invalidate error: %w", err)
		}
	}

	return nil
}

// Close closes the Redis connection
func (c *RedisCache) Close() error {
	return c.client.Close()
}

// prefixKey adds prefix to key
func (c *RedisCache) prefixKey(key string) string {
	if c.keyPrefix == "" {
		return key
	}
	return fmt.Sprintf("%s:%s", c.keyPrefix, key)
}

// InMemoryCache implements Cache using in-memory map (fallback)
type InMemoryCache struct {
	data   map[string]cacheItem
	mutex  sync.RWMutex
	janitor *janitor
}

type cacheItem struct {
	value      []byte
	expiration time.Time
}

// NewInMemoryCache creates a new in-memory cache
func NewInMemoryCache(cleanupInterval time.Duration) *InMemoryCache {
	cache := &InMemoryCache{
		data: make(map[string]cacheItem),
	}

	// Start cleanup goroutine
	cache.janitor = &janitor{
		interval: cleanupInterval,
		stop:     make(chan bool),
	}
	go cache.janitor.run(cache)

	return cache
}

// Get retrieves a value from cache
func (c *InMemoryCache) Get(ctx context.Context, key string, dest interface{}) error {
	c.mutex.RLock()
	item, found := c.data[key]
	c.mutex.RUnlock()

	if !found {
		return ErrCacheMiss
	}

	if !item.expiration.IsZero() && time.Now().After(item.expiration) {
		c.Delete(ctx, key)
		return ErrCacheMiss
	}

	if err := json.Unmarshal(item.value, dest); err != nil {
		return fmt.Errorf("cache unmarshal error: %w", err)
	}

	return nil
}

// Set stores a value in cache
func (c *InMemoryCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache marshal error: %w", err)
	}

	var exp time.Time
	if expiration > 0 {
		exp = time.Now().Add(expiration)
	}

	c.mutex.Lock()
	c.data[key] = cacheItem{
		value:      data,
		expiration: exp,
	}
	c.mutex.Unlock()

	return nil
}

// Delete removes keys from cache
func (c *InMemoryCache) Delete(ctx context.Context, keys ...string) error {
	c.mutex.Lock()
	for _, key := range keys {
		delete(c.data, key)
	}
	c.mutex.Unlock()
	return nil
}

// Exists checks if keys exist
func (c *InMemoryCache) Exists(ctx context.Context, keys ...string) (int64, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var count int64
	now := time.Now()

	for _, key := range keys {
		if item, found := c.data[key]; found {
			if item.expiration.IsZero() || now.Before(item.expiration) {
				count++
			}
		}
	}

	return count, nil
}

// Invalidate removes all keys (pattern matching not supported in memory)
func (c *InMemoryCache) Invalidate(ctx context.Context, pattern string) error {
	// Simple implementation - clear all if pattern is "*"
	if pattern == "*" {
		c.mutex.Lock()
		c.data = make(map[string]cacheItem)
		c.mutex.Unlock()
	}
	return nil
}

// Close stops the cleanup goroutine
func (c *InMemoryCache) Close() error {
	c.janitor.stop <- true
	return nil
}

// deleteExpired removes expired items
func (c *InMemoryCache) deleteExpired() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	for key, item := range c.data {
		if !item.expiration.IsZero() && now.After(item.expiration) {
			delete(c.data, key)
		}
	}
}

// janitor handles cleanup of expired items
type janitor struct {
	interval time.Duration
	stop     chan bool
}

func (j *janitor) run(cache *InMemoryCache) {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cache.deleteExpired()
		case <-j.stop:
			return
		}
	}
}

// CacheWrapper provides cache-aside pattern with automatic fallback
type CacheWrapper struct {
	cache Cache
}

// NewCacheWrapper creates a new cache wrapper
func NewCacheWrapper(cache Cache) *CacheWrapper {
	return &CacheWrapper{cache: cache}
}

// GetOrSet retrieves from cache or executes function and caches result
func (w *CacheWrapper) GetOrSet(ctx context.Context, key string, dest interface{}, ttl time.Duration, fn func() (interface{}, error)) error {
	// Try to get from cache first
	err := w.cache.Get(ctx, key, dest)
	if err == nil {
		return nil
	}

	if err != ErrCacheMiss {
		// Cache error, continue without cache
	}

	// Execute function to get data
	result, err := fn()
	if err != nil {
		return err
	}

	// Store in cache (ignore errors)
	w.cache.Set(ctx, key, result, ttl)

	// Marshal/unmarshal to populate dest
	data, _ := json.Marshal(result)
	json.Unmarshal(data, dest)

	return nil
}

// Example usage:
//
// // Setup cache
// cache, err := cache.NewRedisCache("localhost:6379", "", "order-service", 0)
// if err != nil {
//     // Fallback to in-memory cache
//     cache = cache.NewInMemoryCache(5 * time.Minute)
// }
// defer cache.Close()
//
// wrapper := cache.NewCacheWrapper(cache)
//
// // Use cache-aside pattern
// var order Order
// err = wrapper.GetOrSet(
//     ctx,
//     fmt.Sprintf("order:%s", orderID),
//     &order,
//     5*time.Minute,
//     func() (interface{}, error) {
//         // Fetch from database
//         return orderRepo.GetByID(orderID)
//     },
// )
