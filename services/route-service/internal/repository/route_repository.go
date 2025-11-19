package repository

import (
	"errors"

	"github.com/joshbarros/golang-logistics-services/services/route-service/internal/models"
	"gorm.io/gorm"
)

type RouteRepository interface {
	Create(route *models.Route) error
	GetByID(id string) (*models.Route, error)
	List(page, pageSize int, driverID, status string) ([]*models.Route, int, error)
	Update(route *models.Route) error
	Delete(id string) error
	UpdateStatus(id, status string) error
}

type PostgresRouteRepository struct {
	db *gorm.DB
}

func NewPostgresRouteRepository(db *gorm.DB) RouteRepository {
	return &PostgresRouteRepository{db: db}
}

func (r *PostgresRouteRepository) Create(route *models.Route) error {
	return r.db.Create(route).Error
}

func (r *PostgresRouteRepository) GetByID(id string) (*models.Route, error) {
	var route models.Route
	err := r.db.Where("id = ?", id).First(&route).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("route not found")
		}
		return nil, err
	}
	return &route, nil
}

func (r *PostgresRouteRepository) List(page, pageSize int, driverID, status string) ([]*models.Route, int, error) {
	var routes []*models.Route
	var total int64

	query := r.db.Model(&models.Route{})

	if driverID != "" {
		query = query.Where("driver_id = ?", driverID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&routes).Error
	if err != nil {
		return nil, 0, err
	}

	return routes, int(total), nil
}

func (r *PostgresRouteRepository) Update(route *models.Route) error {
	return r.db.Save(route).Error
}

func (r *PostgresRouteRepository) Delete(id string) error {
	return r.db.Delete(&models.Route{}, "id = ?", id).Error
}

func (r *PostgresRouteRepository) UpdateStatus(id, status string) error {
	return r.db.Model(&models.Route{}).Where("id = ?", id).Update("status", status).Error
}
