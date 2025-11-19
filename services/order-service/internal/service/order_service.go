package service

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/joshbarros/golang-logistics-services/services/order-service/internal/models"
	"github.com/joshbarros/golang-logistics-services/services/order-service/internal/repository"
)

// OrderService handles business logic for orders
type OrderService struct {
	repo *repository.OrderRepository
}

// NewOrderService creates a new order service
func NewOrderService(repo *repository.OrderRepository) *OrderService {
	return &OrderService{repo: repo}
}

// CreateOrder creates a new order
func (s *OrderService) CreateOrder(customerID, customerName, customerEmail string,
	shippingAddr, billingAddr models.Address, items []models.OrderItem) (*models.Order, error) {

	// Calculate total
	var totalAmount float64
	for i := range items {
		items[i].ID = uuid.New().String()
		items[i].TotalPrice = float64(items[i].Quantity) * items[i].UnitPrice
		totalAmount += items[i].TotalPrice
	}

	// Create order
	order := &models.Order{
		ID:              uuid.New().String(),
		CustomerID:      customerID,
		CustomerName:    customerName,
		CustomerEmail:   customerEmail,
		ShippingAddress: shippingAddr,
		BillingAddress:  billingAddr,
		Items:           items,
		TotalAmount:     totalAmount,
		Status:          models.StatusPending,
		TrackingNumber:  generateTrackingNumber(),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Set order ID for items
	for i := range order.Items {
		order.Items[i].OrderID = order.ID
	}

	err := s.repo.Create(order)
	if err != nil {
		return nil, err
	}

	return order, nil
}

// GetOrder retrieves an order by ID
func (s *OrderService) GetOrder(id string) (*models.Order, error) {
	return s.repo.GetByID(id)
}

// ListOrders retrieves orders with pagination
func (s *OrderService) ListOrders(page, pageSize int, customerID, status string) ([]models.Order, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	return s.repo.List(page, pageSize, customerID, status)
}

// UpdateOrder updates an order
func (s *OrderService) UpdateOrder(id, status string, shippingAddr *models.Address) (*models.Order, error) {
	order, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if status != "" {
		order.Status = status
	}
	if shippingAddr != nil {
		order.ShippingAddress = *shippingAddr
	}

	order.UpdatedAt = time.Now()

	err = s.repo.Update(order)
	if err != nil {
		return nil, err
	}

	return order, nil
}

// CancelOrder cancels an order
func (s *OrderService) CancelOrder(id, reason string) (*models.Order, error) {
	order, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if order.Status == models.StatusDelivered {
		return nil, fmt.Errorf("cannot cancel delivered order")
	}

	order.Status = models.StatusCancelled
	order.UpdatedAt = time.Now()

	err = s.repo.Update(order)
	if err != nil {
		return nil, err
	}

	return order, nil
}

// ValidateOrder validates an order
func (s *OrderService) ValidateOrder(id string) (bool, string, error) {
	order, err := s.repo.GetByID(id)
	if err != nil {
		return false, "Order not found", err
	}

	if len(order.Items) == 0 {
		return false, "Order has no items", nil
	}

	if order.TotalAmount <= 0 {
		return false, "Invalid order total", nil
	}

	return true, "Order is valid", nil
}

// generateTrackingNumber generates a unique tracking number
func generateTrackingNumber() string {
	return fmt.Sprintf("ORD-%d-%s", time.Now().Unix(), uuid.New().String()[:8])
}
