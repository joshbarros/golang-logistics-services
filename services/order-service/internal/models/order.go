package models

import (
	"time"

	"gorm.io/gorm"
)

// Order represents an order in the system
type Order struct {
	ID              string         `json:"id" gorm:"primaryKey"`
	CustomerID      string         `json:"customer_id" gorm:"index"`
	CustomerName    string         `json:"customer_name"`
	CustomerEmail   string         `json:"customer_email"`
	ShippingAddress Address        `json:"shipping_address" gorm:"embedded;embeddedPrefix:shipping_"`
	BillingAddress  Address        `json:"billing_address" gorm:"embedded;embeddedPrefix:billing_"`
	Items           []OrderItem    `json:"items" gorm:"foreignKey:OrderID"`
	TotalAmount     float64        `json:"total_amount"`
	Status          string         `json:"status" gorm:"index"`
	TrackingNumber  string         `json:"tracking_number"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

// OrderItem represents an item in an order
type OrderItem struct {
	ID          string  `json:"id" gorm:"primaryKey"`
	OrderID     string  `json:"order_id" gorm:"index"`
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	TotalPrice  float64 `json:"total_price"`
}

// Address represents a physical address
type Address struct {
	Street    string  `json:"street"`
	City      string  `json:"city"`
	State     string  `json:"state"`
	ZipCode   string  `json:"zip_code"`
	Country   string  `json:"country"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// OrderStatus constants
const (
	StatusPending    = "PENDING"
	StatusConfirmed  = "CONFIRMED"
	StatusProcessing = "PROCESSING"
	StatusShipped    = "SHIPPED"
	StatusDelivered  = "DELIVERED"
	StatusCancelled  = "CANCELLED"
)
