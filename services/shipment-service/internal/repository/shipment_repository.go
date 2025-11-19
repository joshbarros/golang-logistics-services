package repository

import (
	"github.com/joshbarros/golang-logistics-services/services/shipment-service/internal/models"
	"gorm.io/gorm"
)

// ShipmentRepository handles database operations for shipments
type ShipmentRepository struct {
	db *gorm.DB
}

// NewShipmentRepository creates a new shipment repository
func NewShipmentRepository(db *gorm.DB) *ShipmentRepository {
	return &ShipmentRepository{db: db}
}

// Create creates a new shipment
func (r *ShipmentRepository) Create(shipment *models.Shipment) error {
	return r.db.Create(shipment).Error
}

// GetByID retrieves a shipment by ID
func (r *ShipmentRepository) GetByID(id string) (*models.Shipment, error) {
	var shipment models.Shipment
	err := r.db.Preload("Events").First(&shipment, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &shipment, nil
}

// GetByTrackingNumber retrieves a shipment by tracking number
func (r *ShipmentRepository) GetByTrackingNumber(trackingNumber string) (*models.Shipment, error) {
	var shipment models.Shipment
	err := r.db.Preload("Events").First(&shipment, "tracking_number = ?", trackingNumber).Error
	if err != nil {
		return nil, err
	}
	return &shipment, nil
}

// List retrieves shipments with pagination and filters
func (r *ShipmentRepository) List(page, pageSize int, orderID, status string) ([]models.Shipment, int64, error) {
	var shipments []models.Shipment
	var total int64

	query := r.db.Model(&models.Shipment{})

	if orderID != "" {
		query = query.Where("order_id = ?", orderID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Count total
	query.Count(&total)

	// Get paginated results
	offset := (page - 1) * pageSize
	err := query.Preload("Events").
		Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&shipments).Error

	return shipments, total, err
}

// Update updates a shipment
func (r *ShipmentRepository) Update(shipment *models.Shipment) error {
	return r.db.Save(shipment).Error
}

// AddTrackingEvent adds a tracking event to a shipment
func (r *ShipmentRepository) AddTrackingEvent(event *models.TrackingEvent) error {
	return r.db.Create(event).Error
}
