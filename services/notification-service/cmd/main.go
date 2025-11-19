package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/joshbarros/golang-logistics-services/services/notification-service/internal/handlers"
	"github.com/joshbarros/golang-logistics-services/services/notification-service/internal/models"
	"github.com/joshbarros/golang-logistics-services/services/notification-service/internal/repository"
	"github.com/joshbarros/golang-logistics-services/services/notification-service/internal/service"
)

func main() {
	log.Println("Starting Notification Service...")

	// Database connection
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "notification_db")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		dbHost, dbUser, dbPassword, dbName, dbPort)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate the schema
	log.Println("Running database migrations...")
	err = db.AutoMigrate(&models.Notification{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}
	log.Println("Database migrations completed")

	// Initialize layers
	repo := repository.NewPostgresNotificationRepository(db)
	svc := service.NewNotificationService(repo)
	handler := handlers.NewHTTPHandler(svc)

	// Setup router
	router := gin.Default()
	handlers.SetupRoutes(router, handler)

	// Start server
	httpPort := getEnv("HTTP_PORT", "8085")
	log.Printf("Notification Service listening on port %s", httpPort)
	if err := router.Run(fmt.Sprintf(":%s", httpPort)); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
