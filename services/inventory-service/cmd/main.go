package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joshbarros/golang-logistics-services/services/inventory-service/internal/database"
	"github.com/joshbarros/golang-logistics-services/services/inventory-service/internal/handlers"
	"github.com/joshbarros/golang-logistics-services/services/inventory-service/internal/repository"
	"github.com/joshbarros/golang-logistics-services/services/inventory-service/internal/service"
)

func main() {
	log.Println("Starting Inventory Service...")

	// Connect to database
	db, err := database.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize repository, service, and handler
	inventoryRepo := repository.NewInventoryRepository(db)
	inventoryService := service.NewInventoryService(inventoryRepo)
	httpHandler := handlers.NewHTTPHandler(inventoryService)

	// Initialize Gin router
	router := gin.Default()

	// Health check
	router.GET("/health", httpHandler.HealthCheck)

	// API routes
	v1 := router.Group("/api/v1")
	{
		inventory := v1.Group("/inventory")
		{
			inventory.POST("/items", httpHandler.CreateItem)
			inventory.GET("/items", httpHandler.ListItems)
			inventory.GET("/items/:id", httpHandler.GetItem)
			inventory.PUT("/items/:id", httpHandler.UpdateItem)
			inventory.PUT("/items/:product_id/stock", httpHandler.UpdateStock)
			inventory.GET("/low-stock", httpHandler.GetLowStockItems)
			inventory.POST("/check-availability", httpHandler.CheckAvailability)
		}
	}

	// Start HTTP server
	httpPort := getEnv("HTTP_PORT", "8082")
	log.Printf("Inventory Service HTTP server listening on port %s", httpPort)
	if err := router.Run(fmt.Sprintf(":%s", httpPort)); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
