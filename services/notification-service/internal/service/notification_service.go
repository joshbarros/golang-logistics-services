package service

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/joshbarros/golang-logistics-services/services/notification-service/internal/models"
	"github.com/joshbarros/golang-logistics-services/services/notification-service/internal/repository"
)

type NotificationService struct {
	repo repository.NotificationRepository
}

func NewNotificationService(repo repository.NotificationRepository) *NotificationService {
	return &NotificationService{repo: repo}
}

// SendNotification creates and sends a notification
func (s *NotificationService) SendNotification(recipient, notifType, subject, message string) (*models.Notification, error) {
	// Validate inputs
	if recipient == "" || message == "" {
		return nil, errors.New("recipient and message are required")
	}

	if !isValidType(notifType) {
		return nil, fmt.Errorf("invalid notification type: %s (must be EMAIL, SMS, or PUSH)", notifType)
	}

	notification := &models.Notification{
		ID:        uuid.New().String(),
		Recipient: recipient,
		Type:      notifType,
		Subject:   subject,
		Message:   message,
		Status:    models.StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := s.repo.Create(notification)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	// Send notification asynchronously
	go s.sendAsync(notification.ID, notifType, recipient, subject, message)

	return notification, nil
}

// sendAsync simulates sending a notification (in production, this would call actual email/SMS/push services)
func (s *NotificationService) sendAsync(notificationID, notifType, recipient, subject, message string) {
	// Simulate network delay
	time.Sleep(500 * time.Millisecond)

	log.Printf("Sending %s notification to %s: %s", notifType, recipient, subject)

	// In production, this would call:
	// - SendGrid/AWS SES for EMAIL
	// - Twilio for SMS
	// - Firebase Cloud Messaging for PUSH

	// Simulate successful send
	now := time.Now()
	notification, err := s.repo.GetByID(notificationID)
	if err != nil {
		log.Printf("Failed to get notification %s: %v", notificationID, err)
		return
	}

	notification.Status = models.StatusSent
	notification.SentAt = &now
	notification.UpdatedAt = now

	err = s.repo.Update(notification)
	if err != nil {
		log.Printf("Failed to update notification %s status: %v", notificationID, err)
	}
}

func (s *NotificationService) GetNotification(id string) (*models.Notification, error) {
	if id == "" {
		return nil, errors.New("notification ID is required")
	}
	return s.repo.GetByID(id)
}

func (s *NotificationService) ListNotifications(page, pageSize int, recipient, notifType, status string) ([]*models.Notification, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.List(page, pageSize, recipient, notifType, status)
}

func (s *NotificationService) UpdateStatus(id, status string) (*models.Notification, error) {
	if !isValidStatus(status) {
		return nil, fmt.Errorf("invalid status: %s", status)
	}

	err := s.repo.UpdateStatus(id, status)
	if err != nil {
		return nil, fmt.Errorf("failed to update status: %w", err)
	}

	return s.repo.GetByID(id)
}

func (s *NotificationService) DeleteNotification(id string) error {
	return s.repo.Delete(id)
}

func isValidType(notifType string) bool {
	validTypes := map[string]bool{
		models.TypeEmail: true,
		models.TypeSMS:   true,
		models.TypePush:  true,
	}
	return validTypes[notifType]
}

func isValidStatus(status string) bool {
	validStatuses := map[string]bool{
		models.StatusPending: true,
		models.StatusSent:    true,
		models.StatusFailed:  true,
	}
	return validStatuses[status]
}
