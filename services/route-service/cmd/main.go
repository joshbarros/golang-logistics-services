package main

import (
	"fmt"
	"net/http"
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
	"github.com/joshbarros/golang-logistics-services/services/route-service/internal/handlers"
	"github.com/joshbarros/golang-logistics-services/services/route-service/internal/models"
	"github.com/joshbarros/golang-logistics-services/services/route-service/internal/repository"
	"github.com/joshbarros/golang-logistics-services/services/route-service/internal/service"
)

const (
	serviceName = "route-service"
)

func main() {
	// Initialize logger
	log := logger.New(serviceName)
	log.SetLevel(logger.FromEnv())
	log.Info("Starting service", logger.Fields{"service": serviceName})

	// Load configuration
	dbConfig := config.LoadDatabaseConfig("route")
	serverConfig := config.LoadServerConfig()
	version := config.GetServiceVersion()

	// Validate configuration
	if err := dbConfig.Validate(); err != nil {
		log.Fatal("Invalid configuration", logger.Fields{"error": err.Error()})
	}

	log.Info("Configuration loaded", logger.Fields{
		"db_host":     dbConfig.Host,
		"db_port":     dbConfig.Port,
		"db_name":     dbConfig.DBName,
		"http_port":   serverConfig.Port,
		"environment": config.GetEnvironment(),
		"version":     version,
	})

	// Connect to database
	log.Info("Connecting to database", nil)
	db, err := gorm.Open(postgres.Open(dbConfig.DSN()), &gorm.Config{
		Logger: nil,
	})
	if err != nil {
		log.Fatal("Failed to connect to database", logger.Fields{"error": err.Error()})
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get database instance", logger.Fields{"error": err.Error()})
	}
	health.ConfigureDBConnectionPool(sqlDB)

	log.Info("Database connection established", nil)

	// Run migrations
	log.Info("Running database migrations", nil)
	if err := db.AutoMigrate(&models.Route{}); err != nil {
		log.Fatal("Failed to run migrations", logger.Fields{"error": err.Error()})
	}
	log.Info("Database migrations completed", nil)

	// Initialize JWT manager
	jwtSecret := config.GetEnv("JWT_SECRET_KEY", "dev-secret-key-change-in-production")
	jwtManager := auth.NewJWTManager(
		jwtSecret,
		15*time.Minute,
		7*24*time.Hour,
		"logistics-platform",
		"logistics-api",
	)

	// Initialize Redis for rate limiting (optional)
	var rateLimiter ratelimit.Limiter
	redisHost := config.GetEnv("REDIS_HOST", "")
	if redisHost != "" {
		redisClient := redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", redisHost, config.GetEnv("REDIS_PORT", "6379")),
			Password: config.GetEnv("REDIS_PASSWORD", ""),
			DB:       config.GetEnvInt("REDIS_DB", 0),
		})
		rateLimiter = ratelimit.NewRedisLimiter(redisClient, 100, time.Minute)
		log.Info("Redis rate limiter enabled", nil)
	} else {
		rateLimiter = ratelimit.NewInMemoryLimiter(100, time.Minute)
		log.Warn("Using in-memory rate limiter (Redis not configured)", nil)
	}

	// Initialize application layers
	repo := repository.NewPostgresRouteRepository(db)
	svc := service.NewRouteService(repo)
	handler := handlers.NewHTTPHandler(svc)

	// Initialize metrics
	httpMetrics := metrics.NewMetrics(serviceName)
	businessMetrics := metrics.NewBusinessMetrics(serviceName)

	// Initialize validator
	v := validator.New()

	// Setup router
	if config.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()

	// Add middleware stack
	router.Use(middleware.RequestID())
	router.Use(middleware.Recovery(log))
	router.Use(log.GinMiddleware())
	router.Use(httpMetrics.Middleware())
	router.Use(middleware.CORS(middleware.DefaultCORSConfig()))
	router.Use(validator.Middleware(v))

	// Health checks
	healthChecker := health.NewChecker(serviceName, version, db)
	router.GET("/health", healthChecker.Check)
	router.GET("/ready", health.ReadinessHandler(serviceName))
	router.GET("/live", health.LivenessHandler(serviceName))

	// Metrics endpoint
	router.GET("/metrics", httpMetrics.Handler())

	// Public routes (no auth)
	public := router.Group("/api/v1")
	{
		// Calculate distance (public utility)
		public.POST("/routes/calculate-distance",
			ratelimit.Middleware(ratelimit.RateLimitConfig{
				Limiter:      rateLimiter,
				KeyExtractor: ratelimit.IPKeyExtractor,
			}),
			handler.CalculateDistance,
		)
	}

	// Protected routes (require authentication)
	protected := router.Group("/api/v1")
	protected.Use(auth.AuthMiddleware(jwtManager))
	{
		routes := protected.Group("/routes")
		{
			// Route optimization (expensive operation - stricter rate limit)
			routes.POST("/optimize",
				ratelimit.Middleware(ratelimit.RateLimitConfig{
					Limiter:      rateLimiter,
					KeyExtractor: ratelimit.UserIDKeyExtractor,
				}),
				func(c *gin.Context) {
					handler.OptimizeRoute(c)
					// Record business metric
					businessMetrics.RecordRouteOptimized()
				},
			)

			// Standard CRUD operations
			routes.GET("", handler.ListRoutes)
			routes.GET("/:id", handler.GetRoute)
			routes.POST("/:id/assign",
				auth.RequireRole(auth.RoleManager, auth.RoleAdmin),
				handler.AssignDriver,
			)
			routes.POST("/:id/start",
				auth.RequireRole(auth.RoleDriver, auth.RoleManager),
				handler.StartRoute,
			)
			routes.POST("/:id/complete",
				auth.RequireRole(auth.RoleDriver, auth.RoleManager),
				handler.CompleteRoute,
			)
		}
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", serverConfig.Port),
		Handler:      router,
		ReadTimeout:  serverConfig.ReadTimeout,
		WriteTimeout: serverConfig.WriteTimeout,
	}

	// Setup graceful shutdown
	shutdownManager := server.NewShutdownManager()

	// Add database cleanup hook
	shutdownManager.AddHook(func() error {
		log.Info("Closing database connections", nil)
		return sqlDB.Close()
	})

	log.Info("Server starting", logger.Fields{
		"port":    serverConfig.Port,
		"version": version,
	})

	// Run server with graceful shutdown
	shutdownConfig := server.DefaultShutdownConfig()
	shutdownConfig.Timeout = serverConfig.ShutdownTimeout

	if err := server.RunWithGracefulShutdown(srv, shutdownConfig, shutdownManager.Shutdown); err != nil {
		log.Fatal("Server error", logger.Fields{"error": err.Error()})
	}

	log.Info("Service stopped", nil)
}
