package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Driver struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Phone       string    `json:"phone"`
	LicenseNo   string    `json:"license_no"`
	VehicleID   string    `json:"vehicle_id"`
	Status      string    `json:"status"` // AVAILABLE, ASSIGNED, OFF_DUTY
	CurrentLat  float64   `json:"current_lat"`
	CurrentLng  float64   `json:"current_lng"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateDriverRequest struct {
	Name      string `json:"name" binding:"required"`
	Email     string `json:"email" binding:"required,email"`
	Phone     string `json:"phone" binding:"required"`
	LicenseNo string `json:"license_no" binding:"required"`
	VehicleID string `json:"vehicle_id"`
}

type AssignDriverRequest struct {
	RouteID string `json:"route_id" binding:"required"`
}

var driversStore = make(map[string]*Driver)

func main() {
	log.Println("Starting Driver Service...")

	router := gin.Default()
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "driver-service"})
	})

	v1 := router.Group("/api/v1")
	{
		drivers := v1.Group("/drivers")
		{
			drivers.POST("", createDriver)
			drivers.GET("", listDrivers)
			drivers.GET("/:id", getDriver)
			drivers.PUT("/:id", updateDriver)
			drivers.POST("/:id/assign", assignDriver)
			drivers.GET("/available", getAvailableDrivers)
		}
	}

	httpPort := getEnv("HTTP_PORT", "8084")
	log.Printf("Driver Service listening on port %s", httpPort)
	router.Run(fmt.Sprintf(":%s", httpPort))
}

func createDriver(c *gin.Context) {
	var req CreateDriverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	driver := &Driver{
		ID:         uuid.New().String(),
		Name:       req.Name,
		Email:      req.Email,
		Phone:      req.Phone,
		LicenseNo:  req.LicenseNo,
		VehicleID:  req.VehicleID,
		Status:     "AVAILABLE",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	driversStore[driver.ID] = driver
	c.JSON(http.StatusCreated, driver)
}

func getDriver(c *gin.Context) {
	id := c.Param("id")
	driver, exists := driversStore[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Driver not found"})
		return
	}
	c.JSON(http.StatusOK, driver)
}

func listDrivers(c *gin.Context) {
	drivers := make([]*Driver, 0, len(driversStore))
	for _, driver := range driversStore {
		drivers = append(drivers, driver)
	}
	c.JSON(http.StatusOK, gin.H{"drivers": drivers, "total": len(drivers)})
}

func updateDriver(c *gin.Context) {
	id := c.Param("id")
	driver, exists := driversStore[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Driver not found"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if status, ok := updates["status"].(string); ok {
		driver.Status = status
	}
	driver.UpdatedAt = time.Now()

	c.JSON(http.StatusOK, driver)
}

func assignDriver(c *gin.Context) {
	id := c.Param("id")
	driver, exists := driversStore[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Driver not found"})
		return
	}

	var req AssignDriverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	driver.Status = "ASSIGNED"
	driver.UpdatedAt = time.Now()

	c.JSON(http.StatusOK, gin.H{
		"driver":   driver,
		"route_id": req.RouteID,
		"message":  "Driver assigned successfully",
	})
}

func getAvailableDrivers(c *gin.Context) {
	available := make([]*Driver, 0)
	for _, driver := range driversStore {
		if driver.Status == "AVAILABLE" {
			available = append(available, driver)
		}
	}
	c.JSON(http.StatusOK, gin.H{"drivers": available, "count": len(available)})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
