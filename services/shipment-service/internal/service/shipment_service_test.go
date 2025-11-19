package service

import (
	"errors"
	"testing"

	"github.com/joshbarros/golang-logistics-services/services/shipment-service/internal/models"
)

type MockShipmentRepository struct {
	createFunc               func(*models.Shipment) error
	getByIDFunc              func(string) (*models.Shipment, error)
	getByTrackingNumberFunc  func(string) (*models.Shipment, error)
	listFunc                 func(int, int, string, string) ([]models.Shipment, int64, error)
	updateFunc               func(*models.Shipment) error
	addTrackingEventFunc     func(*models.TrackingEvent) error
}

func (m *MockShipmentRepository) Create(shipment *models.Shipment) error {
	if m.createFunc != nil {
		return m.createFunc(shipment)
	}
	return nil
}

func (m *MockShipmentRepository) GetByID(id string) (*models.Shipment, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(id)
	}
	return &models.Shipment{ID: id}, nil
}

func (m *MockShipmentRepository) GetByTrackingNumber(trackingNumber string) (*models.Shipment, error) {
	if m.getByTrackingNumberFunc != nil {
		return m.getByTrackingNumberFunc(trackingNumber)
	}
	return &models.Shipment{TrackingNumber: trackingNumber}, nil
}

func (m *MockShipmentRepository) List(page, pageSize int, orderID, status string) ([]models.Shipment, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(page, pageSize, orderID, status)
	}
	return []models.Shipment{}, 0, nil
}

func (m *MockShipmentRepository) Update(shipment *models.Shipment) error {
	if m.updateFunc != nil {
		return m.updateFunc(shipment)
	}
	return nil
}

func (m *MockShipmentRepository) AddTrackingEvent(event *models.TrackingEvent) error {
	if m.addTrackingEventFunc != nil {
		return m.addTrackingEventFunc(event)
	}
	return nil
}

func TestCreateShipment(t *testing.T) {
	tests := []struct {
		name        string
		orderID     string
		carrier     string
		mockCreate  func(*models.Shipment) error
		wantErr     bool
	}{
		{
			name:    "successful creation",
			orderID: "order-123",
			carrier: "FedEx",
			mockCreate: func(s *models.Shipment) error {
				return nil
			},
			wantErr: false,
		},
		{
			name:    "repository error",
			orderID: "order-123",
			carrier: "UPS",
			mockCreate: func(s *models.Shipment) error {
				return errors.New("database error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockShipmentRepository{
				createFunc: tt.mockCreate,
			}
			svc := NewShipmentService(mockRepo)

			origin := models.Address{City: "Boston"}
			destination := models.Address{City: "New York"}

			shipment, err := svc.CreateShipment(tt.orderID, tt.carrier, origin, destination)

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr {
				if shipment.OrderID != tt.orderID {
					t.Errorf("expected orderID %s, got %s", tt.orderID, shipment.OrderID)
				}
				if shipment.Carrier != tt.carrier {
					t.Errorf("expected carrier %s, got %s", tt.carrier, shipment.Carrier)
				}
				if shipment.Status != models.StatusPending {
					t.Errorf("expected status %s, got %s", models.StatusPending, shipment.Status)
				}
				if len(shipment.Events) != 1 {
					t.Errorf("expected 1 tracking event, got %d", len(shipment.Events))
				}
			}
		})
	}
}

func TestTrackShipment(t *testing.T) {
	tests := []struct {
		name                    string
		trackingNumber          string
		mockGetByTrackingNumber func(string) (*models.Shipment, error)
		wantErr                 bool
	}{
		{
			name:           "successful tracking",
			trackingNumber: "TRACK-123",
			mockGetByTrackingNumber: func(tn string) (*models.Shipment, error) {
				return &models.Shipment{TrackingNumber: tn}, nil
			},
			wantErr: false,
		},
		{
			name:           "not found",
			trackingNumber: "TRACK-999",
			mockGetByTrackingNumber: func(tn string) (*models.Shipment, error) {
				return nil, errors.New("not found")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockShipmentRepository{
				getByTrackingNumberFunc: tt.mockGetByTrackingNumber,
			}
			svc := NewShipmentService(mockRepo)

			shipment, err := svc.TrackShipment(tt.trackingNumber)

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr && shipment.TrackingNumber != tt.trackingNumber {
				t.Errorf("expected tracking number %s, got %s", tt.trackingNumber, shipment.TrackingNumber)
			}
		})
	}
}

func TestUpdateShipmentStatus(t *testing.T) {
	tests := []struct {
		name              string
		shipmentID        string
		status            string
		mockGetByID       func(string) (*models.Shipment, error)
		mockAddEvent      func(*models.TrackingEvent) error
		mockUpdate        func(*models.Shipment) error
		wantDeliveryTime  bool
		wantErr           bool
	}{
		{
			name:       "update to in transit",
			shipmentID: "ship-123",
			status:     models.StatusInTransit,
			mockGetByID: func(id string) (*models.Shipment, error) {
				return &models.Shipment{ID: id, Status: models.StatusPending}, nil
			},
			mockAddEvent: func(e *models.TrackingEvent) error {
				return nil
			},
			mockUpdate: func(s *models.Shipment) error {
				return nil
			},
			wantDeliveryTime: false,
			wantErr:          false,
		},
		{
			name:       "update to delivered",
			shipmentID: "ship-123",
			status:     models.StatusDelivered,
			mockGetByID: func(id string) (*models.Shipment, error) {
				return &models.Shipment{ID: id, Status: models.StatusOutForDelivery}, nil
			},
			mockAddEvent: func(e *models.TrackingEvent) error {
				return nil
			},
			mockUpdate: func(s *models.Shipment) error {
				return nil
			},
			wantDeliveryTime: true,
			wantErr:          false,
		},
		{
			name:       "shipment not found",
			shipmentID: "ship-999",
			status:     models.StatusInTransit,
			mockGetByID: func(id string) (*models.Shipment, error) {
				return nil, errors.New("not found")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockShipmentRepository{
				getByIDFunc:          tt.mockGetByID,
				addTrackingEventFunc: tt.mockAddEvent,
				updateFunc:           tt.mockUpdate,
			}
			// Need to set getByIDFunc again for the reload
			reloadCalled := false
			originalGetByID := tt.mockGetByID
			mockRepo.getByIDFunc = func(id string) (*models.Shipment, error) {
				result, err := originalGetByID(id)
				if reloadCalled && result != nil {
					result.Status = tt.status
				}
				reloadCalled = true
				return result, err
			}

			svc := NewShipmentService(mockRepo)

			shipment, err := svc.UpdateShipmentStatus(tt.shipmentID, tt.status, "Boston", "Status updated")

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr {
				if tt.wantDeliveryTime && shipment.ActualDelivery == nil {
					t.Error("expected actual delivery time to be set")
				}
				if !tt.wantDeliveryTime && shipment.ActualDelivery != nil {
					t.Error("expected actual delivery time to be nil")
				}
			}
		})
	}
}

func TestListShipments(t *testing.T) {
	mockRepo := &MockShipmentRepository{
		listFunc: func(page, pageSize int, orderID, status string) ([]models.Shipment, int64, error) {
			if page < 1 {
				t.Error("page should be normalized to at least 1")
			}
			if pageSize < 1 || pageSize > 100 {
				t.Error("pageSize should be normalized to 1-100")
			}
			return []models.Shipment{{ID: "ship-1"}}, 1, nil
		},
	}
	svc := NewShipmentService(mockRepo)

	// Test with valid parameters
	shipments, total, err := svc.ListShipments(1, 10, "", "")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(shipments) != 1 {
		t.Errorf("expected 1 shipment, got %d", len(shipments))
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}

	// Test normalization
	svc.ListShipments(0, 200, "", "") // Should normalize to (1, 10)
}
