package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	notificationsStore = make(map[string]*Notification) // Reset store
	router := gin.New()
	router.POST("/api/v1/notifications/send", sendNotification)
	router.GET("/api/v1/notifications/:id", getNotification)
	router.GET("/api/v1/notifications", listNotifications)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})
	return router
}

func TestSendNotification(t *testing.T) {
	router := setupTestRouter()

	tests := []struct {
		name           string
		requestBody    SendNotificationRequest
		expectedStatus int
	}{
		{
			name: "successful email notification",
			requestBody: SendNotificationRequest{
				Type:      "EMAIL",
				Recipient: "user@example.com",
				Subject:   "Test Subject",
				Message:   "Test message",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "successful SMS notification",
			requestBody: SendNotificationRequest{
				Type:      "SMS",
				Recipient: "+1234567890",
				Message:   "Test SMS",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "missing required fields",
			requestBody: SendNotificationRequest{
				Type: "EMAIL",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/notifications/send", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if w.Code == http.StatusCreated {
				var notification Notification
				json.Unmarshal(w.Body.Bytes(), &notification)
				if notification.ID == "" {
					t.Error("expected notification ID to be set")
				}
				if notification.Type != tt.requestBody.Type {
					t.Errorf("expected type %s, got %s", tt.requestBody.Type, notification.Type)
				}
				if notification.Status != "PENDING" {
					t.Errorf("expected status PENDING, got %s", notification.Status)
				}
			}
		})
	}
}

func TestGetNotification(t *testing.T) {
	router := setupTestRouter()

	// Create a notification
	notification := &Notification{
		ID:        "notif-123",
		Type:      "EMAIL",
		Recipient: "user@example.com",
		Message:   "Test",
		Status:    "SENT",
		SentAt:    time.Now(),
	}
	notificationsStore[notification.ID] = notification

	req, _ := http.NewRequest("GET", "/api/v1/notifications/notif-123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var result Notification
	json.Unmarshal(w.Body.Bytes(), &result)
	if result.ID != notification.ID {
		t.Errorf("expected ID %s, got %s", notification.ID, result.ID)
	}
}

func TestGetNotificationNotFound(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest("GET", "/api/v1/notifications/notif-999", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestListNotifications(t *testing.T) {
	router := setupTestRouter()

	// Add some notifications
	notificationsStore["n1"] = &Notification{ID: "n1", Type: "EMAIL"}
	notificationsStore["n2"] = &Notification{ID: "n2", Type: "SMS"}

	req, _ := http.NewRequest("GET", "/api/v1/notifications", nil)
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

func TestNotificationStatusUpdate(t *testing.T) {
	router := setupTestRouter()

	reqBody := SendNotificationRequest{
		Type:      "EMAIL",
		Recipient: "user@example.com",
		Message:   "Test",
	}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/notifications/send", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	var notification Notification
	json.Unmarshal(w.Body.Bytes(), &notification)

	// Wait a bit for async status update
	time.Sleep(2500 * time.Millisecond)

	// Check if status was updated
	stored := notificationsStore[notification.ID]
	if stored.Status != "SENT" {
		t.Errorf("expected status SENT after async processing, got %s", stored.Status)
	}
}
