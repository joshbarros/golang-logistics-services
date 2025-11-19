package health

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Status represents the health status
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// CheckResult represents the result of a health check
type CheckResult struct {
	Status  Status                 `json:"status"`
	Service string                 `json:"service"`
	Version string                 `json:"version,omitempty"`
	Checks  map[string]CheckDetail `json:"checks"`
}

// CheckDetail represents details of a specific check
type CheckDetail struct {
	Status  Status `json:"status"`
	Message string `json:"message,omitempty"`
	Latency string `json:"latency,omitempty"`
}

// Checker performs health checks
type Checker struct {
	serviceName string
	version     string
	db          *gorm.DB
}

// NewChecker creates a new health checker
func NewChecker(serviceName, version string, db *gorm.DB) *Checker {
	return &Checker{
		serviceName: serviceName,
		version:     version,
		db:          db,
	}
}

// Check performs all health checks
func (h *Checker) Check(c *gin.Context) {
	checks := make(map[string]CheckDetail)
	overallStatus := StatusHealthy

	// Check database connection
	if h.db != nil {
		dbCheck := h.checkDatabase()
		checks["database"] = dbCheck
		if dbCheck.Status != StatusHealthy {
			overallStatus = StatusDegraded
		}
	}

	// Check API
	checks["api"] = CheckDetail{
		Status:  StatusHealthy,
		Message: "API is responding",
	}

	result := CheckResult{
		Status:  overallStatus,
		Service: h.serviceName,
		Version: h.version,
		Checks:  checks,
	}

	statusCode := http.StatusOK
	if overallStatus == StatusDegraded {
		statusCode = http.StatusOK // Still return 200 for degraded
	} else if overallStatus == StatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, result)
}

// checkDatabase checks database connectivity
func (h *Checker) checkDatabase() CheckDetail {
	start := time.Now()

	sqlDB, err := h.db.DB()
	if err != nil {
		return CheckDetail{
			Status:  StatusUnhealthy,
			Message: "Failed to get database connection: " + err.Error(),
			Latency: time.Since(start).String(),
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = sqlDB.PingContext(ctx)
	if err != nil {
		return CheckDetail{
			Status:  StatusUnhealthy,
			Message: "Database ping failed: " + err.Error(),
			Latency: time.Since(start).String(),
		}
	}

	// Check connection pool stats
	stats := sqlDB.Stats()
	if stats.OpenConnections == 0 {
		return CheckDetail{
			Status:  StatusDegraded,
			Message: "No database connections available",
			Latency: time.Since(start).String(),
		}
	}

	return CheckDetail{
		Status:  StatusHealthy,
		Message: "Database connection is healthy",
		Latency: time.Since(start).String(),
	}
}

// ReadinessHandler returns a simple readiness check
func ReadinessHandler(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ready",
			"service": serviceName,
		})
	}
}

// LivenessHandler returns a simple liveness check
func LivenessHandler(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "alive",
			"service": serviceName,
		})
	}
}

// ConfigureDBConnectionPool configures optimal connection pool settings
func ConfigureDBConnectionPool(sqlDB *sql.DB) {
	// Set maximum number of open connections
	sqlDB.SetMaxOpenConns(25)

	// Set maximum number of idle connections
	sqlDB.SetMaxIdleConns(5)

	// Set maximum lifetime of a connection
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	// Set maximum idle time for a connection
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)
}
