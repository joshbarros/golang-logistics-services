package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// AuthorizationHeader is the header name for authorization
	AuthorizationHeader = "Authorization"
	// BearerPrefix is the prefix for bearer tokens
	BearerPrefix = "Bearer "
	// ClaimsContextKey is the context key for JWT claims
	ClaimsContextKey = "jwt_claims"
	// UserIDContextKey is the context key for user ID
	UserIDContextKey = "user_id"
)

// AuthMiddleware creates a middleware for JWT authentication
func AuthMiddleware(jwtManager *JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader(AuthorizationHeader)
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header is required",
			})
			c.Abort()
			return
		}

		// Check Bearer prefix
		if !strings.HasPrefix(authHeader, BearerPrefix) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format. Expected: Bearer <token>",
			})
			c.Abort()
			return
		}

		// Extract token
		tokenString := strings.TrimPrefix(authHeader, BearerPrefix)
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Token is required",
			})
			c.Abort()
			return
		}

		// Validate token
		claims, err := jwtManager.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Optional: Validate client IP
		if claims.ClientIP != "" && claims.ClientIP != c.ClientIP() {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Token was issued for different IP address",
			})
			c.Abort()
			return
		}

		// Store claims in context
		c.Set(ClaimsContextKey, claims)
		c.Set(UserIDContextKey, claims.UserID)

		c.Next()
	}
}

// OptionalAuthMiddleware is like AuthMiddleware but doesn't abort if no token
func OptionalAuthMiddleware(jwtManager *JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(AuthorizationHeader)
		if authHeader == "" {
			c.Next()
			return
		}

		if !strings.HasPrefix(authHeader, BearerPrefix) {
			c.Next()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, BearerPrefix)
		if tokenString == "" {
			c.Next()
			return
		}

		claims, err := jwtManager.ValidateToken(tokenString)
		if err != nil {
			c.Next()
			return
		}

		c.Set(ClaimsContextKey, claims)
		c.Set(UserIDContextKey, claims.UserID)

		c.Next()
	}
}

// RequireRole creates a middleware that requires specific roles
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get claims from context
		claimsInterface, exists := c.Get(ClaimsContextKey)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			c.Abort()
			return
		}

		claims, ok := claimsInterface.(*Claims)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Invalid authentication claims",
			})
			c.Abort()
			return
		}

		// Check if user has any of the required roles
		if !claims.HasAnyRole(roles...) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":          "Insufficient permissions",
				"required_roles": roles,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAllRoles creates a middleware that requires all specified roles
func RequireAllRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsInterface, exists := c.Get(ClaimsContextKey)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			c.Abort()
			return
		}

		claims, ok := claimsInterface.(*Claims)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Invalid authentication claims",
			})
			c.Abort()
			return
		}

		if !claims.HasAllRoles(roles...) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":          "Insufficient permissions",
				"required_roles": roles,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetClaims retrieves JWT claims from Gin context
func GetClaims(c *gin.Context) (*Claims, bool) {
	claimsInterface, exists := c.Get(ClaimsContextKey)
	if !exists {
		return nil, false
	}

	claims, ok := claimsInterface.(*Claims)
	return claims, ok
}

// GetUserID retrieves user ID from Gin context
func GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get(UserIDContextKey)
	if !exists {
		return "", false
	}

	id, ok := userID.(string)
	return id, ok
}

// MustGetClaims retrieves JWT claims or panics
func MustGetClaims(c *gin.Context) *Claims {
	claims, ok := GetClaims(c)
	if !ok {
		panic("claims not found in context")
	}
	return claims
}

// MustGetUserID retrieves user ID or panics
func MustGetUserID(c *gin.Context) string {
	userID, ok := GetUserID(c)
	if !ok {
		panic("user ID not found in context")
	}
	return userID
}
