package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestCORSMiddleware(t *testing.T) {
	router := setupTestRouter()
	router.Use(corsMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	// Test OPTIONS request
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}

	// Check CORS headers
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("expected Access-Control-Allow-Origin header to be *")
	}

	// Test GET request
	req, _ = http.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestHealthCheck(t *testing.T) {
	router := setupTestRouter()
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "api-gateway",
		})
	})

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
	if response["service"] != "api-gateway" {
		t.Errorf("expected service 'api-gateway', got '%s'", response["service"])
	}
}

func TestForwardRequest(t *testing.T) {
	// Create a test backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "backend response"})
	}))
	defer backend.Close()

	router := setupTestRouter()
	router.GET("/proxy", func(c *gin.Context) {
		forwardRequest(c, backend.URL, "GET", nil)
	})

	req, _ := http.NewRequest("GET", "/proxy", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["message"] != "backend response" {
		t.Errorf("expected message 'backend response', got '%s'", response["message"])
	}
}

func TestForwardRequestWithBody(t *testing.T) {
	// Create a test backend server that echoes the request body
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		json.NewEncoder(w).Encode(body)
	}))
	defer backend.Close()

	router := setupTestRouter()
	router.POST("/proxy", func(c *gin.Context) {
		var body map[string]interface{}
		c.ShouldBindJSON(&body)
		forwardRequest(c, backend.URL, "POST", body)
	})

	reqBody := map[string]interface{}{"test": "data"}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/proxy", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Manually create gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Request.Body = http.NoBody

	forwardRequest(c, backend.URL, "POST", reqBody)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["test"] != "data" {
		t.Errorf("expected test 'data', got '%v'", response["test"])
	}
}

func TestForwardRequestServiceUnavailable(t *testing.T) {
	router := setupTestRouter()
	router.GET("/proxy", func(c *gin.Context) {
		// Use invalid URL to trigger error
		forwardRequest(c, "http://invalid-service:9999", "GET", nil)
	})

	req, _ := http.NewRequest("GET", "/proxy", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestProxyRequest(t *testing.T) {
	// Create a test backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"result": "ok"})
	}))
	defer backend.Close()

	router := setupTestRouter()
	handler := proxyRequest(backend.URL, "/test", "GET")
	router.GET("/proxy", handler)

	req, _ := http.NewRequest("GET", "/proxy?page=1&limit=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestProxyRequestWithParam(t *testing.T) {
	// Create a test backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": "123"})
	}))
	defer backend.Close()

	router := setupTestRouter()
	handler := proxyRequestWithParam(backend.URL, "/items", "GET")
	router.GET("/items/:id", handler)

	req, _ := http.NewRequest("GET", "/items/123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		expected     string
	}{
		{
			name:         "use default when env not set",
			key:          "NONEXISTENT_KEY",
			defaultValue: "default",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getEnv(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
