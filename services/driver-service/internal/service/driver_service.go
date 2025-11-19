package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/joshbarros/golang-logistics-services/services/driver-service/internal/models"
	"github.com/joshbarros/golang-logistics-services/services/driver-service/internal/repository"
)

// DriverService handles business logic for drivers
type DriverService struct {
	repo repository.DriverRepository
}

// NewDriverService creates a new driver service
func NewDriverService(repo repository.DriverRepository) *DriverService {
	return &DriverService{repo: repo}
}

// CreateDriver creates a new driver
func (s *DriverService) CreateDriver(name, email, phone, licenseNumber, vehicleType, vehiclePlate string) (*models.Driver, error) {
	// Validate inputs
	if name == "" || email == "" || phone == "" || licenseNumber == "" {
		return nil, errors.New("name, email, phone, and license number are required")
	}

	// Check if email already exists
	existingDriver, _ := s.repo.GetByEmail(email)
	if existingDriver != nil {
		return nil, errors.New("driver with this email already exists")
	}

	driver := &models.Driver{
		ID:            uuid.New().String(),
		Name:          name,
		Email:         email,
		Phone:         phone,
		LicenseNumber: licenseNumber,
		VehicleType:   vehicleType,
		VehiclePlate:  vehiclePlate,
		Status:        models.StatusAvailable,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err := s.repo.Create(driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create driver: %w", err)
	}

	return driver, nil
}

// GetDriver retrieves a driver by ID
func (s *DriverService) GetDriver(id string) (*models.Driver, error) {
	if id == "" {
		return nil, errors.New("driver ID is required")
	}

	return s.repo.GetByID(id)
}

// ListDrivers returns a paginated list of drivers
func (s *DriverService) ListDrivers(page, pageSize int, status string) ([]*models.Driver, int, error) {
	// Normalize pagination
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	return s.repo.List(page, pageSize, status)
}

// UpdateDriver updates driver information
func (s *DriverService) UpdateDriver(id, name, email, phone, status string) (*models.Driver, error) {
	driver, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if name != "" {
		driver.Name = name
	}
	if email != "" {
		driver.Email = email
	}
	if phone != "" {
		driver.Phone = phone
	}
	if status != "" {
		if !isValidStatus(status) {
			return nil, fmt.Errorf("invalid status: %s", status)
		}
		driver.Status = status
	}

	driver.UpdatedAt = time.Now()

	err = s.repo.Update(driver)
	if err != nil {
		return nil, fmt.Errorf("failed to update driver: %w", err)
	}

	return driver, nil
}

// DeleteDriver deletes a driver
func (s *DriverService) DeleteDriver(id string) error {
	driver, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	// Don't delete drivers that are currently assigned
	if driver.Status == models.StatusAssigned || driver.Status == models.StatusDelivering {
		return errors.New("cannot delete driver that is currently assigned or delivering")
	}

	return s.repo.Delete(id)
}

// AssignDriver assigns a driver to a route
func (s *DriverService) AssignDriver(driverID, routeID string) (*models.Driver, error) {
	driver, err := s.repo.GetByID(driverID)
	if err != nil {
		return nil, err
	}

	// Check if driver is available
	if driver.Status != models.StatusAvailable {
		return nil, fmt.Errorf("driver is not available (current status: %s)", driver.Status)
	}

	err = s.repo.AssignRoute(driverID, routeID)
	if err != nil {
		return nil, fmt.Errorf("failed to assign driver: %w", err)
	}

	// Return updated driver
	return s.repo.GetByID(driverID)
}

// UnassignDriver removes route assignment from a driver
func (s *DriverService) UnassignDriver(driverID string) (*models.Driver, error) {
	driver, err := s.repo.GetByID(driverID)
	if err != nil {
		return nil, err
	}

	driver.CurrentRouteID = ""
	driver.Status = models.StatusAvailable
	driver.UpdatedAt = time.Now()

	err = s.repo.Update(driver)
	if err != nil {
		return nil, fmt.Errorf("failed to unassign driver: %w", err)
	}

	return driver, nil
}

// GetAvailableDrivers returns all available drivers
func (s *DriverService) GetAvailableDrivers() ([]*models.Driver, error) {
	return s.repo.GetAvailableDrivers()
}

// UpdateLocation updates a driver's current location
func (s *DriverService) UpdateLocation(driverID string, lat, lng float64) (*models.Driver, error) {
	// Validate coordinates
	if lat < -90 || lat > 90 {
		return nil, errors.New("invalid latitude (must be between -90 and 90)")
	}
	if lng < -180 || lng > 180 {
		return nil, errors.New("invalid longitude (must be between -180 and 180)")
	}

	err := s.repo.UpdateLocation(driverID, lat, lng)
	if err != nil {
		return nil, fmt.Errorf("failed to update location: %w", err)
	}

	return s.repo.GetByID(driverID)
}

// UpdateStatus updates a driver's status
func (s *DriverService) UpdateStatus(driverID, status string) (*models.Driver, error) {
	if !isValidStatus(status) {
		return nil, fmt.Errorf("invalid status: %s", status)
	}

	err := s.repo.UpdateStatus(driverID, status)
	if err != nil {
		return nil, fmt.Errorf("failed to update status: %w", err)
	}

	return s.repo.GetByID(driverID)
}

// Helper function to validate driver status
func isValidStatus(status string) bool {
	validStatuses := map[string]bool{
		models.StatusAvailable:  true,
		models.StatusAssigned:   true,
		models.StatusDelivering: true,
		models.StatusOffDuty:    true,
	}
	return validStatuses[status]
}
