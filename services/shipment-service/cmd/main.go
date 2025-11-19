package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joshbarros/golang-logistics-services/services/shipment-service/internal/database"
	"github.com/joshbarros/golang-logistics-services/services/shipment-service/internal/handlers"
	"github.com/joshbarros/golang-logistics-services/services/shipment-service/internal/repository"
	"github.com/joshbarros/golang-logistics-services/services/shipment-service/internal/service"
)

func main() {
	log.Println("Starting Shipment Service...")

	// Connect to database
	db, err := database.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize repository, service, and handler
	shipmentRepo := repository.NewShipmentRepository(db)
	shipmentService := service.NewShipmentService(shipmentRepo)
	httpHandler := handlers.NewHTTPHandler(shipmentService)

	// Initialize Gin router
	router := gin.Default()

	// Health check
	router.GET("/health", httpHandler.HealthCheck)

	// API routes
	v1 := router.Group("/api/v1")
	{
		shipments := v1.Group("/shipments")
		{
			shipments.POST("", httpHandler.CreateShipment)
			shipments.GET("", httpHandler.ListShipments)
			shipments.GET("/:id", httpHandler.GetShipment)
			shipments.GET("/track/:tracking_number", httpHandler.TrackShipment)
			shipments.PUT("/:id/status", httpHandler.UpdateShipmentStatus)
		}
	}

	// Start HTTP server
	httpPort := getEnv("HTTP_PORT", "8081")
	log.Printf("Shipment Service HTTP server listening on port %s", httpPort)
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
