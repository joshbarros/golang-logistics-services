package repository

import (
	"errors"

	"github.com/joshbarros/golang-logistics-services/services/notification-service/internal/models"
	"gorm.io/gorm"
)

type NotificationRepository interface {
	Create(notification *models.Notification) error
	GetByID(id string) (*models.Notification, error)
	List(page, pageSize int, recipient, notifType, status string) ([]*models.Notification, int, error)
	Update(notification *models.Notification) error
	UpdateStatus(id, status string) error
	Delete(id string) error
}

type PostgresNotificationRepository struct {
	db *gorm.DB
}

func NewPostgresNotificationRepository(db *gorm.DB) NotificationRepository {
	return &PostgresNotificationRepository{db: db}
}

func (r *PostgresNotificationRepository) Create(notification *models.Notification) error {
	return r.db.Create(notification).Error
}

func (r *PostgresNotificationRepository) GetByID(id string) (*models.Notification, error) {
	var notification models.Notification
	err := r.db.Where("id = ?", id).First(&notification).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("notification not found")
		}
		return nil, err
	}
	return &notification, nil
}

func (r *PostgresNotificationRepository) List(page, pageSize int, recipient, notifType, status string) ([]*models.Notification, int, error) {
	var notifications []*models.Notification
	var total int64

	query := r.db.Model(&models.Notification{})

	if recipient != "" {
		query = query.Where("recipient = ?", recipient)
	}
	if notifType != "" {
		query = query.Where("type = ?", notifType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&notifications).Error
	if err != nil {
		return nil, 0, err
	}

	return notifications, int(total), nil
}

func (r *PostgresNotificationRepository) Update(notification *models.Notification) error {
	return r.db.Save(notification).Error
}

func (r *PostgresNotificationRepository) UpdateStatus(id, status string) error {
	return r.db.Model(&models.Notification{}).Where("id = ?", id).Update("status", status).Error
}

func (r *PostgresNotificationRepository) Delete(id string) error {
	return r.db.Delete(&models.Notification{}, "id = ?", id).Error
}
