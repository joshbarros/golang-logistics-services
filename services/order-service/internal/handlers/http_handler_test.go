package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/joshbarros/golang-logistics-services/services/order-service/internal/models"
)

// MockOrderService is a mock implementation of the order service
type MockOrderService struct {
	createOrderFunc   func(string, string, string, models.Address, models.Address, []models.OrderItem) (*models.Order, error)
	getOrderFunc      func(string) (*models.Order, error)
	listOrdersFunc    func(int, int, string, string) ([]models.Order, int64, error)
	updateOrderFunc   func(string, string, *models.Address) (*models.Order, error)
	cancelOrderFunc   func(string, string) (*models.Order, error)
	validateOrderFunc func(string) (bool, string, error)
}

func (m *MockOrderService) CreateOrder(customerID, customerName, customerEmail string, shippingAddr, billingAddr models.Address, items []models.OrderItem) (*models.Order, error) {
	if m.createOrderFunc != nil {
		return m.createOrderFunc(customerID, customerName, customerEmail, shippingAddr, billingAddr, items)
	}
	return &models.Order{ID: "order-123"}, nil
}

func (m *MockOrderService) GetOrder(id string) (*models.Order, error) {
	if m.getOrderFunc != nil {
		return m.getOrderFunc(id)
	}
	return &models.Order{ID: id}, nil
}

func (m *MockOrderService) ListOrders(page, pageSize int, customerID, status string) ([]models.Order, int64, error) {
	if m.listOrdersFunc != nil {
		return m.listOrdersFunc(page, pageSize, customerID, status)
	}
	return []models.Order{}, 0, nil
}

func (m *MockOrderService) UpdateOrder(id, status string, shippingAddr *models.Address) (*models.Order, error) {
	if m.updateOrderFunc != nil {
		return m.updateOrderFunc(id, status, shippingAddr)
	}
	return &models.Order{ID: id}, nil
}

func (m *MockOrderService) CancelOrder(id, reason string) (*models.Order, error) {
	if m.cancelOrderFunc != nil {
		return m.cancelOrderFunc(id, reason)
	}
	return &models.Order{ID: id, Status: models.StatusCancelled}, nil
}

func (m *MockOrderService) ValidateOrder(id string) (bool, string, error) {
	if m.validateOrderFunc != nil {
		return m.validateOrderFunc(id)
	}
	return true, "Valid", nil
}

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestCreateOrder(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockCreate     func(string, string, string, models.Address, models.Address, []models.OrderItem) (*models.Order, error)
		expectedStatus int
	}{
		{
			name: "successful creation",
			requestBody: CreateOrderRequest{
				CustomerID:    "cust-1",
				CustomerName:  "John Doe",
				CustomerEmail: "john@example.com",
				ShippingAddress: models.Address{
					Street: "123 Main St",
					City:   "Boston",
				},
				BillingAddress: models.Address{
					Street: "123 Main St",
					City:   "Boston",
				},
				Items: []models.OrderItem{
					{ProductID: "p1", ProductName: "Widget", Quantity: 1, UnitPrice: 10},
				},
			},
			mockCreate: func(cid, cn, ce string, sa, ba models.Address, items []models.OrderItem) (*models.Order, error) {
				return &models.Order{ID: "order-123", CustomerID: cid}, nil
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "invalid request body",
			requestBody: map[string]string{
				"invalid": "data",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "service error",
			requestBody: CreateOrderRequest{
				CustomerID:    "cust-1",
				CustomerName:  "John Doe",
				CustomerEmail: "john@example.com",
				ShippingAddress: models.Address{
					Street: "123 Main St",
					City:   "Boston",
				},
				BillingAddress: models.Address{
					Street: "123 Main St",
					City:   "Boston",
				},
				Items: []models.OrderItem{
					{ProductID: "p1", ProductName: "Widget", Quantity: 1, UnitPrice: 10},
				},
			},
			mockCreate: func(cid, cn, ce string, sa, ba models.Address, items []models.OrderItem) (*models.Order, error) {
				return nil, errors.New("service error")
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockOrderService{
				createOrderFunc: tt.mockCreate,
			}
			handler := NewHTTPHandler(mockService)
			router := setupRouter()
			router.POST("/orders", handler.CreateOrder)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/orders", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestGetOrder(t *testing.T) {
	tests := []struct {
		name           string
		orderID        string
		mockGet        func(string) (*models.Order, error)
		expectedStatus int
	}{
		{
			name:    "successful get",
			orderID: "order-123",
			mockGet: func(id string) (*models.Order, error) {
				return &models.Order{ID: id}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "order not found",
			orderID: "order-999",
			mockGet: func(id string) (*models.Order, error) {
				return nil, errors.New("not found")
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockOrderService{
				getOrderFunc: tt.mockGet,
			}
			handler := NewHTTPHandler(mockService)
			router := setupRouter()
			router.GET("/orders/:id", handler.GetOrder)

			req, _ := http.NewRequest("GET", "/orders/"+tt.orderID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestListOrders(t *testing.T) {
	mockService := &MockOrderService{
		listOrdersFunc: func(page, pageSize int, customerID, status string) ([]models.Order, int64, error) {
			return []models.Order{{ID: "order-1"}}, 1, nil
		},
	}
	handler := NewHTTPHandler(mockService)
	router := setupRouter()
	router.GET("/orders", handler.ListOrders)

	req, _ := http.NewRequest("GET", "/orders?page=1&page_size=10", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestCancelOrder(t *testing.T) {
	tests := []struct {
		name           string
		orderID        string
		requestBody    CancelOrderRequest
		mockCancel     func(string, string) (*models.Order, error)
		expectedStatus int
	}{
		{
			name:    "successful cancellation",
			orderID: "order-123",
			requestBody: CancelOrderRequest{
				Reason: "Customer request",
			},
			mockCancel: func(id, reason string) (*models.Order, error) {
				return &models.Order{ID: id, Status: models.StatusCancelled}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid request",
			orderID:        "order-123",
			requestBody:    CancelOrderRequest{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:    "service error",
			orderID: "order-123",
			requestBody: CancelOrderRequest{
				Reason: "Test",
			},
			mockCancel: func(id, reason string) (*models.Order, error) {
				return nil, errors.New("cannot cancel")
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockOrderService{
				cancelOrderFunc: tt.mockCancel,
			}
			handler := NewHTTPHandler(mockService)
			router := setupRouter()
			router.DELETE("/orders/:id", handler.CancelOrder)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("DELETE", "/orders/"+tt.orderID, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHealthCheck(t *testing.T) {
	mockService := &MockOrderService{}
	handler := NewHTTPHandler(mockService)
	router := setupRouter()
	router.GET("/health", handler.HealthCheck)

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got '%s'", response["status"])
	}
	if response["service"] != "order-service" {
		t.Errorf("expected service 'order-service', got '%s'", response["service"])
	}
}
