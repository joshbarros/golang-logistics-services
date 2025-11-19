package graphql

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Middleware provides common GraphQL middleware functions

// AuthMiddleware validates authentication tokens
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Validate Bearer token format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		token := parts[1]

		// Validate JWT token (simplified - use proper JWT library in production)
		// In production, use github.com/golang-jwt/jwt
		userID, err := validateToken(token, jwtSecret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Add user info to context
		c.Set("user_id", userID)
		c.Next()
	}
}

// validateToken validates a JWT token and returns user ID
func validateToken(token, secret string) (string, error) {
	// Implement proper JWT validation
	// This is a placeholder
	if token == "" {
		return "", fmt.Errorf("empty token")
	}
	return "user-123", nil
}

// RateLimitMiddleware limits requests per user/IP
func RateLimitMiddleware(requestsPerMinute int) gin.HandlerFunc {
	// In production, use Redis-based rate limiting
	type rateLimiter struct {
		requests []time.Time
	}

	limiters := make(map[string]*rateLimiter)

	return func(c *gin.Context) {
		// Get identifier (user ID or IP)
		identifier := c.GetString("user_id")
		if identifier == "" {
			identifier = c.ClientIP()
		}

		// Get or create rate limiter
		limiter, exists := limiters[identifier]
		if !exists {
			limiter = &rateLimiter{requests: make([]time.Time, 0)}
			limiters[identifier] = limiter
		}

		// Remove old requests (older than 1 minute)
		now := time.Now()
		cutoff := now.Add(-1 * time.Minute)
		validRequests := make([]time.Time, 0)
		for _, t := range limiter.requests {
			if t.After(cutoff) {
				validRequests = append(validRequests, t)
			}
		}
		limiter.requests = validRequests

		// Check rate limit
		if len(limiter.requests) >= requestsPerMinute {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
				"retry_after": limiter.requests[0].Add(1 * time.Minute).Sub(now).Seconds(),
			})
			c.Abort()
			return
		}

		// Add current request
		limiter.requests = append(limiter.requests, now)

		c.Next()
	}
}

// LoggingMiddleware logs GraphQL queries and mutations
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Read request body
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Next()
			return
		}

		// Parse GraphQL request
		var gqlRequest struct {
			Query         string                 `json:"query"`
			OperationName string                 `json:"operationName"`
			Variables     map[string]interface{} `json:"variables"`
		}

		if err := json.Unmarshal(body, &gqlRequest); err == nil {
			// Log query info
			operation := extractOperation(gqlRequest.Query)
			fmt.Printf("[GraphQL] %s - %s - User: %v\n",
				time.Now().Format(time.RFC3339),
				operation,
				c.GetString("user_id"),
			)
		}

		// Restore body for next handler
		c.Request.Body = io.NopCloser(strings.NewReader(string(body)))

		c.Next()
	}
}

// extractOperation extracts the operation type from query
func extractOperation(query string) string {
	query = strings.TrimSpace(query)
	if strings.HasPrefix(query, "mutation") {
		return "Mutation"
	} else if strings.HasPrefix(query, "subscription") {
		return "Subscription"
	}
	return "Query"
}

// ComplexityMiddleware limits query complexity
func ComplexityMiddleware(maxComplexity int) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Read request body
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Next()
			return
		}

		// Parse GraphQL request
		var gqlRequest struct {
			Query string `json:"query"`
		}

		if err := json.Unmarshal(body, &gqlRequest); err == nil {
			complexity := calculateComplexity(gqlRequest.Query)
			if complexity > maxComplexity {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("Query too complex: %d (max: %d)", complexity, maxComplexity),
				})
				c.Abort()
				return
			}
		}

		// Restore body
		c.Request.Body = io.NopCloser(strings.NewReader(string(body)))

		c.Next()
	}
}

// calculateComplexity calculates query complexity score
func calculateComplexity(query string) int {
	// Simplified complexity calculation
	// In production, use proper GraphQL parser and cost analysis

	complexity := 0

	// Count fields (each field adds 1)
	complexity += strings.Count(query, "\n")

	// Nested queries increase complexity
	complexity += strings.Count(query, "{") * 2

	// Lists multiply complexity
	complexity += strings.Count(query, "[") * 5

	return complexity
}

// DepthMiddleware limits query depth
func DepthMiddleware(maxDepth int) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Read request body
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Next()
			return
		}

		// Parse GraphQL request
		var gqlRequest struct {
			Query string `json:"query"`
		}

		if err := json.Unmarshal(body, &gqlRequest); err == nil {
			depth := calculateDepth(gqlRequest.Query)
			if depth > maxDepth {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("Query too deep: %d (max: %d)", depth, maxDepth),
				})
				c.Abort()
				return
			}
		}

		// Restore body
		c.Request.Body = io.NopCloser(strings.NewReader(string(body)))

		c.Next()
	}
}

// calculateDepth calculates query depth
func calculateDepth(query string) int {
	maxDepth := 0
	currentDepth := 0

	for _, char := range query {
		if char == '{' {
			currentDepth++
			if currentDepth > maxDepth {
				maxDepth = currentDepth
			}
		} else if char == '}' {
			currentDepth--
		}
	}

	return maxDepth
}

// CORSMiddleware handles CORS for GraphQL requests
func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// Handle preflight
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// TimeoutMiddleware enforces request timeout
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		// Channel to signal completion
		done := make(chan struct{})

		go func() {
			c.Next()
			close(done)
		}()

		select {
		case <-done:
			// Request completed
		case <-ctx.Done():
			// Timeout
			c.JSON(http.StatusRequestTimeout, gin.H{
				"error": "Request timeout",
			})
			c.Abort()
		}
	}
}

// CacheMiddleware caches GraphQL query results
func CacheMiddleware(ttl time.Duration) gin.HandlerFunc {
	cache := make(map[string]cacheEntry)

	type cacheEntry struct {
		response  string
		expiresAt time.Time
	}

	return func(c *gin.Context) {
		// Only cache GET requests (queries)
		if c.Request.Method != "GET" {
			c.Next()
			return
		}

		// Read request body to create cache key
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Next()
			return
		}

		cacheKey := string(body)

		// Check cache
		if entry, exists := cache[cacheKey]; exists && time.Now().Before(entry.expiresAt) {
			c.Header("X-Cache", "HIT")
			c.String(http.StatusOK, entry.response)
			c.Abort()
			return
		}

		// Restore body
		c.Request.Body = io.NopCloser(strings.NewReader(string(body)))

		// Capture response
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           &strings.Builder{},
		}
		c.Writer = writer

		c.Next()

		// Cache response if successful
		if c.Writer.Status() == http.StatusOK {
			cache[cacheKey] = cacheEntry{
				response:  writer.body.String(),
				expiresAt: time.Now().Add(ttl),
			}
		}
	}
}

type responseWriter struct {
	gin.ResponseWriter
	body *strings.Builder
}

func (w *responseWriter) Write(data []byte) (int, error) {
	w.body.Write(data)
	return w.ResponseWriter.Write(data)
}

// MetricsMiddleware tracks GraphQL metrics
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Read request body
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Next()
			return
		}

		// Parse GraphQL request
		var gqlRequest struct {
			Query         string `json:"query"`
			OperationName string `json:"operationName"`
		}

		operation := "unknown"
		if err := json.Unmarshal(body, &gqlRequest); err == nil {
			operation = extractOperation(gqlRequest.Query)
			if gqlRequest.OperationName != "" {
				operation = gqlRequest.OperationName
			}
		}

		// Restore body
		c.Request.Body = io.NopCloser(strings.NewReader(string(body)))

		c.Next()

		// Record metrics
		duration := time.Since(start)
		status := c.Writer.Status()

		fmt.Printf("[Metrics] Operation: %s, Duration: %v, Status: %d\n",
			operation, duration, status)

		// In production, send to Prometheus/StatsD:
		// metrics.RecordGraphQLOperation(operation, duration, status)
	}
}

// Example usage:
//
// router := gin.Default()
//
// // Apply middleware
// router.Use(CORSMiddleware([]string{"http://localhost:3000"}))
// router.Use(AuthMiddleware("jwt-secret"))
// router.Use(RateLimitMiddleware(100)) // 100 requests per minute
// router.Use(LoggingMiddleware())
// router.Use(ComplexityMiddleware(1000))
// router.Use(DepthMiddleware(10))
// router.Use(TimeoutMiddleware(30 * time.Second))
// router.Use(MetricsMiddleware())
//
// // GraphQL endpoint
// router.POST("/graphql", gqlServer.Handler())
// router.GET("/playground", gqlServer.PlaygroundHandler())
