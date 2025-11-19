package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/joshbarros/golang-logistics-services/services/notification-service/internal/service"
)

type HTTPHandler struct {
	service *service.NotificationService
}

func NewHTTPHandler(service *service.NotificationService) *HTTPHandler {
	return &HTTPHandler{service: service}
}

type SendNotificationRequest struct {
	Recipient string `json:"recipient" binding:"required"`
	Type      string `json:"type" binding:"required"`
	Subject   string `json:"subject"`
	Message   string `json:"message" binding:"required"`
}

func (h *HTTPHandler) SendNotification(c *gin.Context) {
	var req SendNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	notification, err := h.service.SendNotification(req.Recipient, req.Type, req.Subject, req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, notification)
}

func (h *HTTPHandler) GetNotification(c *gin.Context) {
	id := c.Param("id")

	notification, err := h.service.GetNotification(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, notification)
}

func (h *HTTPHandler) ListNotifications(c *gin.Context) {
	page := 1
	pageSize := 20
	recipient := c.Query("recipient")
	notifType := c.Query("type")
	status := c.Query("status")

	if p := c.Query("page"); p != "" {
		if val, err := parseIntParam(p); err == nil {
			page = val
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if val, err := parseIntParam(ps); err == nil {
			pageSize = val
		}
	}

	notifications, total, err := h.service.ListNotifications(page, pageSize, recipient, notifType, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
		"total":         total,
		"page":          page,
		"page_size":     pageSize,
	})
}

func (h *HTTPHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "notification-service",
	})
}

func parseIntParam(s string) (int, error) {
	var val int
	_, err := fmt.Sscanf(s, "%d", &val)
	return val, err
}

func SetupRoutes(router *gin.Engine, handler *HTTPHandler) {
	router.GET("/health", handler.HealthCheck)

	v1 := router.Group("/api/v1")
	{
		notifications := v1.Group("/notifications")
		{
			notifications.POST("", handler.SendNotification)
			notifications.GET("", handler.ListNotifications)
			notifications.GET("/:id", handler.GetNotification)
		}
	}
}
