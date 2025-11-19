package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Notification struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"` // EMAIL, SMS, PUSH
	Recipient string    `json:"recipient"`
	Subject   string    `json:"subject"`
	Message   string    `json:"message"`
	Status    string    `json:"status"` // PENDING, SENT, FAILED
	SentAt    time.Time `json:"sent_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type SendNotificationRequest struct {
	Type      string `json:"type" binding:"required"`
	Recipient string `json:"recipient" binding:"required"`
	Subject   string `json:"subject"`
	Message   string `json:"message" binding:"required"`
}

var notificationsStore = make(map[string]*Notification)

func main() {
	log.Println("Starting Notification Service...")

	router := gin.Default()
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "notification-service"})
	})

	v1 := router.Group("/api/v1")
	{
		notifications := v1.Group("/notifications")
		{
			notifications.POST("/send", sendNotification)
			notifications.GET("/:id", getNotification)
			notifications.GET("", listNotifications)
		}
	}

	httpPort := getEnv("HTTP_PORT", "8085")
	log.Printf("Notification Service listening on port %s", httpPort)
	router.Run(fmt.Sprintf(":%s", httpPort))
}

func sendNotification(c *gin.Context) {
	var req SendNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	notification := &Notification{
		ID:        uuid.New().String(),
		Type:      req.Type,
		Recipient: req.Recipient,
		Subject:   req.Subject,
		Message:   req.Message,
		Status:    "PENDING",
		CreatedAt: time.Now(),
	}

	// Simulate sending notification
	go func() {
		time.Sleep(2 * time.Second)
		notification.Status = "SENT"
		notification.SentAt = time.Now()
		log.Printf("Notification sent: %s to %s via %s", notification.ID, notification.Recipient, notification.Type)
	}()

	notificationsStore[notification.ID] = notification
	c.JSON(http.StatusCreated, notification)
}

func getNotification(c *gin.Context) {
	id := c.Param("id")
	notification, exists := notificationsStore[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}
	c.JSON(http.StatusOK, notification)
}

func listNotifications(c *gin.Context) {
	notifications := make([]*Notification, 0, len(notificationsStore))
	for _, n := range notificationsStore {
		notifications = append(notifications, n)
	}
	c.JSON(http.StatusOK, gin.H{"notifications": notifications, "total": len(notifications)})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
