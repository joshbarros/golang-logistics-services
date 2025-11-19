package service

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/joshbarros/golang-logistics-services/services/route-service/internal/models"
	"github.com/joshbarros/golang-logistics-services/services/route-service/internal/repository"
)

type RouteService struct {
	repo repository.RouteRepository
}

func NewRouteService(repo repository.RouteRepository) *RouteService {
	return &RouteService{repo: repo}
}

// OptimizeRoute creates and optimizes a new route
func (s *RouteService) OptimizeRoute(name string, start, end models.Location, waypoints []models.Waypoint) (*models.Route, error) {
	// Optimize waypoint order using nearest neighbor algorithm
	optimizedWaypoints := s.nearestNeighbor(start, waypoints, end)

	// Assign sequence numbers
	for i := range optimizedWaypoints {
		optimizedWaypoints[i].Sequence = i + 1
		optimizedWaypoints[i].ID = uuid.New().String()
	}

	// Calculate total distance
	totalDistance := s.calculateTotalDistance(start, optimizedWaypoints, end)

	// Estimate duration (assuming 50 km/h average speed)
	estimatedDuration := int((totalDistance / 50.0) * 60)

	if name == "" {
		name = fmt.Sprintf("Route %s", uuid.New().String()[:8])
	}

	route := &models.Route{
		ID:                uuid.New().String(),
		Name:              name,
		StartLocation:     start,
		EndLocation:       end,
		Waypoints:         optimizedWaypoints,
		TotalDistance:     totalDistance,
		EstimatedDuration: estimatedDuration,
		Status:            models.StatusPlanned,
		Optimized:         true,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Set route_id for all waypoints
	for i := range route.Waypoints {
		route.Waypoints[i].RouteID = route.ID
	}

	err := s.repo.Create(route)
	if err != nil {
		return nil, fmt.Errorf("failed to create route: %w", err)
	}

	return route, nil
}

// CreateRoute creates a new route without optimization
func (s *RouteService) CreateRoute(name string, start, end models.Location, waypoints []models.Waypoint) (*models.Route, error) {
	if name == "" {
		name = fmt.Sprintf("Route %s", uuid.New().String()[:8])
	}

	// Assign IDs and sequence numbers
	for i := range waypoints {
		waypoints[i].ID = uuid.New().String()
		waypoints[i].Sequence = i + 1
	}

	totalDistance := s.calculateTotalDistance(start, waypoints, end)
	estimatedDuration := int((totalDistance / 50.0) * 60)

	route := &models.Route{
		ID:                uuid.New().String(),
		Name:              name,
		StartLocation:     start,
		EndLocation:       end,
		Waypoints:         waypoints,
		TotalDistance:     totalDistance,
		EstimatedDuration: estimatedDuration,
		Status:            models.StatusPlanned,
		Optimized:         false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	for i := range route.Waypoints {
		route.Waypoints[i].RouteID = route.ID
	}

	err := s.repo.Create(route)
	if err != nil {
		return nil, fmt.Errorf("failed to create route: %w", err)
	}

	return route, nil
}

func (s *RouteService) GetRoute(id string) (*models.Route, error) {
	if id == "" {
		return nil, errors.New("route ID is required")
	}
	return s.repo.GetByID(id)
}

func (s *RouteService) ListRoutes(page, pageSize int, driverID, status string) ([]*models.Route, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.List(page, pageSize, driverID, status)
}

func (s *RouteService) AssignDriver(routeID, driverID, vehicleID string) (*models.Route, error) {
	route, err := s.repo.GetByID(routeID)
	if err != nil {
		return nil, err
	}

	if route.Status != models.StatusPlanned {
		return nil, fmt.Errorf("can only assign driver to planned routes (current status: %s)", route.Status)
	}

	route.DriverID = driverID
	route.VehicleID = vehicleID
	route.UpdatedAt = time.Now()

	err = s.repo.Update(route)
	if err != nil {
		return nil, fmt.Errorf("failed to assign driver: %w", err)
	}

	return route, nil
}

func (s *RouteService) StartRoute(routeID string) (*models.Route, error) {
	route, err := s.repo.GetByID(routeID)
	if err != nil {
		return nil, err
	}

	if route.DriverID == "" {
		return nil, errors.New("cannot start route without assigned driver")
	}

	now := time.Now()
	route.Status = models.StatusInProgress
	route.StartedAt = &now
	route.UpdatedAt = now

	err = s.repo.Update(route)
	if err != nil {
		return nil, fmt.Errorf("failed to start route: %w", err)
	}

	return route, nil
}

func (s *RouteService) CompleteRoute(routeID string) (*models.Route, error) {
	route, err := s.repo.GetByID(routeID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	route.Status = models.StatusCompleted
	route.CompletedAt = &now
	route.UpdatedAt = now

	err = s.repo.Update(route)
	if err != nil {
		return nil, fmt.Errorf("failed to complete route: %w", err)
	}

	return route, nil
}

// Nearest neighbor algorithm for route optimization
func (s *RouteService) nearestNeighbor(start models.Location, waypoints []models.Waypoint, end models.Location) []models.Waypoint {
	if len(waypoints) == 0 {
		return []models.Waypoint{}
	}

	optimized := []models.Waypoint{}
	remaining := make([]models.Waypoint, len(waypoints))
	copy(remaining, waypoints)
	currentLat, currentLng := start.Latitude, start.Longitude

	for len(remaining) > 0 {
		nearest := 0
		minDist := math.MaxFloat64

		for i, wp := range remaining {
			dist := haversineDistance(currentLat, currentLng, wp.Location.Latitude, wp.Location.Longitude)
			if dist < minDist {
				minDist = dist
				nearest = i
			}
		}

		optimized = append(optimized, remaining[nearest])
		currentLat = remaining[nearest].Location.Latitude
		currentLng = remaining[nearest].Location.Longitude
		remaining = append(remaining[:nearest], remaining[nearest+1:]...)
	}

	return optimized
}

func (s *RouteService) calculateTotalDistance(start models.Location, waypoints []models.Waypoint, end models.Location) float64 {
	if len(waypoints) == 0 {
		return haversineDistance(start.Latitude, start.Longitude, end.Latitude, end.Longitude)
	}

	totalDistance := 0.0

	// Start to first waypoint
	totalDistance += haversineDistance(start.Latitude, start.Longitude, waypoints[0].Location.Latitude, waypoints[0].Location.Longitude)

	// Between waypoints
	for i := 0; i < len(waypoints)-1; i++ {
		totalDistance += haversineDistance(
			waypoints[i].Location.Latitude, waypoints[i].Location.Longitude,
			waypoints[i+1].Location.Latitude, waypoints[i+1].Location.Longitude,
		)
	}

	// Last waypoint to end
	totalDistance += haversineDistance(waypoints[len(waypoints)-1].Location.Latitude, waypoints[len(waypoints)-1].Location.Longitude, end.Latitude, end.Longitude)

	return totalDistance
}

// CalculateDistance calculates distance between two points
func (s *RouteService) CalculateDistance(from, to models.Location) float64 {
	return haversineDistance(from.Latitude, from.Longitude, to.Latitude, to.Longitude)
}

// Haversine formula to calculate distance between two coordinates
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth radius in kilometers

	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}
