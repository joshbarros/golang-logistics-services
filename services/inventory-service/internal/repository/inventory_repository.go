package repository

import (
	"github.com/joshbarros/golang-logistics-services/services/inventory-service/internal/models"
	"gorm.io/gorm"
)

// InventoryRepository handles database operations for inventory
type InventoryRepository struct {
	db *gorm.DB
}

// NewInventoryRepository creates a new inventory repository
func NewInventoryRepository(db *gorm.DB) *InventoryRepository {
	return &InventoryRepository{db: db}
}

// Create creates a new inventory item
func (r *InventoryRepository) Create(item *models.InventoryItem) error {
	return r.db.Create(item).Error
}

// GetByID retrieves an inventory item by ID
func (r *InventoryRepository) GetByID(id string) (*models.InventoryItem, error) {
	var item models.InventoryItem
	err := r.db.First(&item, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// GetByProductID retrieves an inventory item by product ID
func (r *InventoryRepository) GetByProductID(productID string) (*models.InventoryItem, error) {
	var item models.InventoryItem
	err := r.db.First(&item, "product_id = ?", productID).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// List retrieves inventory items with pagination and filters
func (r *InventoryRepository) List(page, pageSize int, warehouse, status string) ([]models.InventoryItem, int64, error) {
	var items []models.InventoryItem
	var total int64

	query := r.db.Model(&models.InventoryItem{})

	if warehouse != "" {
		query = query.Where("warehouse = ?", warehouse)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Count total
	query.Count(&total)

	// Get paginated results
	offset := (page - 1) * pageSize
	err := query.Offset(offset).
		Limit(pageSize).
		Order("product_name ASC").
		Find(&items).Error

	return items, total, err
}

// Update updates an inventory item
func (r *InventoryRepository) Update(item *models.InventoryItem) error {
	return r.db.Save(item).Error
}

// AddMovement records a stock movement
func (r *InventoryRepository) AddMovement(movement *models.StockMovement) error {
	return r.db.Create(movement).Error
}

// GetLowStockItems retrieves items with low stock
func (r *InventoryRepository) GetLowStockItems() ([]models.InventoryItem, error) {
	var items []models.InventoryItem
	err := r.db.Where("quantity <= reorder_level AND status != ?", models.StatusDiscontinued).
		Find(&items).Error
	return items, err
}
