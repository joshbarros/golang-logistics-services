package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/joshbarros/golang-logistics-services/services/driver-service/internal/service"
)

// HTTPHandler handles HTTP requests for drivers
type HTTPHandler struct {
	service *service.DriverService
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(service *service.DriverService) *HTTPHandler {
	return &HTTPHandler{service: service}
}

// CreateDriverRequest represents the request to create a driver
type CreateDriverRequest struct {
	Name          string `json:"name" binding:"required"`
	Email         string `json:"email" binding:"required,email"`
	Phone         string `json:"phone" binding:"required"`
	LicenseNumber string `json:"license_number" binding:"required"`
	VehicleType   string `json:"vehicle_type"`
	VehiclePlate  string `json:"vehicle_plate"`
}

// UpdateDriverRequest represents the request to update a driver
type UpdateDriverRequest struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	Phone  string `json:"phone"`
	Status string `json:"status"`
}

// AssignDriverRequest represents the request to assign a driver
type AssignDriverRequest struct {
	RouteID string `json:"route_id" binding:"required"`
}

// UpdateLocationRequest represents the request to update driver location
type UpdateLocationRequest struct {
	Latitude  float64 `json:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
}

// CreateDriver handles POST /api/v1/drivers
func (h *HTTPHandler) CreateDriver(c *gin.Context) {
	var req CreateDriverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	driver, err := h.service.CreateDriver(
		req.Name,
		req.Email,
		req.Phone,
		req.LicenseNumber,
		req.VehicleType,
		req.VehiclePlate,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, driver)
}

// GetDriver handles GET /api/v1/drivers/:id
func (h *HTTPHandler) GetDriver(c *gin.Context) {
	id := c.Param("id")

	driver, err := h.service.GetDriver(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, driver)
}

// ListDrivers handles GET /api/v1/drivers
func (h *HTTPHandler) ListDrivers(c *gin.Context) {
	// Parse query parameters
	page := 1
	pageSize := 20
	status := c.Query("status")

	if p := c.Query("page"); p != "" {
		if val, err := parseIntParam(p); err == nil {
			page = val
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if val, err := parseIntParam(ps); err == nil {
			pageSize = val
		}
	}

	drivers, total, err := h.service.ListDrivers(page, pageSize, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"drivers":   drivers,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// UpdateDriver handles PUT /api/v1/drivers/:id
func (h *HTTPHandler) UpdateDriver(c *gin.Context) {
	id := c.Param("id")

	var req UpdateDriverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	driver, err := h.service.UpdateDriver(id, req.Name, req.Email, req.Phone, req.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, driver)
}

// DeleteDriver handles DELETE /api/v1/drivers/:id
func (h *HTTPHandler) DeleteDriver(c *gin.Context) {
	id := c.Param("id")

	err := h.service.DeleteDriver(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Driver deleted successfully"})
}

// AssignDriver handles POST /api/v1/drivers/:id/assign
func (h *HTTPHandler) AssignDriver(c *gin.Context) {
	id := c.Param("id")

	var req AssignDriverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	driver, err := h.service.AssignDriver(id, req.RouteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, driver)
}

// UnassignDriver handles POST /api/v1/drivers/:id/unassign
func (h *HTTPHandler) UnassignDriver(c *gin.Context) {
	id := c.Param("id")

	driver, err := h.service.UnassignDriver(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, driver)
}

// GetAvailableDrivers handles GET /api/v1/drivers/available
func (h *HTTPHandler) GetAvailableDrivers(c *gin.Context) {
	drivers, err := h.service.GetAvailableDrivers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"drivers": drivers})
}

// UpdateLocation handles POST /api/v1/drivers/:id/location
func (h *HTTPHandler) UpdateLocation(c *gin.Context) {
	id := c.Param("id")

	var req UpdateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	driver, err := h.service.UpdateLocation(id, req.Latitude, req.Longitude)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, driver)
}

// HealthCheck handles GET /health
func (h *HTTPHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "driver-service",
	})
}

// Helper function to parse int query parameters
func parseIntParam(s string) (int, error) {
	var val int
	_, err := fmt.Sscanf(s, "%d", &val)
	return val, err
}
