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
	driversStore = make(map[string]*Driver) // Reset store
	router := gin.New()
	router.POST("/api/v1/drivers", createDriver)
	router.GET("/api/v1/drivers", listDrivers)
	router.GET("/api/v1/drivers/:id", getDriver)
	router.PUT("/api/v1/drivers/:id", updateDriver)
	router.POST("/api/v1/drivers/:id/assign", assignDriver)
	router.GET("/api/v1/drivers/available", getAvailableDrivers)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})
	return router
}

func TestCreateDriver(t *testing.T) {
	router := setupTestRouter()

	tests := []struct {
		name           string
		requestBody    CreateDriverRequest
		expectedStatus int
	}{
		{
			name: "successful creation",
			requestBody: CreateDriverRequest{
				Name:      "John Doe",
				Email:     "john@example.com",
				Phone:     "555-1234",
				LicenseNo: "DL123456",
				VehicleID: "vehicle-1",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "missing required fields",
			requestBody: CreateDriverRequest{
				Name: "John",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/drivers", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if w.Code == http.StatusCreated {
				var driver Driver
				json.Unmarshal(w.Body.Bytes(), &driver)
				if driver.ID == "" {
					t.Error("expected driver ID to be set")
				}
				if driver.Status != "AVAILABLE" {
					t.Errorf("expected status AVAILABLE, got %s", driver.Status)
				}
			}
		})
	}
}

func TestGetDriver(t *testing.T) {
	router := setupTestRouter()

	// Create a driver first
	driver := &Driver{
		ID:     "driver-123",
		Name:   "John Doe",
		Email:  "john@example.com",
		Status: "AVAILABLE",
	}
	driversStore[driver.ID] = driver

	req, _ := http.NewRequest("GET", "/api/v1/drivers/driver-123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var result Driver
	json.Unmarshal(w.Body.Bytes(), &result)
	if result.ID != driver.ID {
		t.Errorf("expected ID %s, got %s", driver.ID, result.ID)
	}
}

func TestGetDriverNotFound(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest("GET", "/api/v1/drivers/driver-999", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestListDrivers(t *testing.T) {
	router := setupTestRouter()

	// Add some drivers
	driversStore["d1"] = &Driver{ID: "d1", Name: "Driver 1"}
	driversStore["d2"] = &Driver{ID: "d2", Name: "Driver 2"}

	req, _ := http.NewRequest("GET", "/api/v1/drivers", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	if total, ok := response["total"].(float64); !ok || total != 2 {
		t.Errorf("expected total 2, got %v", response["total"])
	}
}

func TestUpdateDriver(t *testing.T) {
	router := setupTestRouter()

	driver := &Driver{
		ID:     "driver-123",
		Name:   "John Doe",
		Status: "AVAILABLE",
	}
	driversStore[driver.ID] = driver

	updateBody := map[string]interface{}{
		"status": "OFF_DUTY",
	}
	body, _ := json.Marshal(updateBody)
	req, _ := http.NewRequest("PUT", "/api/v1/drivers/driver-123", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var result Driver
	json.Unmarshal(w.Body.Bytes(), &result)
	if result.Status != "OFF_DUTY" {
		t.Errorf("expected status OFF_DUTY, got %s", result.Status)
	}
}

func TestAssignDriver(t *testing.T) {
	router := setupTestRouter()

	driver := &Driver{
		ID:     "driver-123",
		Name:   "John Doe",
		Status: "AVAILABLE",
	}
	driversStore[driver.ID] = driver

	assignBody := AssignDriverRequest{
		RouteID: "route-456",
	}
	body, _ := json.Marshal(assignBody)
	req, _ := http.NewRequest("POST", "/api/v1/drivers/driver-123/assign", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["route_id"] != "route-456" {
		t.Errorf("expected route_id route-456, got %v", response["route_id"])
	}

	// Check driver status was updated
	if driversStore[driver.ID].Status != "ASSIGNED" {
		t.Errorf("expected driver status ASSIGNED, got %s", driversStore[driver.ID].Status)
	}
}

func TestGetAvailableDrivers(t *testing.T) {
	router := setupTestRouter()

	driversStore["d1"] = &Driver{ID: "d1", Name: "Driver 1", Status: "AVAILABLE"}
	driversStore["d2"] = &Driver{ID: "d2", Name: "Driver 2", Status: "ASSIGNED"}
	driversStore["d3"] = &Driver{ID: "d3", Name: "Driver 3", Status: "AVAILABLE"}

	req, _ := http.NewRequest("GET", "/api/v1/drivers/available", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	if count, ok := response["count"].(float64); !ok || count != 2 {
		t.Errorf("expected count 2, got %v", response["count"])
	}
}
