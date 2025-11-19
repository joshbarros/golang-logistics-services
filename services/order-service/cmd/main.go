package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joshbarros/golang-logistics-services/services/order-service/internal/database"
	"github.com/joshbarros/golang-logistics-services/services/order-service/internal/handlers"
	"github.com/joshbarros/golang-logistics-services/services/order-service/internal/repository"
	"github.com/joshbarros/golang-logistics-services/services/order-service/internal/service"
)

func main() {
	log.Println("Starting Order Service...")

	// Connect to database
	db, err := database.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize repository, service, and handler
	orderRepo := repository.NewOrderRepository(db)
	orderService := service.NewOrderService(orderRepo)
	httpHandler := handlers.NewHTTPHandler(orderService)

	// Initialize Gin router
	router := gin.Default()

	// Health check
	router.GET("/health", httpHandler.HealthCheck)

	// API routes
	v1 := router.Group("/api/v1")
	{
		orders := v1.Group("/orders")
		{
			orders.POST("", httpHandler.CreateOrder)
			orders.GET("", httpHandler.ListOrders)
			orders.GET("/:id", httpHandler.GetOrder)
			orders.PUT("/:id", httpHandler.UpdateOrder)
			orders.DELETE("/:id", httpHandler.CancelOrder)
		}
	}

	// Start HTTP server
	httpPort := getEnv("HTTP_PORT", "8080")
	log.Printf("Order Service HTTP server listening on port %s", httpPort)
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
