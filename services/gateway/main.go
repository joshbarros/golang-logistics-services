package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"golang-logistics-services/pkg/circuitbreaker"
	"golang-logistics-services/pkg/graphql"
	"golang-logistics-services/pkg/metrics"
	"golang-logistics-services/pkg/shutdown"
	"golang-logistics-services/pkg/tracing"
)

// Config holds gateway configuration
type Config struct {
	Port              string
	JWTSecret         string
	EnablePlayground  bool
	RateLimitRPM      int
	MaxComplexity     int
	MaxDepth          int
	QueryTimeout      time.Duration
	AllowedOrigins    []string

	// Service endpoints
	OrderServiceURL      string
	ShipmentServiceURL   string
	InventoryServiceURL  string
	DriverServiceURL     string
	RouteServiceURL      string
	NotificationServiceURL string
}

// DefaultConfig returns default gateway configuration
func DefaultConfig() Config {
	return Config{
		Port:             getEnv("PORT", "8080"),
		JWTSecret:        getEnv("JWT_SECRET", "secret-key"),
		EnablePlayground: getEnv("ENABLE_PLAYGROUND", "true") == "true",
		RateLimitRPM:     100,
		MaxComplexity:    1000,
		MaxDepth:         10,
		QueryTimeout:     30 * time.Second,
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:8080"},

		OrderServiceURL:      getEnv("ORDER_SERVICE_URL", "localhost:50051"),
		ShipmentServiceURL:   getEnv("SHIPMENT_SERVICE_URL", "localhost:50052"),
		InventoryServiceURL:  getEnv("INVENTORY_SERVICE_URL", "localhost:50053"),
		DriverServiceURL:     getEnv("DRIVER_SERVICE_URL", "localhost:50054"),
		RouteServiceURL:      getEnv("ROUTE_SERVICE_URL", "localhost:50055"),
		NotificationServiceURL: getEnv("NOTIFICATION_SERVICE_URL", "localhost:50056"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	// Load configuration
	config := DefaultConfig()

	// Initialize metrics
	metricsConfig := metrics.DefaultConfig()
	metricsConfig.ServiceName = "gateway"
	metricsConfig.Port = 9090
	metricsManager, err := metrics.NewMetricsManager(metricsConfig)
	if err != nil {
		log.Fatalf("Failed to initialize metrics: %v", err)
	}
	defer metricsManager.Close()

	// Initialize tracing
	tracingConfig := tracing.DefaultConfig()
	tracingConfig.ServiceName = "gateway"
	tracingConfig.JaegerEndpoint = getEnv("JAEGER_ENDPOINT", "http://localhost:14268/api/traces")
	tracingManager, err := tracing.NewTracingManager(tracingConfig)
	if err != nil {
		log.Fatalf("Failed to initialize tracing: %v", err)
	}
	defer tracingManager.Close()

	// Initialize circuit breakers for each service
	cbManager := circuitbreaker.NewManager()

	orderCB := cbManager.GetOrCreate("order-service", circuitbreaker.DefaultConfig())
	shipmentCB := cbManager.GetOrCreate("shipment-service", circuitbreaker.DefaultConfig())
	inventoryCB := cbManager.GetOrCreate("inventory-service", circuitbreaker.DefaultConfig())
	driverCB := cbManager.GetOrCreate("driver-service", circuitbreaker.DefaultConfig())
	routeCB := cbManager.GetOrCreate("route-service", circuitbreaker.DefaultConfig())
	notificationCB := cbManager.GetOrCreate("notification-service", circuitbreaker.DefaultConfig())

	log.Printf("Circuit breakers initialized: order=%s, shipment=%s, inventory=%s",
		orderCB.State(), shipmentCB.State(), inventoryCB.State())

	// Initialize service clients (gRPC clients wrapped with circuit breakers)
	serviceClients := graphql.ServiceClients{
		// In production, initialize actual gRPC clients here
		// OrderClient: pb.NewOrderServiceClient(orderConn),
		// These would be wrapped with circuit breaker protection
	}

	// Create GraphQL server
	gqlConfig := graphql.Config{
		Playground:     config.EnablePlayground,
		Pretty:         true,
		MaxComplexity:  config.MaxComplexity,
		MaxDepth:       config.MaxDepth,
		Timeout:        config.QueryTimeout,
		ServiceClients: serviceClients,
	}

	gqlServer, err := graphql.NewServer(gqlConfig)
	if err != nil {
		log.Fatalf("Failed to create GraphQL server: %v", err)
	}

	// Setup Gin router
	router := gin.Default()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"service": "gateway",
			"timestamp": time.Now().Unix(),
			"circuit_breakers": gin.H{
				"order":        orderCB.State(),
				"shipment":     shipmentCB.State(),
				"inventory":    inventoryCB.State(),
				"driver":       driverCB.State(),
				"route":        routeCB.State(),
				"notification": notificationCB.State(),
			},
		})
	})

	// Readiness check endpoint
	router.GET("/ready", func(c *gin.Context) {
		// Check if all services are reachable
		ready := true
		services := map[string]string{
			"order":      orderCB.State(),
			"shipment":   shipmentCB.State(),
			"inventory":  inventoryCB.State(),
			"driver":     driverCB.State(),
			"route":      routeCB.State(),
			"notification": notificationCB.State(),
		}

		for _, state := range services {
			if state == "open" {
				ready = false
				break
			}
		}

		if ready {
			c.JSON(http.StatusOK, gin.H{
				"status": "ready",
				"services": services,
			})
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not_ready",
				"services": services,
			})
		}
	})

	// Metrics endpoint
	router.GET("/metrics", gin.WrapH(metricsManager.Handler()))

	// GraphQL endpoints with middleware
	graphqlGroup := router.Group("/graphql")
	{
		// Apply middleware
		graphqlGroup.Use(graphql.CORSMiddleware(config.AllowedOrigins))
		graphqlGroup.Use(graphql.MetricsMiddleware())
		graphqlGroup.Use(graphql.LoggingMiddleware())
		graphqlGroup.Use(tracingManager.Middleware())

		// Optional auth middleware (can be disabled for public queries)
		if getEnv("REQUIRE_AUTH", "false") == "true" {
			graphqlGroup.Use(graphql.AuthMiddleware(config.JWTSecret))
		}

		graphqlGroup.Use(graphql.RateLimitMiddleware(config.RateLimitRPM))
		graphqlGroup.Use(graphql.ComplexityMiddleware(config.MaxComplexity))
		graphqlGroup.Use(graphql.DepthMiddleware(config.MaxDepth))
		graphqlGroup.Use(graphql.TimeoutMiddleware(config.QueryTimeout))

		// GraphQL endpoint
		graphqlGroup.POST("", gqlServer.Handler())
		graphqlGroup.GET("", gqlServer.Handler())
	}

	// GraphQL Playground (development only)
	if config.EnablePlayground {
		router.GET("/playground", gqlServer.PlaygroundHandler())
		log.Printf("GraphQL Playground available at http://localhost:%s/playground", config.Port)
	}

	// API documentation
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "Logistics Platform API Gateway",
			"version": "1.0.0",
			"endpoints": gin.H{
				"graphql":     "/graphql",
				"playground":  "/playground",
				"health":      "/health",
				"ready":       "/ready",
				"metrics":     "/metrics",
			},
			"features": []string{
				"GraphQL API with unified schema",
				"Real-time subscriptions",
				"DataLoader for N+1 query optimization",
				"Circuit breaker protection",
				"Rate limiting",
				"Query complexity and depth limits",
				"Distributed tracing",
				"Prometheus metrics",
			},
		})
	})

	// Start server
	srv := &http.Server{
		Addr:           ":" + config.Port,
		Handler:        router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Initialize graceful shutdown manager
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	shutdownMgr := shutdown.NewManager(shutdown.Config{
		Logger:          logger,
		ShutdownTimeout: 30 * time.Second,
	})

	// Register components for graceful shutdown (priority-based)
	// Priority 10: HTTP server (stop accepting new requests first)
	shutdownMgr.RegisterHTTPServer("gateway-http", srv)

	// Priority 15: Metrics manager (allow final metrics collection)
	shutdownMgr.Register("metrics", 15, func(ctx context.Context) error {
		log.Println("Closing metrics manager...")
		return metricsManager.Close()
	})

	// Priority 20: Tracing manager (flush remaining traces)
	shutdownMgr.Register("tracing", 20, func(ctx context.Context) error {
		log.Println("Closing tracing manager...")
		return tracingManager.Close()
	})

	// Priority 25: Circuit breaker manager (cleanup)
	shutdownMgr.Register("circuit-breakers", 25, func(ctx context.Context) error {
		log.Println("Shutting down circuit breakers...")
		// Circuit breakers cleanup if needed
		return nil
	})

	// Start server in background
	go func() {
		log.Printf("Starting Gateway API on port %s", config.Port)
		log.Printf("GraphQL endpoint: http://localhost:%s/graphql", config.Port)
		log.Printf("Health endpoint: http://localhost:%s/health", config.Port)
		log.Printf("Metrics endpoint: http://localhost:%s/metrics", config.Port)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for shutdown signal and execute graceful shutdown
	if err := shutdownMgr.Wait(); err != nil {
		log.Fatalf("Shutdown error: %v", err)
	}

	log.Println("Gateway API exited gracefully")
}

// Example GraphQL queries:
//
// # Query single order with nested relationships
// query GetOrder {
//   order(id: "order-123") {
//     id
//     customerId
//     status
//     totalAmount
//     items {
//       productId
//       quantity
//       price
//     }
//     shipment {
//       trackingNumber
//       status
//       estimatedDelivery
//       driver {
//         name
//         phone
//         currentLocation {
//           latitude
//           longitude
//         }
//         vehicleInfo {
//           make
//           model
//           licensePlate
//         }
//       }
//     }
//   }
// }
//
// # Query multiple orders with filtering
// query GetOrders {
//   orders(customerId: "customer-123", status: "pending", limit: 10) {
//     id
//     status
//     totalAmount
//     createdAt
//   }
// }
//
// # Create order mutation
// mutation CreateOrder {
//   createOrder(customerId: "customer-123", items: ["item1", "item2"]) {
//     id
//     status
//     totalAmount
//   }
// }
//
// # Update shipment status
// mutation UpdateShipment {
//   updateShipment(id: "shipment-456", status: "delivered") {
//     id
//     status
//     trackingNumber
//   }
// }
//
// # Subscribe to order updates
// subscription OrderUpdates {
//   orderUpdates(orderId: "order-123") {
//     id
//     status
//     updatedAt
//   }
// }
//
// # Complex aggregation query
// query Dashboard {
//   orders(status: "active", limit: 5) {
//     id
//     totalAmount
//     shipment {
//       status
//       driver {
//         name
//         status
//       }
//     }
//   }
//   drivers(status: "available") {
//     id
//     name
//     currentLocation {
//       latitude
//       longitude
//     }
//   }
//   inventories(warehouseId: "wh-001") {
//     productId
//     available
//   }
// }
