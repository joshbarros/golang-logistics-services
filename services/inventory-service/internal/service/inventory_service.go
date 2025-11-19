package service

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/joshbarros/golang-logistics-services/services/inventory-service/internal/models"
	"github.com/joshbarros/golang-logistics-services/services/inventory-service/internal/repository"
)

// InventoryService handles business logic for inventory
type InventoryService struct {
	repo *repository.InventoryRepository
}

// NewInventoryService creates a new inventory service
func NewInventoryService(repo *repository.InventoryRepository) *InventoryService {
	return &InventoryService{repo: repo}
}

// CreateItem creates a new inventory item
func (s *InventoryService) CreateItem(productID, productName, sku, location, warehouse string, quantity, reorderLevel int, unitPrice float64) (*models.InventoryItem, error) {
	status := s.determineStatus(quantity, reorderLevel)

	item := &models.InventoryItem{
		ID:           uuid.New().String(),
		ProductID:    productID,
		ProductName:  productName,
		SKU:          sku,
		Quantity:     quantity,
		ReorderLevel: reorderLevel,
		Location:     location,
		Warehouse:    warehouse,
		UnitPrice:    unitPrice,
		Status:       status,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err := s.repo.Create(item)
	if err != nil {
		return nil, err
	}

	// Record initial stock movement
	movement := &models.StockMovement{
		ID:          uuid.New().String(),
		ProductID:   productID,
		Type:        models.MovementIn,
		Quantity:    quantity,
		PreviousQty: 0,
		NewQty:      quantity,
		Reference:   "INITIAL_STOCK",
		Notes:       "Initial inventory",
		CreatedAt:   time.Now(),
	}
	s.repo.AddMovement(movement)

	return item, nil
}

// GetItem retrieves an inventory item by ID
func (s *InventoryService) GetItem(id string) (*models.InventoryItem, error) {
	return s.repo.GetByID(id)
}

// GetItemByProductID retrieves an inventory item by product ID
func (s *InventoryService) GetItemByProductID(productID string) (*models.InventoryItem, error) {
	return s.repo.GetByProductID(productID)
}

// ListItems retrieves inventory items with pagination
func (s *InventoryService) ListItems(page, pageSize int, warehouse, status string) ([]models.InventoryItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	return s.repo.List(page, pageSize, warehouse, status)
}

// UpdateStock updates stock quantity
func (s *InventoryService) UpdateStock(productID string, quantity int, movementType, reference, notes string) (*models.InventoryItem, error) {
	item, err := s.repo.GetByProductID(productID)
	if err != nil {
		return nil, err
	}

	previousQty := item.Quantity
	var newQty int

	switch movementType {
	case models.MovementIn:
		newQty = item.Quantity + quantity
	case models.MovementOut:
		if item.Quantity < quantity {
			return nil, fmt.Errorf("insufficient stock: available %d, requested %d", item.Quantity, quantity)
		}
		newQty = item.Quantity - quantity
	case models.MovementAdjustment:
		newQty = quantity
	default:
		return nil, fmt.Errorf("invalid movement type: %s", movementType)
	}

	item.Quantity = newQty
	item.Status = s.determineStatus(newQty, item.ReorderLevel)
	item.UpdatedAt = time.Now()

	err = s.repo.Update(item)
	if err != nil {
		return nil, err
	}

	// Record stock movement
	movement := &models.StockMovement{
		ID:          uuid.New().String(),
		ProductID:   productID,
		Type:        movementType,
		Quantity:    quantity,
		PreviousQty: previousQty,
		NewQty:      newQty,
		Reference:   reference,
		Notes:       notes,
		CreatedAt:   time.Now(),
	}
	s.repo.AddMovement(movement)

	return item, nil
}

// CheckAvailability checks if sufficient quantity is available
func (s *InventoryService) CheckAvailability(productID string, quantity int) (bool, error) {
	item, err := s.repo.GetByProductID(productID)
	if err != nil {
		return false, err
	}

	return item.Quantity >= quantity, nil
}

// GetLowStockItems retrieves items with low stock
func (s *InventoryService) GetLowStockItems() ([]models.InventoryItem, error) {
	return s.repo.GetLowStockItems()
}

// determineStatus determines item status based on quantity
func (s *InventoryService) determineStatus(quantity, reorderLevel int) string {
	if quantity == 0 {
		return models.StatusOutOfStock
	} else if quantity <= reorderLevel {
		return models.StatusLowStock
	}
	return models.StatusAvailable
}
