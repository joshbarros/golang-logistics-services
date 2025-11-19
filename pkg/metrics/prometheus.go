package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	requestSize     *prometheus.SummaryVec
	responseSize    *prometheus.SummaryVec
	activeRequests  *prometheus.GaugeVec
}

// NewMetrics creates and registers Prometheus metrics
func NewMetrics(serviceName string) *Metrics {
	m := &Metrics{
		requestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
			},
			[]string{"method", "endpoint", "status"},
		),
		requestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "http_request_duration_seconds",
				Help: "HTTP request duration in seconds",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
				Buckets: prometheus.DefBuckets, // [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]
			},
			[]string{"method", "endpoint", "status"},
		),
		requestSize: promauto.NewSummaryVec(
			prometheus.SummaryOpts{
				Name: "http_request_size_bytes",
				Help: "HTTP request size in bytes",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
			},
			[]string{"method", "endpoint"},
		),
		responseSize: promauto.NewSummaryVec(
			prometheus.SummaryOpts{
				Name: "http_response_size_bytes",
				Help: "HTTP response size in bytes",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
			},
			[]string{"method", "endpoint", "status"},
		),
		activeRequests: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "http_requests_active",
				Help: "Number of active HTTP requests",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
			},
			[]string{"method", "endpoint"},
		),
	}

	return m
}

// Middleware returns a Gin middleware for Prometheus metrics
func (m *Metrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip metrics endpoint itself
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		start := time.Now()
		method := c.Request.Method
		endpoint := c.FullPath()

		// If endpoint template not found, use path
		if endpoint == "" {
			endpoint = c.Request.URL.Path
		}

		// Track active requests
		m.activeRequests.WithLabelValues(method, endpoint).Inc()
		defer m.activeRequests.WithLabelValues(method, endpoint).Dec()

		// Track request size
		if c.Request.ContentLength > 0 {
			m.requestSize.WithLabelValues(method, endpoint).Observe(float64(c.Request.ContentLength))
		}

		// Process request
		c.Next()

		// Track request completion
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())

		m.requestsTotal.WithLabelValues(method, endpoint, status).Inc()
		m.requestDuration.WithLabelValues(method, endpoint, status).Observe(duration)
		m.responseSize.WithLabelValues(method, endpoint, status).Observe(float64(c.Writer.Size()))
	}
}

// Handler returns the Prometheus HTTP handler
func (m *Metrics) Handler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// Custom Business Metrics

// BusinessMetrics holds custom business metrics
type BusinessMetrics struct {
	ordersCreated       prometheus.Counter
	ordersCompleted     prometheus.Counter
	ordersCancelled     prometheus.Counter
	orderValue          prometheus.Histogram
	shipmentsCreated    prometheus.Counter
	shipmentsDelivered  prometheus.Counter
	inventoryLevel      *prometheus.GaugeVec
	driversActive       prometheus.Gauge
	routesOptimized     prometheus.Counter
	notificationsSent   *prometheus.CounterVec
	databaseConnections *prometheus.GaugeVec
	cacheHits           *prometheus.CounterVec
}

// NewBusinessMetrics creates custom business metrics
func NewBusinessMetrics(serviceName string) *BusinessMetrics {
	return &BusinessMetrics{
		ordersCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "orders_created_total",
			Help: "Total number of orders created",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		}),
		ordersCompleted: promauto.NewCounter(prometheus.CounterOpts{
			Name: "orders_completed_total",
			Help: "Total number of orders completed",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		}),
		ordersCancelled: promauto.NewCounter(prometheus.CounterOpts{
			Name: "orders_cancelled_total",
			Help: "Total number of orders cancelled",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		}),
		orderValue: promauto.NewHistogram(prometheus.HistogramOpts{
			Name: "order_value_dollars",
			Help: "Distribution of order values in dollars",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
			Buckets: []float64{10, 50, 100, 250, 500, 1000, 2500, 5000},
		}),
		shipmentsCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "shipments_created_total",
			Help: "Total number of shipments created",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		}),
		shipmentsDelivered: promauto.NewCounter(prometheus.CounterOpts{
			Name: "shipments_delivered_total",
			Help: "Total number of shipments delivered",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		}),
		inventoryLevel: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "inventory_items_count",
			Help: "Current inventory level by product",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		}, []string{"product_id", "warehouse"}),
		driversActive: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "drivers_active_count",
			Help: "Number of active drivers",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		}),
		routesOptimized: promauto.NewCounter(prometheus.CounterOpts{
			Name: "routes_optimized_total",
			Help: "Total number of routes optimized",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		}),
		notificationsSent: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "notifications_sent_total",
			Help: "Total number of notifications sent",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		}, []string{"type", "status"}),
		databaseConnections: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "database_connections",
			Help: "Current database connections",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		}, []string{"state"}),
		cacheHits: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "cache_operations_total",
			Help: "Total cache operations",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		}, []string{"operation", "result"}),
	}
}

// RecordOrderCreated increments orders created counter
func (m *BusinessMetrics) RecordOrderCreated() {
	m.ordersCreated.Inc()
}

// RecordOrderCompleted increments orders completed counter
func (m *BusinessMetrics) RecordOrderCompleted() {
	m.ordersCompleted.Inc()
}

// RecordOrderCancelled increments orders cancelled counter
func (m *BusinessMetrics) RecordOrderCancelled() {
	m.ordersCancelled.Inc()
}

// RecordOrderValue records order value
func (m *BusinessMetrics) RecordOrderValue(value float64) {
	m.orderValue.Observe(value)
}

// RecordShipmentCreated increments shipments created counter
func (m *BusinessMetrics) RecordShipmentCreated() {
	m.shipmentsCreated.Inc()
}

// RecordShipmentDelivered increments shipments delivered counter
func (m *BusinessMetrics) RecordShipmentDelivered() {
	m.shipmentsDelivered.Inc()
}

// SetInventoryLevel sets inventory level for a product
func (m *BusinessMetrics) SetInventoryLevel(productID, warehouse string, level float64) {
	m.inventoryLevel.WithLabelValues(productID, warehouse).Set(level)
}

// SetActiveDrivers sets number of active drivers
func (m *BusinessMetrics) SetActiveDrivers(count float64) {
	m.driversActive.Set(count)
}

// RecordRouteOptimized increments routes optimized counter
func (m *BusinessMetrics) RecordRouteOptimized() {
	m.routesOptimized.Inc()
}

// RecordNotificationSent increments notifications sent counter
func (m *BusinessMetrics) RecordNotificationSent(notificationType, status string) {
	m.notificationsSent.WithLabelValues(notificationType, status).Inc()
}

// SetDatabaseConnections sets current database connections
func (m *BusinessMetrics) SetDatabaseConnections(state string, count float64) {
	m.databaseConnections.WithLabelValues(state).Set(count)
}

// RecordCacheOperation records a cache operation
func (m *BusinessMetrics) RecordCacheOperation(operation, result string) {
	m.cacheHits.WithLabelValues(operation, result).Inc()
}
