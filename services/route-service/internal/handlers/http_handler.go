package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/joshbarros/golang-logistics-services/services/route-service/internal/models"
	"github.com/joshbarros/golang-logistics-services/services/route-service/internal/service"
)

type HTTPHandler struct {
	service *service.RouteService
}

func NewHTTPHandler(service *service.RouteService) *HTTPHandler {
	return &HTTPHandler{service: service}
}

type OptimizeRouteRequest struct {
	Name        string            `json:"name"`
	Start       models.Location   `json:"start" binding:"required"`
	End         models.Location   `json:"end" binding:"required"`
	Waypoints   []models.Waypoint `json:"waypoints"`
	Optimize    bool              `json:"optimize"`
}

type AssignDriverRequest struct {
	DriverID  string `json:"driver_id" binding:"required"`
	VehicleID string `json:"vehicle_id"`
}

type CalculateDistanceRequest struct {
	From models.Location `json:"from" binding:"required"`
	To   models.Location `json:"to" binding:"required"`
}

func (h *HTTPHandler) OptimizeRoute(c *gin.Context) {
	var req OptimizeRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var route *models.Route
	var err error

	if req.Optimize {
		route, err = h.service.OptimizeRoute(req.Name, req.Start, req.End, req.Waypoints)
	} else {
		route, err = h.service.CreateRoute(req.Name, req.Start, req.End, req.Waypoints)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, route)
}

func (h *HTTPHandler) GetRoute(c *gin.Context) {
	id := c.Param("id")

	route, err := h.service.GetRoute(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, route)
}

func (h *HTTPHandler) ListRoutes(c *gin.Context) {
	page := 1
	pageSize := 20
	driverID := c.Query("driver_id")
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

	routes, total, err := h.service.ListRoutes(page, pageSize, driverID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"routes":    routes,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *HTTPHandler) AssignDriver(c *gin.Context) {
	routeID := c.Param("id")

	var req AssignDriverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	route, err := h.service.AssignDriver(routeID, req.DriverID, req.VehicleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, route)
}

func (h *HTTPHandler) StartRoute(c *gin.Context) {
	routeID := c.Param("id")

	route, err := h.service.StartRoute(routeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, route)
}

func (h *HTTPHandler) CompleteRoute(c *gin.Context) {
	routeID := c.Param("id")

	route, err := h.service.CompleteRoute(routeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, route)
}

func (h *HTTPHandler) CalculateDistance(c *gin.Context) {
	var req CalculateDistanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	distance := h.service.CalculateDistance(req.From, req.To)

	c.JSON(http.StatusOK, gin.H{
		"distance_km": distance,
		"estimated_duration_minutes": int((distance / 50.0) * 60),
	})
}

func (h *HTTPHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "route-service",
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
		routes := v1.Group("/routes")
		{
			routes.POST("/optimize", handler.OptimizeRoute)
			routes.POST("/calculate-distance", handler.CalculateDistance)
			routes.GET("", handler.ListRoutes)
			routes.GET("/:id", handler.GetRoute)
			routes.POST("/:id/assign", handler.AssignDriver)
			routes.POST("/:id/start", handler.StartRoute)
			routes.POST("/:id/complete", handler.CompleteRoute)
		}
	}
}
