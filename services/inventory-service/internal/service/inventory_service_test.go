package service

import (
	"errors"
	"testing"

	"github.com/joshbarros/golang-logistics-services/services/inventory-service/internal/models"
)

type MockInventoryRepository struct {
	createFunc          func(*models.InventoryItem) error
	getByIDFunc         func(string) (*models.InventoryItem, error)
	getByProductIDFunc  func(string) (*models.InventoryItem, error)
	listFunc            func(int, int, string, string) ([]models.InventoryItem, int64, error)
	updateFunc          func(*models.InventoryItem) error
	addMovementFunc     func(*models.StockMovement) error
	getLowStockItemsFunc func() ([]models.InventoryItem, error)
}

func (m *MockInventoryRepository) Create(item *models.InventoryItem) error {
	if m.createFunc != nil {
		return m.createFunc(item)
	}
	return nil
}

func (m *MockInventoryRepository) GetByID(id string) (*models.InventoryItem, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(id)
	}
	return &models.InventoryItem{ID: id}, nil
}

func (m *MockInventoryRepository) GetByProductID(productID string) (*models.InventoryItem, error) {
	if m.getByProductIDFunc != nil {
		return m.getByProductIDFunc(productID)
	}
	return &models.InventoryItem{ProductID: productID}, nil
}

func (m *MockInventoryRepository) List(page, pageSize int, warehouse, status string) ([]models.InventoryItem, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(page, pageSize, warehouse, status)
	}
	return []models.InventoryItem{}, 0, nil
}

func (m *MockInventoryRepository) Update(item *models.InventoryItem) error {
	if m.updateFunc != nil {
		return m.updateFunc(item)
	}
	return nil
}

func (m *MockInventoryRepository) AddMovement(movement *models.StockMovement) error {
	if m.addMovementFunc != nil {
		return m.addMovementFunc(movement)
	}
	return nil
}

func (m *MockInventoryRepository) GetLowStockItems() ([]models.InventoryItem, error) {
	if m.getLowStockItemsFunc != nil {
		return m.getLowStockItemsFunc()
	}
	return []models.InventoryItem{}, nil
}

func TestCreateItem(t *testing.T) {
	tests := []struct {
		name         string
		productID    string
		quantity     int
		reorderLevel int
		mockCreate   func(*models.InventoryItem) error
		wantStatus   string
		wantErr      bool
	}{
		{
			name:         "create with sufficient stock",
			productID:    "prod-1",
			quantity:     100,
			reorderLevel: 10,
			mockCreate:   func(item *models.InventoryItem) error { return nil },
			wantStatus:   models.StatusAvailable,
			wantErr:      false,
		},
		{
			name:         "create with low stock",
			productID:    "prod-2",
			quantity:     5,
			reorderLevel: 10,
			mockCreate:   func(item *models.InventoryItem) error { return nil },
			wantStatus:   models.StatusLowStock,
			wantErr:      false,
		},
		{
			name:         "create with no stock",
			productID:    "prod-3",
			quantity:     0,
			reorderLevel: 10,
			mockCreate:   func(item *models.InventoryItem) error { return nil },
			wantStatus:   models.StatusOutOfStock,
			wantErr:      false,
		},
		{
			name:         "repository error",
			productID:    "prod-4",
			quantity:     100,
			reorderLevel: 10,
			mockCreate:   func(item *models.InventoryItem) error { return errors.New("db error") },
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockInventoryRepository{
				createFunc:      tt.mockCreate,
				addMovementFunc: func(m *models.StockMovement) error { return nil },
			}
			svc := NewInventoryService(mockRepo)

			item, err := svc.CreateItem(tt.productID, "Product", "SKU123", "A1", "Warehouse1", tt.quantity, tt.reorderLevel, 10.0)

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr && item.Status != tt.wantStatus {
				t.Errorf("expected status %s, got %s", tt.wantStatus, item.Status)
			}
		})
	}
}

func TestUpdateStock(t *testing.T) {
	tests := []struct {
		name             string
		productID        string
		quantity         int
		movementType     string
		initialQty       int
		expectedFinalQty int
		wantErr          bool
		wantErrMsg       string
	}{
		{
			name:             "add stock",
			productID:        "prod-1",
			quantity:         50,
			movementType:     models.MovementIn,
			initialQty:       100,
			expectedFinalQty: 150,
			wantErr:          false,
		},
		{
			name:             "remove stock",
			productID:        "prod-1",
			quantity:         30,
			movementType:     models.MovementOut,
			initialQty:       100,
			expectedFinalQty: 70,
			wantErr:          false,
		},
		{
			name:             "insufficient stock",
			productID:        "prod-1",
			quantity:         150,
			movementType:     models.MovementOut,
			initialQty:       100,
			wantErr:          true,
			wantErrMsg:       "insufficient stock: available 100, requested 150",
		},
		{
			name:             "adjustment",
			productID:        "prod-1",
			quantity:         75,
			movementType:     models.MovementAdjustment,
			initialQty:       100,
			expectedFinalQty: 75,
			wantErr:          false,
		},
		{
			name:         "invalid movement type",
			productID:    "prod-1",
			quantity:     10,
			movementType: "INVALID",
			initialQty:   100,
			wantErr:      true,
			wantErrMsg:   "invalid movement type: INVALID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockInventoryRepository{
				getByProductIDFunc: func(pid string) (*models.InventoryItem, error) {
					return &models.InventoryItem{
						ProductID:    pid,
						Quantity:     tt.initialQty,
						ReorderLevel: 10,
					}, nil
				},
				updateFunc: func(item *models.InventoryItem) error {
					return nil
				},
				addMovementFunc: func(m *models.StockMovement) error {
					return nil
				},
			}
			svc := NewInventoryService(mockRepo)

			item, err := svc.UpdateStock(tt.productID, tt.quantity, tt.movementType, "ref-1", "test")

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				if tt.wantErrMsg != "" && err.Error() != tt.wantErrMsg {
					t.Errorf("expected error %q, got %q", tt.wantErrMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if item.Quantity != tt.expectedFinalQty {
					t.Errorf("expected final quantity %d, got %d", tt.expectedFinalQty, item.Quantity)
				}
			}
		})
	}
}

func TestCheckAvailability(t *testing.T) {
	tests := []struct {
		name           string
		productID      string
		requestedQty   int
		availableQty   int
		wantAvailable  bool
		wantErr        bool
	}{
		{
			name:          "sufficient stock",
			productID:     "prod-1",
			requestedQty:  50,
			availableQty:  100,
			wantAvailable: true,
			wantErr:       false,
		},
		{
			name:          "insufficient stock",
			productID:     "prod-1",
			requestedQty:  150,
			availableQty:  100,
			wantAvailable: false,
			wantErr:       false,
		},
		{
			name:          "exact stock",
			productID:     "prod-1",
			requestedQty:  100,
			availableQty:  100,
			wantAvailable: true,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockInventoryRepository{
				getByProductIDFunc: func(pid string) (*models.InventoryItem, error) {
					return &models.InventoryItem{
						ProductID: pid,
						Quantity:  tt.availableQty,
					}, nil
				},
			}
			svc := NewInventoryService(mockRepo)

			available, err := svc.CheckAvailability(tt.productID, tt.requestedQty)

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if available != tt.wantAvailable {
				t.Errorf("expected availability %v, got %v", tt.wantAvailable, available)
			}
		})
	}
}

func TestGetLowStockItems(t *testing.T) {
	mockRepo := &MockInventoryRepository{
		getLowStockItemsFunc: func() ([]models.InventoryItem, error) {
			return []models.InventoryItem{
				{ID: "item-1", Quantity: 5, ReorderLevel: 10},
				{ID: "item-2", Quantity: 0, ReorderLevel: 20},
			}, nil
		},
	}
	svc := NewInventoryService(mockRepo)

	items, err := svc.GetLowStockItems()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}
