package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Route statuses
const (
	StatusPlanned    = "PLANNED"
	StatusInProgress = "IN_PROGRESS"
	StatusCompleted  = "COMPLETED"
	StatusCancelled  = "CANCELLED"
)

// Location represents a geographical coordinate
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Waypoint represents a stop on a route
type Waypoint struct {
	ID        string   `json:"id"`
	RouteID   string   `json:"route_id"`
	Address   string   `json:"address"`
	Location  Location `json:"location" gorm:"embedded;embeddedPrefix:location_"`
	Sequence  int      `json:"sequence"`
	Priority  int      `json:"priority"`
	Completed bool     `json:"completed"`
}

// Waypoints is a custom type for JSON serialization
type Waypoints []Waypoint

// Scan implements sql.Scanner
func (w *Waypoints) Scan(value interface{}) error {
	if value == nil {
		*w = Waypoints{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, w)
}

// Value implements driver.Valuer
func (w Waypoints) Value() (driver.Value, error) {
	if len(w) == 0 {
		return nil, nil
	}
	return json.Marshal(w)
}

// Route represents a delivery route
type Route struct {
	ID                string         `json:"id" gorm:"primaryKey"`
	Name              string         `json:"name"`
	DriverID          string         `json:"driver_id" gorm:"index"`
	VehicleID         string         `json:"vehicle_id"`
	StartLocation     Location       `json:"start_location" gorm:"embedded;embeddedPrefix:start_"`
	EndLocation       Location       `json:"end_location" gorm:"embedded;embeddedPrefix:end_"`
	Waypoints         Waypoints      `json:"waypoints" gorm:"type:jsonb"`
	TotalDistance     float64        `json:"total_distance"`
	EstimatedDuration int            `json:"estimated_duration_minutes"`
	Status            string         `json:"status" gorm:"index;default:PLANNED"`
	Optimized         bool           `json:"optimized" gorm:"default:false"`
	StartedAt         *time.Time     `json:"started_at"`
	CompletedAt       *time.Time     `json:"completed_at"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName specifies the table name for Route
func (Route) TableName() string {
	return "routes"
}
