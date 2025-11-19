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
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/joshbarros/golang-logistics-services/pkg/auth"
	"github.com/joshbarros/golang-logistics-services/pkg/config"
	"github.com/joshbarros/golang-logistics-services/pkg/health"
	"github.com/joshbarros/golang-logistics-services/pkg/logger"
	"github.com/joshbarros/golang-logistics-services/pkg/metrics"
	"github.com/joshbarros/golang-logistics-services/pkg/middleware"
	"github.com/joshbarros/golang-logistics-services/pkg/ratelimit"
	"github.com/joshbarros/golang-logistics-services/pkg/server"
	"github.com/joshbarros/golang-logistics-services/pkg/validator"
	"github.com/joshbarros/golang-logistics-services/services/notification-service/internal/handlers"
	"github.com/joshbarros/golang-logistics-services/services/notification-service/internal/models"
	"github.com/joshbarros/golang-logistics-services/services/notification-service/internal/repository"
	"github.com/joshbarros/golang-logistics-services/services/notification-service/internal/service"
)

const (
	serviceName = "notification-service"
	version     = "2.0.0"
)

func main() {
	// Initialize structured logger
	log := logger.New(serviceName)
	log.SetLevel(logger.FromEnv())
	log.Info("Starting Notification Service...", logger.Fields{"version": version})

	// Load configuration
	dbConfig := config.LoadDatabaseConfig("notification")
	serverConfig := config.LoadServerConfig()

	// Database connection
	db, err := gorm.Open(postgres.Open(dbConfig.DSN()), &gorm.Config{})
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

	// Auto-migrate the schema
	log.Info("Running database migrations...", nil)
	err = db.AutoMigrate(&models.Notification{})
	if err != nil {
		log.Fatal("Failed to migrate database", logger.Fields{"error": err})
	}
	log.Info("Database migrations completed", nil)

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
			rateLimiter = ratelimit.NewRedisLimiter(redisClient, 100, time.Minute, "notification")
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

	// Initialize layers
	repo := repository.NewPostgresNotificationRepository(db)
	svc := service.NewNotificationService(repo)
	handler := handlers.NewHTTPHandler(svc)

	// Setup router with explicit middleware stack
	router := gin.New()

	// Core middleware (applies to all routes)
	router.Use(middleware.RequestID())
	router.Use(middleware.Recovery(log))
	router.Use(log.GinMiddleware())
	router.Use(httpMetrics.Middleware())
	router.Use(middleware.CORS(middleware.DefaultCORSConfig()))
	router.Use(validator.Middleware(v))

	// Health and metrics endpoints (public, no auth)
	healthChecker := health.NewChecker(serviceName, version, db)
	router.GET("/health", healthChecker.Check)
	router.GET("/ready", healthChecker.Ready)
	router.GET("/live", healthChecker.Live)
	router.GET("/metrics", httpMetrics.Handler())

	// Public routes (minimal or no auth, rate limited by IP)
	public := router.Group("/api/v1")
	{
		// Health check endpoint for notifications (public for monitoring)
		public.GET("/notifications/health",
			ratelimit.Middleware(ratelimit.RateLimitConfig{
				Limiter:      rateLimiter,
				KeyExtractor: ratelimit.IPKeyExtractor,
			}),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok", "service": serviceName})
			},
		)
	}

	// Protected routes (require JWT authentication)
	protected := router.Group("/api/v1")
	protected.Use(auth.AuthMiddleware(jwtManager))
	{
		notifications := protected.Group("/notifications")
		{
			// Send notification - rate limited per user to prevent spam
			notifications.POST("",
				ratelimit.Middleware(ratelimit.RateLimitConfig{
					Limiter:      rateLimiter,
					KeyExtractor: ratelimit.UserIDKeyExtractor,
				}),
				func(c *gin.Context) {
					handler.SendNotification(c)
					businessMetrics.RecordNotificationSent()
				},
			)

			// Get notification by ID
			notifications.GET("/:id", handler.GetNotification)

			// List user notifications (rate limited by user)
			notifications.GET("",
				ratelimit.Middleware(ratelimit.RateLimitConfig{
					Limiter:      rateLimiter,
					KeyExtractor: ratelimit.UserIDKeyExtractor,
				}),
				handler.ListNotifications,
			)

			// Update notification status (e.g., mark as read)
			notifications.PUT("/:id/status", handler.UpdateNotificationStatus)

			// Delete notification - requires manager or admin role
			notifications.DELETE("/:id",
				auth.RequireRole(auth.RoleManager, auth.RoleAdmin),
				handler.DeleteNotification,
			)
		}

		// Admin-only routes for notification management
		admin := protected.Group("/admin")
		admin.Use(auth.RequireRole(auth.RoleAdmin))
		{
			// Send bulk notifications
			admin.POST("/notifications/bulk",
				ratelimit.Middleware(ratelimit.RateLimitConfig{
					Limiter:      rateLimiter,
					KeyExtractor: ratelimit.UserIDKeyExtractor,
				}),
				func(c *gin.Context) {
					handler.SendBulkNotifications(c)
					businessMetrics.RecordNotificationSent()
				},
			)

			// Get all notifications (admin view)
			admin.GET("/notifications",
				ratelimit.Middleware(ratelimit.RateLimitConfig{
					Limiter:      rateLimiter,
					KeyExtractor: ratelimit.UserIDKeyExtractor,
				}),
				handler.ListAllNotifications,
			)
		}
	}

	log.Info("Routes configured", logger.Fields{
		"public_routes":    1,
		"protected_routes": 5,
		"admin_routes":     2,
	})

	// Create HTTP server
	httpPort := getEnv("HTTP_PORT", "8085")
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", httpPort),
		Handler:           router,
		ReadTimeout:       serverConfig.ReadTimeout,
		WriteTimeout:      serverConfig.WriteTimeout,
		IdleTimeout:       serverConfig.IdleTimeout,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Info("Notification Service ready", logger.Fields{
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

	log.Info("Notification Service stopped gracefully", nil)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
