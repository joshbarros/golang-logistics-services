package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/joshbarros/golang-logistics-services/pkg/config"
	"github.com/joshbarros/golang-logistics-services/pkg/health"
	"github.com/joshbarros/golang-logistics-services/pkg/logger"
	"github.com/joshbarros/golang-logistics-services/pkg/middleware"
	"github.com/joshbarros/golang-logistics-services/pkg/server"
	"github.com/joshbarros/golang-logistics-services/services/driver-service/internal/handlers"
	"github.com/joshbarros/golang-logistics-services/services/driver-service/internal/models"
	"github.com/joshbarros/golang-logistics-services/services/driver-service/internal/repository"
	"github.com/joshbarros/golang-logistics-services/services/driver-service/internal/service"
)

const (
	serviceName = "driver-service"
)

func main() {
	// Initialize logger
	log := logger.New(serviceName)
	log.SetLevel(logger.FromEnv())
	log.Info("Starting service", logger.Fields{"service": serviceName})

	// Load configuration
	dbConfig := config.LoadDatabaseConfig("driver")
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
		Logger: nil, // Disable gorm's default logger
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
	if err := db.AutoMigrate(&models.Driver{}); err != nil {
		log.Fatal("Failed to run migrations", logger.Fields{"error": err.Error()})
	}
	log.Info("Database migrations completed", nil)

	// Initialize application layers
	repo := repository.NewPostgresDriverRepository(db)
	svc := service.NewDriverService(repo)
	handler := handlers.NewHTTPHandler(svc)

	// Setup router
	if config.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New() // Create router without default middleware

	// Add middleware
	router.Use(middleware.RequestID())
	router.Use(middleware.Recovery(log))
	router.Use(log.GinMiddleware())
	router.Use(middleware.CORS(middleware.DefaultCORSConfig()))

	// Health checks
	healthChecker := health.NewChecker(serviceName, version, db)
	router.GET("/health", healthChecker.Check)
	router.GET("/ready", health.ReadinessHandler(serviceName))
	router.GET("/live", health.LivenessHandler(serviceName))

	// Setup application routes
	handlers.SetupRoutes(router, handler)

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
