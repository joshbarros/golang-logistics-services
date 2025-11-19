package models

import (
	"time"

	"gorm.io/gorm"
)

// Driver statuses
const (
	StatusAvailable  = "AVAILABLE"
	StatusAssigned   = "ASSIGNED"
	StatusDelivering = "DELIVERING"
	StatusOffDuty    = "OFF_DUTY"
)

// Driver represents a delivery driver
type Driver struct {
	ID            string         `json:"id" gorm:"primaryKey"`
	Name          string         `json:"name" gorm:"not null"`
	Email         string         `json:"email" gorm:"uniqueIndex;not null"`
	Phone         string         `json:"phone" gorm:"not null"`
	LicenseNumber string         `json:"license_number" gorm:"uniqueIndex;not null"`
	VehicleType   string         `json:"vehicle_type"`
	VehiclePlate  string         `json:"vehicle_plate"`
	Status        string         `json:"status" gorm:"index;default:AVAILABLE"`
	CurrentLat    float64        `json:"current_lat"`
	CurrentLng    float64        `json:"current_lng"`
	CurrentRouteID string        `json:"current_route_id" gorm:"index"`
	LastLocationUpdate time.Time `json:"last_location_update"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName specifies the table name for Driver
func (Driver) TableName() string {
	return "drivers"
}

// Location represents a geographical location
type Location struct {
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Timestamp time.Time `json:"timestamp"`
}
