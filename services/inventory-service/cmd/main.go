package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/joshbarros/golang-logistics-services/pkg/auth"
	"github.com/joshbarros/golang-logistics-services/pkg/config"
	"github.com/joshbarros/golang-logistics-services/pkg/health"
	"github.com/joshbarros/golang-logistics-services/pkg/logger"
	"github.com/joshbarros/golang-logistics-services/pkg/metrics"
	"github.com/joshbarros/golang-logistics-services/pkg/middleware"
	"github.com/joshbarros/golang-logistics-services/pkg/ratelimit"
	"github.com/joshbarros/golang-logistics-services/pkg/server"
	"github.com/joshbarros/golang-logistics-services/pkg/validator"
	"github.com/joshbarros/golang-logistics-services/services/inventory-service/internal/database"
	"github.com/joshbarros/golang-logistics-services/services/inventory-service/internal/handlers"
	"github.com/joshbarros/golang-logistics-services/services/inventory-service/internal/repository"
	"github.com/joshbarros/golang-logistics-services/services/inventory-service/internal/service"
)

const (
	serviceName = "inventory-service"
	version     = "2.0.0"
)

func main() {
	// Initialize structured logger
	log := logger.New(serviceName)
	log.SetLevel(logger.FromEnv())
	log.Info("Starting Inventory Service...", logger.Fields{"version": version})

	// Connect to database
	db, err := database.Connect()
	if err != nil {
		log.Fatal("Failed to connect to database", logger.Fields{"error": err})
	}

	// Get underlying SQL DB for connection pool configuration and health checks
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get database connection", logger.Fields{"error": err})
	}

	// Configure connection pool
	health.ConfigureDBConnectionPool(sqlDB)
	log.Info("Database connection pool configured", logger.Fields{
		"max_open_conns":     25,
		"max_idle_conns":     5,
		"conn_max_lifetime": "5m",
	})

	// Initialize JWT manager
	jwtSecret := getEnv("JWT_SECRET", "your-secret-key-change-in-production")
	if jwtSecret == "your-secret-key-change-in-production" {
		log.Warn("Using default JWT secret - change in production!", nil)
	}
	jwtManager := auth.NewJWTManager(
		jwtSecret,
		15*time.Minute,  // access token expiry
		7*24*time.Hour,  // refresh token expiry
		"logistics-platform",
		"logistics-api",
	)

	// Initialize Redis rate limiter (optional - falls back to in-memory)
	var rateLimiter ratelimit.Limiter
	redisHost := getEnv("REDIS_HOST", "")
	if redisHost != "" {
		redisPort := getEnv("REDIS_PORT", "6379")
		redisPassword := getEnv("REDIS_PASSWORD", "")
		redisDB := 0

		redisClient := redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
			Password: redisPassword,
			DB:       redisDB,
		})

		// Test Redis connection
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := redisClient.Ping(ctx).Err(); err != nil {
			log.Warn("Failed to connect to Redis, falling back to in-memory rate limiter", logger.Fields{"error": err})
			rateLimiter = ratelimit.NewInMemoryLimiter(100, time.Minute)
		} else {
			log.Info("Connected to Redis for rate limiting", logger.Fields{"addr": fmt.Sprintf("%s:%s", redisHost, redisPort)})
			rateLimiter = ratelimit.NewRedisLimiter(redisClient, 100, time.Minute, "inventory")
		}
	} else {
		log.Info("Redis not configured, using in-memory rate limiter", nil)
		rateLimiter = ratelimit.NewInMemoryLimiter(100, time.Minute)
	}

	// Initialize metrics
	httpMetrics := metrics.NewMetrics(serviceName)
	businessMetrics := metrics.NewBusinessMetrics(serviceName)

	// Initialize validator
	v := validator.New()

	// Initialize repository, service, and handler
	inventoryRepo := repository.NewInventoryRepository(db)
	inventoryService := service.NewInventoryService(inventoryRepo)
	httpHandler := handlers.NewHTTPHandler(inventoryService)

	// Setup router with explicit middleware stack
	router := gin.New()

	// Core middleware (applies to all routes)
	router.Use(middleware.RequestID())
	router.Use(middleware.Recovery(log))
	router.Use(log.GinMiddleware())
	router.Use(httpMetrics.Middleware())
	router.Use(middleware.CORS(middleware.DefaultCORSConfig()))
	router.Use(validator.Middleware(v))

	// Load server configuration
	serverConfig := config.LoadServerConfig()

	// Health and metrics endpoints (public, no auth)
	healthChecker := health.NewChecker(serviceName, version, db)
	router.GET("/health", healthChecker.Check)
	router.GET("/ready", healthChecker.Ready)
	router.GET("/live", healthChecker.Live)
	router.GET("/metrics", httpMetrics.Handler())

	// Public routes
	public := router.Group("/api/v1")
	{
		// Public health check
		public.GET("/inventory/health",
			ratelimit.Middleware(ratelimit.RateLimitConfig{
				Limiter:      rateLimiter,
				KeyExtractor: ratelimit.IPKeyExtractor,
			}),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok", "service": serviceName})
			},
		)

		// Check availability endpoint (public but rate limited)
		public.POST("/inventory/check-availability",
			ratelimit.Middleware(ratelimit.RateLimitConfig{
				Limiter:      rateLimiter,
				KeyExtractor: ratelimit.IPKeyExtractor,
			}),
			httpHandler.CheckAvailability,
		)
	}

	// Protected routes (require JWT authentication)
	protected := router.Group("/api/v1")
	protected.Use(auth.AuthMiddleware(jwtManager))
	{
		inventory := protected.Group("/inventory")
		{
			// Create item - managers and admins only
			inventory.POST("/items",
				ratelimit.Middleware(ratelimit.RateLimitConfig{
					Limiter:      rateLimiter,
					KeyExtractor: ratelimit.UserIDKeyExtractor,
				}),
				auth.RequireRole(auth.RoleManager, auth.RoleAdmin),
				httpHandler.CreateItem,
			)

			// Get item by ID
			inventory.GET("/items/:id",
				auth.RequireRole(auth.RoleUser, auth.RoleManager, auth.RoleAdmin),
				httpHandler.GetItem,
			)

			// List items - rate limited
			inventory.GET("/items",
				ratelimit.Middleware(ratelimit.RateLimitConfig{
					Limiter:      rateLimiter,
					KeyExtractor: ratelimit.UserIDKeyExtractor,
				}),
				auth.RequireRole(auth.RoleUser, auth.RoleManager, auth.RoleAdmin),
				httpHandler.ListItems,
			)

			// Update item - managers and admins only
			inventory.PUT("/items/:id",
				auth.RequireRole(auth.RoleManager, auth.RoleAdmin),
				httpHandler.UpdateItem,
			)

			// Update stock - managers and admins only
			inventory.PUT("/items/:product_id/stock",
				auth.RequireRole(auth.RoleManager, auth.RoleAdmin),
				func(c *gin.Context) {
					httpHandler.UpdateStock(c)
					businessMetrics.RecordInventoryUpdate()
				},
			)

			// Get low stock items - managers and admins only
			inventory.GET("/low-stock",
				auth.RequireRole(auth.RoleManager, auth.RoleAdmin),
				httpHandler.GetLowStockItems,
			)
		}

		// Admin-only routes
		admin := protected.Group("/admin")
		admin.Use(auth.RequireRole(auth.RoleAdmin))
		{
			// Get all inventory items (admin view)
			admin.GET("/inventory",
				ratelimit.Middleware(ratelimit.RateLimitConfig{
					Limiter:      rateLimiter,
					KeyExtractor: ratelimit.UserIDKeyExtractor,
				}),
				httpHandler.ListItems,
			)

			// Bulk inventory operations
			admin.POST("/inventory/bulk-update",
				ratelimit.Middleware(ratelimit.RateLimitConfig{
					Limiter:      rateLimiter,
					KeyExtractor: ratelimit.UserIDKeyExtractor,
				}),
				func(c *gin.Context) {
					// Placeholder for bulk operations
					c.JSON(http.StatusNotImplemented, gin.H{"message": "Bulk operations coming soon"})
				},
			)
		}
	}

	log.Info("Routes configured", logger.Fields{
		"public_routes":    2,
		"protected_routes": 6,
		"admin_routes":     2,
	})

	// Create HTTP server
	httpPort := getEnv("HTTP_PORT", "8082")
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", httpPort),
		Handler:           router,
		ReadTimeout:       serverConfig.ReadTimeout,
		WriteTimeout:      serverConfig.WriteTimeout,
		IdleTimeout:       serverConfig.IdleTimeout,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Info("Inventory Service ready", logger.Fields{
		"http_port": httpPort,
		"version":   version,
	})

	// Setup graceful shutdown
	shutdownManager := server.NewShutdownManager()

	// Add cleanup hooks
	shutdownManager.AddHook(func() error {
		log.Info("Closing database connections...", nil)
		return sqlDB.Close()
	})

	shutdownConfig := server.DefaultGracefulShutdownConfig()

	// Start server with graceful shutdown
	if err := server.RunWithGracefulShutdown(srv, shutdownConfig, shutdownManager.Shutdown); err != nil {
		log.Fatal("Server failed", logger.Fields{"error": err})
	}

	log.Info("Inventory Service stopped gracefully", nil)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
