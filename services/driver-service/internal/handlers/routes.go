package handlers

import (
	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all routes for the driver service
func SetupRoutes(router *gin.Engine, handler *HTTPHandler) {
	// Health check
	router.GET("/health", handler.HealthCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		drivers := v1.Group("/drivers")
		{
			// CRUD operations
			drivers.POST("", handler.CreateDriver)
			drivers.GET("", handler.ListDrivers)
			drivers.GET("/available", handler.GetAvailableDrivers) // Must be before /:id
			drivers.GET("/:id", handler.GetDriver)
			drivers.PUT("/:id", handler.UpdateDriver)
			drivers.DELETE("/:id", handler.DeleteDriver)

			// Driver operations
			drivers.POST("/:id/assign", handler.AssignDriver)
			drivers.POST("/:id/unassign", handler.UnassignDriver)
			drivers.POST("/:id/location", handler.UpdateLocation)
		}
	}
}
