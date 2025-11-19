package service

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/joshbarros/golang-logistics-services/services/shipment-service/internal/models"
	"github.com/joshbarros/golang-logistics-services/services/shipment-service/internal/repository"
)

// ShipmentService handles business logic for shipments
type ShipmentService struct {
	repo *repository.ShipmentRepository
}

// NewShipmentService creates a new shipment service
func NewShipmentService(repo *repository.ShipmentRepository) *ShipmentService {
	return &ShipmentService{repo: repo}
}

// CreateShipment creates a new shipment
func (s *ShipmentService) CreateShipment(orderID, carrier string, origin, destination models.Address) (*models.Shipment, error) {
	trackingNumber := generateTrackingNumber()
	estimatedDelivery := time.Now().Add(5 * 24 * time.Hour) // 5 days

	shipment := &models.Shipment{
		ID:                uuid.New().String(),
		OrderID:           orderID,
		TrackingNumber:    trackingNumber,
		Carrier:           carrier,
		Origin:            origin,
		Destination:       destination,
		Status:            models.StatusPending,
		EstimatedDelivery: estimatedDelivery,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Create initial tracking event
	event := models.TrackingEvent{
		ID:          uuid.New().String(),
		ShipmentID:  shipment.ID,
		Location:    origin.City,
		Description: "Shipment created",
		Status:      models.StatusPending,
		Timestamp:   time.Now(),
	}
	shipment.Events = []models.TrackingEvent{event}

	err := s.repo.Create(shipment)
	if err != nil {
		return nil, err
	}

	return shipment, nil
}

// GetShipment retrieves a shipment by ID
func (s *ShipmentService) GetShipment(id string) (*models.Shipment, error) {
	return s.repo.GetByID(id)
}

// TrackShipment retrieves a shipment by tracking number
func (s *ShipmentService) TrackShipment(trackingNumber string) (*models.Shipment, error) {
	return s.repo.GetByTrackingNumber(trackingNumber)
}

// ListShipments retrieves shipments with pagination
func (s *ShipmentService) ListShipments(page, pageSize int, orderID, status string) ([]models.Shipment, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	return s.repo.List(page, pageSize, orderID, status)
}

// UpdateShipmentStatus updates shipment status and adds tracking event
func (s *ShipmentService) UpdateShipmentStatus(id, status, location, description string) (*models.Shipment, error) {
	shipment, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	shipment.Status = status
	shipment.UpdatedAt = time.Now()

	// If delivered, set actual delivery time
	if status == models.StatusDelivered {
		now := time.Now()
		shipment.ActualDelivery = &now
	}

	// Add tracking event
	event := models.TrackingEvent{
		ID:          uuid.New().String(),
		ShipmentID:  shipment.ID,
		Location:    location,
		Description: description,
		Status:      status,
		Timestamp:   time.Now(),
	}

	if err := s.repo.AddTrackingEvent(&event); err != nil {
		return nil, err
	}

	if err := s.repo.Update(shipment); err != nil {
		return nil, err
	}

	// Reload to get all events
	return s.repo.GetByID(id)
}

// AssignDriver assigns a driver to a shipment
func (s *ShipmentService) AssignDriver(id, driverID string) error {
	shipment, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	shipment.DriverID = driverID
	shipment.UpdatedAt = time.Now()

	return s.repo.Update(shipment)
}

// generateTrackingNumber generates a unique tracking number
func generateTrackingNumber() string {
	return fmt.Sprintf("SHIP-%d-%s", time.Now().Unix(), uuid.New().String()[:8])
}
