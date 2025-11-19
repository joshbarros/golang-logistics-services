package models

import (
	"time"

	"gorm.io/gorm"
)

// InventoryItem represents an inventory item
type InventoryItem struct {
	ID          string         `json:"id" gorm:"primaryKey"`
	ProductID   string         `json:"product_id" gorm:"uniqueIndex"`
	ProductName string         `json:"product_name"`
	SKU         string         `json:"sku" gorm:"uniqueIndex"`
	Quantity    int            `json:"quantity"`
	ReorderLevel int           `json:"reorder_level"`
	Location    string         `json:"location"`
	Warehouse   string         `json:"warehouse"`
	UnitPrice   float64        `json:"unit_price"`
	Status      string         `json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// StockMovement represents stock movement history
type StockMovement struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	ProductID   string    `json:"product_id" gorm:"index"`
	Type        string    `json:"type"` // IN, OUT, ADJUSTMENT
	Quantity    int       `json:"quantity"`
	PreviousQty int       `json:"previous_qty"`
	NewQty      int       `json:"new_qty"`
	Reference   string    `json:"reference"` // Order ID, etc.
	Notes       string    `json:"notes"`
	CreatedAt   time.Time `json:"created_at"`
}

// Status constants
const (
	StatusAvailable    = "AVAILABLE"
	StatusLowStock     = "LOW_STOCK"
	StatusOutOfStock   = "OUT_OF_STOCK"
	StatusDiscontinued = "DISCONTINUED"
)

// Movement types
const (
	MovementIn         = "IN"
	MovementOut        = "OUT"
	MovementAdjustment = "ADJUSTMENT"
)
