package models

import (
	"time"

	"gorm.io/gorm"
)

// Notification types
const (
	TypeEmail = "EMAIL"
	TypeSMS   = "SMS"
	TypePush  = "PUSH"
)

// Notification statuses
const (
	StatusPending = "PENDING"
	StatusSent    = "SENT"
	StatusFailed  = "FAILED"
)

// Notification represents a notification to be sent
type Notification struct {
	ID           string         `json:"id" gorm:"primaryKey"`
	Recipient    string         `json:"recipient" gorm:"not null;index"`
	Type         string         `json:"type" gorm:"not null;index"`
	Subject      string         `json:"subject"`
	Message      string         `json:"message" gorm:"not null"`
	Status       string         `json:"status" gorm:"index;default:PENDING"`
	ErrorMessage string         `json:"error_message"`
	SentAt       *time.Time     `json:"sent_at"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName specifies the table name for Notification
func (Notification) TableName() string {
	return "notifications"
}
