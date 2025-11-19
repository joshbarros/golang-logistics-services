package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

var (
	orderServiceURL        string
	shipmentServiceURL     string
	inventoryServiceURL    string
	routeServiceURL        string
	driverServiceURL       string
	notificationServiceURL string
)

func init() {
	orderServiceURL = getEnv("ORDER_SERVICE_URL", "http://order-service:8080")
	shipmentServiceURL = getEnv("SHIPMENT_SERVICE_URL", "http://shipment-service:8081")
	inventoryServiceURL = getEnv("INVENTORY_SERVICE_URL", "http://inventory-service:8082")
	routeServiceURL = getEnv("ROUTE_SERVICE_URL", "http://route-service:8083")
	driverServiceURL = getEnv("DRIVER_SERVICE_URL", "http://driver-service:8084")
	notificationServiceURL = getEnv("NOTIFICATION_SERVICE_URL", "http://notification-service:8085")
}

func main() {
	log.Println("Starting API Gateway...")

	router := gin.Default()

	// CORS middleware
	router.Use(corsMiddleware())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "api-gateway",
		})
	})

	// API version 1
	v1 := router.Group("/api/v1")
	{
		// Order routes
		orders := v1.Group("/orders")
		{
			orders.POST("", proxyRequest(orderServiceURL, "/api/v1/orders", "POST"))
			orders.GET("", proxyRequest(orderServiceURL, "/api/v1/orders", "GET"))
			orders.GET("/:id", proxyRequestWithParam(orderServiceURL, "/api/v1/orders", "GET"))
			orders.PUT("/:id", proxyRequestWithParam(orderServiceURL, "/api/v1/orders", "PUT"))
			orders.DELETE("/:id", proxyRequestWithParam(orderServiceURL, "/api/v1/orders", "DELETE"))
		}

		// Shipment routes
		shipments := v1.Group("/shipments")
		{
			shipments.POST("", proxyRequest(shipmentServiceURL, "/api/v1/shipments", "POST"))
			shipments.GET("", proxyRequest(shipmentServiceURL, "/api/v1/shipments", "GET"))
			shipments.GET("/:id", proxyRequestWithParam(shipmentServiceURL, "/api/v1/shipments", "GET"))
			shipments.GET("/track/:tracking_number", func(c *gin.Context) {
				trackingNumber := c.Param("tracking_number")
				url := fmt.Sprintf("%s/api/v1/shipments/track/%s", shipmentServiceURL, trackingNumber)
				forwardRequest(c, url, "GET", nil)
			})
			shipments.PUT("/:id/status", func(c *gin.Context) {
				id := c.Param("id")
				url := fmt.Sprintf("%s/api/v1/shipments/%s/status", shipmentServiceURL, id)
				var body map[string]interface{}
				c.ShouldBindJSON(&body)
				forwardRequest(c, url, "PUT", body)
			})
		}

		// Inventory routes
		inventory := v1.Group("/inventory")
		{
			inventory.POST("/items", proxyRequest(inventoryServiceURL, "/api/v1/inventory/items", "POST"))
			inventory.GET("/items", proxyRequest(inventoryServiceURL, "/api/v1/inventory/items", "GET"))
			inventory.GET("/items/:id", proxyRequestWithParam(inventoryServiceURL, "/api/v1/inventory/items", "GET"))
			inventory.PUT("/items/:id", proxyRequestWithParam(inventoryServiceURL, "/api/v1/inventory/items", "PUT"))
			inventory.PUT("/items/:product_id/stock", func(c *gin.Context) {
				productID := c.Param("product_id")
				url := fmt.Sprintf("%s/api/v1/inventory/items/%s/stock", inventoryServiceURL, productID)
				var body map[string]interface{}
				c.ShouldBindJSON(&body)
				forwardRequest(c, url, "PUT", body)
			})
			inventory.GET("/low-stock", proxyRequest(inventoryServiceURL, "/api/v1/inventory/low-stock", "GET"))
			inventory.POST("/check-availability", proxyRequest(inventoryServiceURL, "/api/v1/inventory/check-availability", "POST"))
		}

		// Route routes
		routes := v1.Group("/routes")
		{
			routes.POST("/optimize", proxyRequest(routeServiceURL, "/api/v1/routes/optimize", "POST"))
			routes.GET("/:id", proxyRequestWithParam(routeServiceURL, "/api/v1/routes", "GET"))
			routes.GET("", proxyRequest(routeServiceURL, "/api/v1/routes", "GET"))
		}

		// Driver routes
		drivers := v1.Group("/drivers")
		{
			drivers.POST("", proxyRequest(driverServiceURL, "/api/v1/drivers", "POST"))
			drivers.GET("", proxyRequest(driverServiceURL, "/api/v1/drivers", "GET"))
			drivers.GET("/:id", proxyRequestWithParam(driverServiceURL, "/api/v1/drivers", "GET"))
			drivers.PUT("/:id", proxyRequestWithParam(driverServiceURL, "/api/v1/drivers", "PUT"))
			drivers.POST("/:id/assign", func(c *gin.Context) {
				id := c.Param("id")
				url := fmt.Sprintf("%s/api/v1/drivers/%s/assign", driverServiceURL, id)
				var body map[string]interface{}
				c.ShouldBindJSON(&body)
				forwardRequest(c, url, "POST", body)
			})
			drivers.GET("/available", proxyRequest(driverServiceURL, "/api/v1/drivers/available", "GET"))
		}

		// Notification routes
		notifications := v1.Group("/notifications")
		{
			notifications.POST("/send", proxyRequest(notificationServiceURL, "/api/v1/notifications/send", "POST"))
			notifications.GET("/:id", proxyRequestWithParam(notificationServiceURL, "/api/v1/notifications", "GET"))
			notifications.GET("", proxyRequest(notificationServiceURL, "/api/v1/notifications", "GET"))
		}
	}

	httpPort := getEnv("HTTP_PORT", "8000")
	log.Printf("API Gateway listening on port %s", httpPort)
	router.Run(fmt.Sprintf(":%s", httpPort))
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func proxyRequest(serviceURL, path, method string) gin.HandlerFunc {
	return func(c *gin.Context) {
		url := fmt.Sprintf("%s%s", serviceURL, path)

		// Get query parameters
		if c.Request.URL.RawQuery != "" {
			url = fmt.Sprintf("%s?%s", url, c.Request.URL.RawQuery)
		}

		var body map[string]interface{}
		if method == "POST" || method == "PUT" {
			c.ShouldBindJSON(&body)
		}

		forwardRequest(c, url, method, body)
	}
}

func proxyRequestWithParam(serviceURL, path, method string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		url := fmt.Sprintf("%s%s/%s", serviceURL, path, id)

		var body map[string]interface{}
		if method == "PUT" || method == "DELETE" {
			c.ShouldBindJSON(&body)
		}

		forwardRequest(c, url, method, body)
	}
}

func forwardRequest(c *gin.Context, url, method string, body interface{}) {
	var req *http.Request
	var err error

	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req, err = http.NewRequest(method, url, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	// Copy headers
	for key, values := range c.Request.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error forwarding request to %s: %v", url, err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service unavailable"})
		return
	}
	defer resp.Body.Close()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	// Forward response
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), responseBody)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
