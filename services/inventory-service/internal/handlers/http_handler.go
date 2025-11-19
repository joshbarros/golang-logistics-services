package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/joshbarros/golang-logistics-services/services/inventory-service/internal/service"
)

// HTTPHandler handles HTTP requests
type HTTPHandler struct {
	inventoryService *service.InventoryService
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(inventoryService *service.InventoryService) *HTTPHandler {
	return &HTTPHandler{inventoryService: inventoryService}
}

type CreateItemRequest struct {
	ProductID    string  `json:"product_id" binding:"required"`
	ProductName  string  `json:"product_name" binding:"required"`
	SKU          string  `json:"sku" binding:"required"`
	Location     string  `json:"location" binding:"required"`
	Warehouse    string  `json:"warehouse" binding:"required"`
	Quantity     int     `json:"quantity" binding:"required"`
	ReorderLevel int     `json:"reorder_level" binding:"required"`
	UnitPrice    float64 `json:"unit_price" binding:"required"`
}

func (h *HTTPHandler) CreateItem(c *gin.Context) {
	var req CreateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, err := h.inventoryService.CreateItem(
		req.ProductID, req.ProductName, req.SKU, req.Location, req.Warehouse,
		req.Quantity, req.ReorderLevel, req.UnitPrice,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, item)
}

func (h *HTTPHandler) GetItem(c *gin.Context) {
	id := c.Param("id")
	item, err := h.inventoryService.GetItem(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *HTTPHandler) ListItems(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	warehouse := c.Query("warehouse")
	status := c.Query("status")

	items, total, err := h.inventoryService.ListItems(page, pageSize, warehouse, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *HTTPHandler) UpdateItem(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Not implemented"})
}

type UpdateStockRequest struct {
	Quantity     int    `json:"quantity" binding:"required"`
	MovementType string `json:"movement_type" binding:"required"`
	Reference    string `json:"reference"`
	Notes        string `json:"notes"`
}

func (h *HTTPHandler) UpdateStock(c *gin.Context) {
	productID := c.Param("product_id")
	var req UpdateStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, err := h.inventoryService.UpdateStock(productID, req.Quantity, req.MovementType, req.Reference, req.Notes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, item)
}

func (h *HTTPHandler) GetLowStockItems(c *gin.Context) {
	items, err := h.inventoryService.GetLowStockItems()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

type CheckAvailabilityRequest struct {
	ProductID string `json:"product_id" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required"`
}

func (h *HTTPHandler) CheckAvailability(c *gin.Context) {
	var req CheckAvailabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	available, err := h.inventoryService.CheckAvailability(req.ProductID, req.Quantity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"available": available})
}

func (h *HTTPHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "inventory-service"})
}
