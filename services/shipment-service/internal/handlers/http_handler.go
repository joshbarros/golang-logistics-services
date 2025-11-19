package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/joshbarros/golang-logistics-services/services/shipment-service/internal/models"
	"github.com/joshbarros/golang-logistics-services/services/shipment-service/internal/service"
)

// HTTPHandler handles HTTP requests
type HTTPHandler struct {
	shipmentService *service.ShipmentService
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(shipmentService *service.ShipmentService) *HTTPHandler {
	return &HTTPHandler{shipmentService: shipmentService}
}

// CreateShipmentRequest represents the request to create a shipment
type CreateShipmentRequest struct {
	OrderID     string         `json:"order_id" binding:"required"`
	Carrier     string         `json:"carrier" binding:"required"`
	Origin      models.Address `json:"origin" binding:"required"`
	Destination models.Address `json:"destination" binding:"required"`
}

// CreateShipment handles POST /shipments
func (h *HTTPHandler) CreateShipment(c *gin.Context) {
	var req CreateShipmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	shipment, err := h.shipmentService.CreateShipment(req.OrderID, req.Carrier, req.Origin, req.Destination)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, shipment)
}

// GetShipment handles GET /shipments/:id
func (h *HTTPHandler) GetShipment(c *gin.Context) {
	id := c.Param("id")

	shipment, err := h.shipmentService.GetShipment(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Shipment not found"})
		return
	}

	c.JSON(http.StatusOK, shipment)
}

// TrackShipment handles GET /shipments/track/:tracking_number
func (h *HTTPHandler) TrackShipment(c *gin.Context) {
	trackingNumber := c.Param("tracking_number")

	shipment, err := h.shipmentService.TrackShipment(trackingNumber)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Shipment not found"})
		return
	}

	c.JSON(http.StatusOK, shipment)
}

// ListShipments handles GET /shipments
func (h *HTTPHandler) ListShipments(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	orderID := c.Query("order_id")
	status := c.Query("status")

	shipments, total, err := h.shipmentService.ListShipments(page, pageSize, orderID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"shipments":  shipments,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
	})
}

// UpdateStatusRequest represents the request to update shipment status
type UpdateStatusRequest struct {
	Status      string `json:"status" binding:"required"`
	Location    string `json:"location" binding:"required"`
	Description string `json:"description"`
}

// UpdateShipmentStatus handles PUT /shipments/:id/status
func (h *HTTPHandler) UpdateShipmentStatus(c *gin.Context) {
	id := c.Param("id")

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	shipment, err := h.shipmentService.UpdateShipmentStatus(id, req.Status, req.Location, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, shipment)
}

// HealthCheck handles GET /health
func (h *HTTPHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "shipment-service"})
}
