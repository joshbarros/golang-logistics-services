package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type Waypoint struct {
	Address  string   `json:"address"`
	Location Location `json:"location"`
	Priority int      `json:"priority"`
}

type Route struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Waypoints   []Waypoint `json:"waypoints"`
	Distance    float64    `json:"distance"`
	Duration    float64    `json:"duration"`
	Optimized   bool       `json:"optimized"`
	DriverID    string     `json:"driver_id,omitempty"`
	VehicleID   string     `json:"vehicle_id,omitempty"`
	Status      string     `json:"status"`
}

type OptimizeRouteRequest struct {
	Origin      Waypoint   `json:"origin" binding:"required"`
	Destination Waypoint   `json:"destination" binding:"required"`
	Waypoints   []Waypoint `json:"waypoints"`
}

func main() {
	log.Println("Starting Route Service...")

	router := gin.Default()
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "route-service"})
	})

	v1 := router.Group("/api/v1")
	{
		routes := v1.Group("/routes")
		{
			routes.POST("/optimize", optimizeRoute)
			routes.GET("/:id", getRoute)
			routes.GET("", listRoutes)
		}
	}

	httpPort := getEnv("HTTP_PORT", "8083")
	log.Printf("Route Service listening on port %s", httpPort)
	router.Run(fmt.Sprintf(":%s", httpPort))
}

func optimizeRoute(c *gin.Context) {
	var req OptimizeRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Simple route optimization (nearest neighbor algorithm)
	optimizedWaypoints := nearestNeighbor(req.Origin, req.Waypoints, req.Destination)

	// Calculate total distance
	totalDistance := 0.0
	allPoints := append([]Waypoint{req.Origin}, optimizedWaypoints...)
	allPoints = append(allPoints, req.Destination)

	for i := 0; i < len(allPoints)-1; i++ {
		totalDistance += haversineDistance(
			allPoints[i].Location.Lat, allPoints[i].Location.Lng,
			allPoints[i+1].Location.Lat, allPoints[i+1].Location.Lng,
		)
	}

	route := Route{
		ID:        uuid.New().String(),
		Name:      fmt.Sprintf("Route %s", uuid.New().String()[:8]),
		Waypoints: optimizedWaypoints,
		Distance:  totalDistance,
		Duration:  totalDistance / 60.0 * 60, // Assuming 60 km/h average speed
		Optimized: true,
		Status:    "PLANNED",
	}

	c.JSON(http.StatusOK, route)
}

func getRoute(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"id": id, "message": "Route details"})
}

func listRoutes(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"routes": []Route{}})
}

// Nearest neighbor algorithm for route optimization
func nearestNeighbor(origin Waypoint, waypoints []Waypoint, destination Waypoint) []Waypoint {
	if len(waypoints) == 0 {
		return []Waypoint{}
	}

	optimized := []Waypoint{}
	remaining := make([]Waypoint, len(waypoints))
	copy(remaining, waypoints)
	current := origin

	for len(remaining) > 0 {
		nearest := 0
		minDist := math.MaxFloat64

		for i, wp := range remaining {
			dist := haversineDistance(current.Location.Lat, current.Location.Lng, wp.Location.Lat, wp.Location.Lng)
			if dist < minDist {
				minDist = dist
				nearest = i
			}
		}

		optimized = append(optimized, remaining[nearest])
		current = remaining[nearest]
		remaining = append(remaining[:nearest], remaining[nearest+1:]...)
	}

	return optimized
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

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
