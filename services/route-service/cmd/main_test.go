package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/routes/optimize", optimizeRoute)
	router.GET("/api/v1/routes/:id", getRoute)
	router.GET("/api/v1/routes", listRoutes)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})
	return router
}

func TestOptimizeRoute(t *testing.T) {
	router := setupTestRouter()

	tests := []struct {
		name           string
		requestBody    OptimizeRouteRequest
		expectedStatus int
	}{
		{
			name: "successful optimization",
			requestBody: OptimizeRouteRequest{
				Origin: Waypoint{
					Address:  "Origin",
					Location: Location{Lat: 42.3601, Lng: -71.0589},
					Priority: 1,
				},
				Destination: Waypoint{
					Address:  "Destination",
					Location: Location{Lat: 40.7128, Lng: -74.0060},
					Priority: 3,
				},
				Waypoints: []Waypoint{
					{Address: "Stop 1", Location: Location{Lat: 41.8781, Lng: -87.6298}, Priority: 2},
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid request",
			requestBody:    OptimizeRouteRequest{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/routes/optimize", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if w.Code == http.StatusOK {
				var route Route
				json.Unmarshal(w.Body.Bytes(), &route)
				if route.ID == "" {
					t.Error("expected route ID to be set")
				}
				if !route.Optimized {
					t.Error("expected route to be optimized")
				}
			}
		})
	}
}

func TestGetRoute(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest("GET", "/api/v1/routes/route-123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestListRoutes(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest("GET", "/api/v1/routes", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestHaversineDistance(t *testing.T) {
	tests := []struct {
		name     string
		lat1     float64
		lon1     float64
		lat2     float64
		lon2     float64
		expected float64
		delta    float64
	}{
		{
			name:     "Boston to New York",
			lat1:     42.3601,
			lon1:     -71.0589,
			lat2:     40.7128,
			lon2:     -74.0060,
			expected: 306, // approximately 306 km
			delta:    10,
		},
		{
			name:     "Same location",
			lat1:     42.3601,
			lon1:     -71.0589,
			lat2:     42.3601,
			lon2:     -71.0589,
			expected: 0,
			delta:    0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distance := haversineDistance(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			if distance < tt.expected-tt.delta || distance > tt.expected+tt.delta {
				t.Errorf("expected distance around %f, got %f", tt.expected, distance)
			}
		})
	}
}

func TestNearestNeighbor(t *testing.T) {
	origin := Waypoint{
		Address:  "Origin",
		Location: Location{Lat: 0, Lng: 0},
	}

	waypoints := []Waypoint{
		{Address: "Far", Location: Location{Lat: 10, Lng: 10}},
		{Address: "Near", Location: Location{Lat: 1, Lng: 1}},
		{Address: "Medium", Location: Location{Lat: 5, Lng: 5}},
	}

	destination := Waypoint{
		Address:  "Destination",
		Location: Location{Lat: 11, Lng: 11},
	}

	optimized := nearestNeighbor(origin, waypoints, destination)

	if len(optimized) != 3 {
		t.Errorf("expected 3 waypoints, got %d", len(optimized))
	}

	// First should be nearest to origin
	if optimized[0].Address != "Near" {
		t.Errorf("expected first waypoint to be 'Near', got '%s'", optimized[0].Address)
	}
}

func TestHealthCheck(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got '%s'", response["status"])
	}
}
