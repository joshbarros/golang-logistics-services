package service

import (
	"errors"
	"testing"

	"github.com/joshbarros/golang-logistics-services/services/order-service/internal/models"
)

// MockOrderRepository is a mock implementation of the repository
type MockOrderRepository struct {
	createFunc       func(*models.Order) error
	getByIDFunc      func(string) (*models.Order, error)
	listFunc         func(int, int, string, string) ([]models.Order, int64, error)
	updateFunc       func(*models.Order) error
	deleteFunc       func(string) error
	updateStatusFunc func(string, string) error
}

func (m *MockOrderRepository) Create(order *models.Order) error {
	if m.createFunc != nil {
		return m.createFunc(order)
	}
	return nil
}

func (m *MockOrderRepository) GetByID(id string) (*models.Order, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(id)
	}
	return &models.Order{ID: id}, nil
}

func (m *MockOrderRepository) List(page, pageSize int, customerID, status string) ([]models.Order, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(page, pageSize, customerID, status)
	}
	return []models.Order{}, 0, nil
}

func (m *MockOrderRepository) Update(order *models.Order) error {
	if m.updateFunc != nil {
		return m.updateFunc(order)
	}
	return nil
}

func (m *MockOrderRepository) Delete(id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(id)
	}
	return nil
}

func (m *MockOrderRepository) UpdateStatus(id, status string) error {
	if m.updateStatusFunc != nil {
		return m.updateStatusFunc(id, status)
	}
	return nil
}

func TestCreateOrder(t *testing.T) {
	tests := []struct {
		name          string
		customerID    string
		customerName  string
		customerEmail string
		items         []models.OrderItem
		mockCreate    func(*models.Order) error
		wantErr       bool
	}{
		{
			name:          "successful order creation",
			customerID:    "cust-123",
			customerName:  "John Doe",
			customerEmail: "john@example.com",
			items: []models.OrderItem{
				{ProductID: "prod-1", ProductName: "Widget", Quantity: 2, UnitPrice: 10.0},
			},
			mockCreate: func(o *models.Order) error {
				return nil
			},
			wantErr: false,
		},
		{
			name:          "repository error",
			customerID:    "cust-123",
			customerName:  "John Doe",
			customerEmail: "john@example.com",
			items: []models.OrderItem{
				{ProductID: "prod-1", ProductName: "Widget", Quantity: 2, UnitPrice: 10.0},
			},
			mockCreate: func(o *models.Order) error {
				return errors.New("database error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockOrderRepository{
				createFunc: tt.mockCreate,
			}
			svc := NewOrderService(mockRepo)

			shippingAddr := models.Address{Street: "123 Main St", City: "Boston"}
			billingAddr := models.Address{Street: "123 Main St", City: "Boston"}

			order, err := svc.CreateOrder(tt.customerID, tt.customerName, tt.customerEmail, shippingAddr, billingAddr, tt.items)

			if tt.wantErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr {
				if order == nil {
					t.Error("expected order but got nil")
				}
				if order.CustomerID != tt.customerID {
					t.Errorf("expected customerID %s, got %s", tt.customerID, order.CustomerID)
				}
				if order.Status != models.StatusPending {
					t.Errorf("expected status %s, got %s", models.StatusPending, order.Status)
				}
				if len(order.Items) != len(tt.items) {
					t.Errorf("expected %d items, got %d", len(tt.items), len(order.Items))
				}
				if order.TotalAmount != 20.0 {
					t.Errorf("expected total 20.0, got %f", order.TotalAmount)
				}
			}
		})
	}
}

func TestGetOrder(t *testing.T) {
	tests := []struct {
		name        string
		orderID     string
		mockGetByID func(string) (*models.Order, error)
		wantErr     bool
	}{
		{
			name:    "successful get",
			orderID: "order-123",
			mockGetByID: func(id string) (*models.Order, error) {
				return &models.Order{ID: id, CustomerID: "cust-1"}, nil
			},
			wantErr: false,
		},
		{
			name:    "order not found",
			orderID: "order-999",
			mockGetByID: func(id string) (*models.Order, error) {
				return nil, errors.New("not found")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockOrderRepository{
				getByIDFunc: tt.mockGetByID,
			}
			svc := NewOrderService(mockRepo)

			order, err := svc.GetOrder(tt.orderID)

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr && order.ID != tt.orderID {
				t.Errorf("expected order ID %s, got %s", tt.orderID, order.ID)
			}
		})
	}
}

func TestListOrders(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		pageSize   int
		customerID string
		status     string
		mockList   func(int, int, string, string) ([]models.Order, int64, error)
		wantErr    bool
	}{
		{
			name:       "successful list",
			page:       1,
			pageSize:   10,
			customerID: "cust-1",
			status:     "",
			mockList: func(page, pageSize int, customerID, status string) ([]models.Order, int64, error) {
				return []models.Order{{ID: "order-1"}, {ID: "order-2"}}, 2, nil
			},
			wantErr: false,
		},
		{
			name:       "normalize invalid page",
			page:       0,
			pageSize:   10,
			customerID: "",
			status:     "",
			mockList: func(page, pageSize int, customerID, status string) ([]models.Order, int64, error) {
				if page != 1 {
					t.Errorf("expected page to be normalized to 1, got %d", page)
				}
				return []models.Order{}, 0, nil
			},
			wantErr: false,
		},
		{
			name:       "normalize invalid page size",
			page:       1,
			pageSize:   200,
			customerID: "",
			status:     "",
			mockList: func(page, pageSize int, customerID, status string) ([]models.Order, int64, error) {
				if pageSize != 10 {
					t.Errorf("expected pageSize to be normalized to 10, got %d", pageSize)
				}
				return []models.Order{}, 0, nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockOrderRepository{
				listFunc: tt.mockList,
			}
			svc := NewOrderService(mockRepo)

			_, _, err := svc.ListOrders(tt.page, tt.pageSize, tt.customerID, tt.status)

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCancelOrder(t *testing.T) {
	tests := []struct {
		name        string
		orderID     string
		reason      string
		mockGetByID func(string) (*models.Order, error)
		mockUpdate  func(*models.Order) error
		wantErr     bool
		wantErrMsg  string
	}{
		{
			name:    "successful cancellation",
			orderID: "order-123",
			reason:  "Customer request",
			mockGetByID: func(id string) (*models.Order, error) {
				return &models.Order{ID: id, Status: models.StatusPending}, nil
			},
			mockUpdate: func(o *models.Order) error {
				return nil
			},
			wantErr: false,
		},
		{
			name:    "cannot cancel delivered order",
			orderID: "order-123",
			reason:  "Too late",
			mockGetByID: func(id string) (*models.Order, error) {
				return &models.Order{ID: id, Status: models.StatusDelivered}, nil
			},
			mockUpdate: func(o *models.Order) error {
				return nil
			},
			wantErr:    true,
			wantErrMsg: "cannot cancel delivered order",
		},
		{
			name:    "order not found",
			orderID: "order-999",
			reason:  "Not needed",
			mockGetByID: func(id string) (*models.Order, error) {
				return nil, errors.New("not found")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockOrderRepository{
				getByIDFunc: tt.mockGetByID,
				updateFunc:  tt.mockUpdate,
			}
			svc := NewOrderService(mockRepo)

			order, err := svc.CancelOrder(tt.orderID, tt.reason)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				if tt.wantErrMsg != "" && err.Error() != tt.wantErrMsg {
					t.Errorf("expected error message %q, got %q", tt.wantErrMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if order.Status != models.StatusCancelled {
					t.Errorf("expected status %s, got %s", models.StatusCancelled, order.Status)
				}
			}
		})
	}
}

func TestValidateOrder(t *testing.T) {
	tests := []struct {
		name        string
		orderID     string
		mockGetByID func(string) (*models.Order, error)
		wantValid   bool
		wantMessage string
	}{
		{
			name:    "valid order",
			orderID: "order-123",
			mockGetByID: func(id string) (*models.Order, error) {
				return &models.Order{
					ID:          id,
					Items:       []models.OrderItem{{ProductID: "p1", Quantity: 1, UnitPrice: 10}},
					TotalAmount: 10.0,
				}, nil
			},
			wantValid:   true,
			wantMessage: "Order is valid",
		},
		{
			name:    "order with no items",
			orderID: "order-123",
			mockGetByID: func(id string) (*models.Order, error) {
				return &models.Order{
					ID:          id,
					Items:       []models.OrderItem{},
					TotalAmount: 10.0,
				}, nil
			},
			wantValid:   false,
			wantMessage: "Order has no items",
		},
		{
			name:    "order with invalid total",
			orderID: "order-123",
			mockGetByID: func(id string) (*models.Order, error) {
				return &models.Order{
					ID:          id,
					Items:       []models.OrderItem{{ProductID: "p1"}},
					TotalAmount: 0,
				}, nil
			},
			wantValid:   false,
			wantMessage: "Invalid order total",
		},
		{
			name:    "order not found",
			orderID: "order-999",
			mockGetByID: func(id string) (*models.Order, error) {
				return nil, errors.New("not found")
			},
			wantValid:   false,
			wantMessage: "Order not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockOrderRepository{
				getByIDFunc: tt.mockGetByID,
			}
			svc := NewOrderService(mockRepo)

			valid, message, _ := svc.ValidateOrder(tt.orderID)

			if valid != tt.wantValid {
				t.Errorf("expected valid=%v, got %v", tt.wantValid, valid)
			}
			if message != tt.wantMessage {
				t.Errorf("expected message %q, got %q", tt.wantMessage, message)
			}
		})
	}
}
