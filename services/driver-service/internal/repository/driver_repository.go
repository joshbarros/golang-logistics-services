package repository

import (
	"errors"

	"github.com/joshbarros/golang-logistics-services/services/driver-service/internal/models"
	"gorm.io/gorm"
)

// DriverRepository defines the interface for driver data operations
type DriverRepository interface {
	Create(driver *models.Driver) error
	GetByID(id string) (*models.Driver, error)
	GetByEmail(email string) (*models.Driver, error)
	List(page, pageSize int, status string) ([]*models.Driver, int, error)
	Update(driver *models.Driver) error
	Delete(id string) error
	GetAvailableDrivers() ([]*models.Driver, error)
	UpdateLocation(id string, lat, lng float64) error
	AssignRoute(driverID, routeID string) error
	UpdateStatus(id, status string) error
}

// PostgresDriverRepository implements DriverRepository using PostgreSQL
type PostgresDriverRepository struct {
	db *gorm.DB
}

// NewPostgresDriverRepository creates a new PostgreSQL driver repository
func NewPostgresDriverRepository(db *gorm.DB) DriverRepository {
	return &PostgresDriverRepository{db: db}
}

func (r *PostgresDriverRepository) Create(driver *models.Driver) error {
	return r.db.Create(driver).Error
}

func (r *PostgresDriverRepository) GetByID(id string) (*models.Driver, error) {
	var driver models.Driver
	err := r.db.Where("id = ?", id).First(&driver).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("driver not found")
		}
		return nil, err
	}
	return &driver, nil
}

func (r *PostgresDriverRepository) GetByEmail(email string) (*models.Driver, error) {
	var driver models.Driver
	err := r.db.Where("email = ?", email).First(&driver).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("driver not found")
		}
		return nil, err
	}
	return &driver, nil
}

func (r *PostgresDriverRepository) List(page, pageSize int, status string) ([]*models.Driver, int, error) {
	var drivers []*models.Driver
	var total int64

	query := r.db.Model(&models.Driver{})

	// Apply status filter if provided
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&drivers).Error
	if err != nil {
		return nil, 0, err
	}

	return drivers, int(total), nil
}

func (r *PostgresDriverRepository) Update(driver *models.Driver) error {
	return r.db.Save(driver).Error
}

func (r *PostgresDriverRepository) Delete(id string) error {
	return r.db.Delete(&models.Driver{}, "id = ?", id).Error
}

func (r *PostgresDriverRepository) GetAvailableDrivers() ([]*models.Driver, error) {
	var drivers []*models.Driver
	err := r.db.Where("status = ?", models.StatusAvailable).Find(&drivers).Error
	return drivers, err
}

func (r *PostgresDriverRepository) UpdateLocation(id string, lat, lng float64) error {
	return r.db.Model(&models.Driver{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"current_lat": lat,
			"current_lng": lng,
			"last_location_update": gorm.Expr("NOW()"),
		}).Error
}

func (r *PostgresDriverRepository) AssignRoute(driverID, routeID string) error {
	return r.db.Model(&models.Driver{}).
		Where("id = ?", driverID).
		Updates(map[string]interface{}{
			"current_route_id": routeID,
			"status":          models.StatusAssigned,
		}).Error
}

func (r *PostgresDriverRepository) UpdateStatus(id, status string) error {
	return r.db.Model(&models.Driver{}).
		Where("id = ?", id).
		Update("status", status).Error
}
