package repository

import (
	"github.com/joshbarros/golang-logistics-services/services/order-service/internal/models"
	"gorm.io/gorm"
)

// OrderRepository handles database operations for orders
type OrderRepository struct {
	db *gorm.DB
}

// NewOrderRepository creates a new order repository
func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// Create creates a new order
func (r *OrderRepository) Create(order *models.Order) error {
	return r.db.Create(order).Error
}

// GetByID retrieves an order by ID
func (r *OrderRepository) GetByID(id string) (*models.Order, error) {
	var order models.Order
	err := r.db.Preload("Items").First(&order, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// List retrieves orders with pagination and filters
func (r *OrderRepository) List(page, pageSize int, customerID, status string) ([]models.Order, int64, error) {
	var orders []models.Order
	var total int64

	query := r.db.Model(&models.Order{})

	if customerID != "" {
		query = query.Where("customer_id = ?", customerID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Count total
	query.Count(&total)

	// Get paginated results
	offset := (page - 1) * pageSize
	err := query.Preload("Items").
		Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&orders).Error

	return orders, total, err
}

// Update updates an order
func (r *OrderRepository) Update(order *models.Order) error {
	return r.db.Save(order).Error
}

// Delete soft deletes an order
func (r *OrderRepository) Delete(id string) error {
	return r.db.Delete(&models.Order{}, "id = ?", id).Error
}

// UpdateStatus updates order status
func (r *OrderRepository) UpdateStatus(id, status string) error {
	return r.db.Model(&models.Order{}).Where("id = ?", id).Update("status", status).Error
}
