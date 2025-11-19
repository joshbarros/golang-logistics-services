package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/joshbarros/golang-logistics-services/services/order-service/internal/models"
	"github.com/joshbarros/golang-logistics-services/services/order-service/internal/service"
)

// HTTPHandler handles HTTP requests
type HTTPHandler struct {
	orderService *service.OrderService
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(orderService *service.OrderService) *HTTPHandler {
	return &HTTPHandler{orderService: orderService}
}

// CreateOrderRequest represents the request to create an order
type CreateOrderRequest struct {
	CustomerID      string              `json:"customer_id" binding:"required"`
	CustomerName    string              `json:"customer_name" binding:"required"`
	CustomerEmail   string              `json:"customer_email" binding:"required,email"`
	ShippingAddress models.Address      `json:"shipping_address" binding:"required"`
	BillingAddress  models.Address      `json:"billing_address" binding:"required"`
	Items           []models.OrderItem  `json:"items" binding:"required,min=1"`
}

// CreateOrder handles POST /orders
func (h *HTTPHandler) CreateOrder(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.orderService.CreateOrder(
		req.CustomerID,
		req.CustomerName,
		req.CustomerEmail,
		req.ShippingAddress,
		req.BillingAddress,
		req.Items,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, order)
}

// GetOrder handles GET /orders/:id
func (h *HTTPHandler) GetOrder(c *gin.Context) {
	id := c.Param("id")

	order, err := h.orderService.GetOrder(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	c.JSON(http.StatusOK, order)
}

// ListOrders handles GET /orders
func (h *HTTPHandler) ListOrders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	customerID := c.Query("customer_id")
	status := c.Query("status")

	orders, total, err := h.orderService.ListOrders(page, pageSize, customerID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders":     orders,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
	})
}

// UpdateOrderRequest represents the request to update an order
type UpdateOrderRequest struct {
	Status          string          `json:"status"`
	ShippingAddress *models.Address `json:"shipping_address"`
}

// UpdateOrder handles PUT /orders/:id
func (h *HTTPHandler) UpdateOrder(c *gin.Context) {
	id := c.Param("id")

	var req UpdateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.orderService.UpdateOrder(id, req.Status, req.ShippingAddress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, order)
}

// CancelOrderRequest represents the request to cancel an order
type CancelOrderRequest struct {
	Reason string `json:"reason"`
}

// CancelOrder handles DELETE /orders/:id
func (h *HTTPHandler) CancelOrder(c *gin.Context) {
	id := c.Param("id")

	var req CancelOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.orderService.CancelOrder(id, req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, order)
}

// HealthCheck handles GET /health
func (h *HTTPHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "order-service"})
}
