package models

import (
	"time"

	"gorm.io/gorm"
)

// Shipment represents a shipment in the system
type Shipment struct {
	ID                 string           `json:"id" gorm:"primaryKey"`
	OrderID            string           `json:"order_id" gorm:"index"`
	TrackingNumber     string           `json:"tracking_number" gorm:"uniqueIndex"`
	Carrier            string           `json:"carrier"`
	Origin             Address          `json:"origin" gorm:"embedded;embeddedPrefix:origin_"`
	Destination        Address          `json:"destination" gorm:"embedded;embeddedPrefix:destination_"`
	Status             string           `json:"status" gorm:"index"`
	Events             []TrackingEvent  `json:"events" gorm:"foreignKey:ShipmentID"`
	DriverID           string           `json:"driver_id"`
	EstimatedDelivery  time.Time        `json:"estimated_delivery"`
	ActualDelivery     *time.Time       `json:"actual_delivery,omitempty"`
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`
	DeletedAt          gorm.DeletedAt   `json:"-" gorm:"index"`
}

// TrackingEvent represents a tracking event
type TrackingEvent struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	ShipmentID  string    `json:"shipment_id" gorm:"index"`
	Location    string    `json:"location"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Timestamp   time.Time `json:"timestamp"`
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

// ShipmentStatus constants
const (
	StatusPending         = "PENDING"
	StatusPickedUp        = "PICKED_UP"
	StatusInTransit       = "IN_TRANSIT"
	StatusOutForDelivery  = "OUT_FOR_DELIVERY"
	StatusDelivered       = "DELIVERED"
	StatusFailed          = "FAILED"
	StatusReturned        = "RETURNED"
)
