## Shared Packages

This directory contains reusable packages shared across all microservices in the logistics platform.

## Overview

| Package | Purpose | Usage |
|---------|---------|-------|
| `config` | Configuration management | Load env vars, validate config |
| `logger` | Structured logging | Consistent logging across services |
| `middleware` | HTTP middleware | Request ID, CORS, Recovery |
| `health` | Health checks | Kubernetes liveness/readiness probes |
| `server` | Server utilities | Graceful shutdown, signal handling |

---

## 📦 Package: `config`

**Purpose:** Centralized configuration management with environment variable loading and validation.

### Features

- Environment variable loading with defaults
- Type-safe config (int, bool, duration)
- Database configuration builder
- Server configuration
- Environment detection (dev/staging/prod)

### Usage Example

```go
import "github.com/joshbarros/golang-logistics-services/pkg/config"

// Load database configuration
dbConfig := config.LoadDatabaseConfig("order")  // Creates order_db
if err := dbConfig.Validate(); err != nil {
    log.Fatal(err)
}

// Get DSN string
dsn := dbConfig.DSN()
// Output: host=localhost user=postgres password=*** dbname=order_db port=5432 sslmode=disable

// Load server configuration
serverConfig := config.LoadServerConfig()
fmt.Println(serverConfig.Port)  // "8080" or HTTP_PORT env var

// Check environment
if config.IsProduction() {
    // Production-specific logic
}
```

### Environment Variables

```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=secret
DB_NAME=driver_db
DB_SSLMODE=disable

# Server
HTTP_PORT=8080
HTTP_READ_TIMEOUT=15s
HTTP_WRITE_TIMEOUT=15s
HTTP_SHUTDOWN_TIMEOUT=30s

# Service
SERVICE_NAME=driver-service
SERVICE_VERSION=v1.2.3
ENVIRONMENT=production  # development, staging, production
```

---

## 📦 Package: `logger`

**Purpose:** Structured logging with levels, fields, and Gin middleware integration.

### Features

- Log levels: DEBUG, INFO, WARN, ERROR, FATAL
- Structured fields (key-value pairs)
- Gin middleware for request/response logging
- Request ID integration
- Configurable output

### Usage Example

```go
import "github.com/joshbarros/golang-logistics-services/pkg/logger"

// Create logger
log := logger.New("order-service")
log.SetLevel(logger.FromEnv())  // Set from LOG_LEVEL env var

// Basic logging
log.Info("Order created", logger.Fields{
    "order_id": "ord-123",
    "customer_id": "cust-456",
    "total": 99.99,
})

// Error logging
log.Error("Database error", logger.Fields{
    "error": err.Error(),
    "query": "SELECT * FROM orders",
})

// Use as Gin middleware
router := gin.New()
router.Use(log.GinMiddleware())
```

### Log Output Format

```
[2025-11-19T10:30:45Z] INFO service=order-service message="Order created" order_id=ord-123 customer_id=cust-456 total=99.99
[2025-11-19T10:30:46Z] INFO service=order-service message="Request completed" method=POST path=/api/v1/orders request_id=abc-123 status=201 duration_ms=45 client_ip=192.168.1.1
[2025-11-19T10:30:47Z] ERROR service=order-service message="Database error" error="connection refused" query="SELECT * FROM orders"
```

### Environment Variables

```bash
LOG_LEVEL=INFO  # DEBUG, INFO, WARN, ERROR, FATAL
```

---

## 📦 Package: `middleware`

**Purpose:** HTTP middleware for cross-cutting concerns.

### Available Middleware

#### 1. Request ID

Generates unique request ID for distributed tracing.

```go
import "github.com/joshbarros/golang-logistics-services/pkg/middleware"

router.Use(middleware.RequestID())

// Access request ID in handlers
func handler(c *gin.Context) {
    requestID := middleware.GetRequestID(c)
    log.Printf("Processing request: %s", requestID)
}
```

**Headers:**
- Reads: `X-Request-ID` (if provided by client)
- Sets: `X-Request-ID` (in response)

#### 2. Recovery

Recovers from panics and returns 500 error with stack trace.

```go
router.Use(middleware.Recovery(log))
```

**Features:**
- Catches panics
- Logs stack trace
- Returns JSON error with request ID
- Prevents service crash

#### 3. CORS

Configurable CORS middleware with security defaults.

```go
// Use default config
router.Use(middleware.CORS(middleware.DefaultCORSConfig()))

// Or customize
corsConfig := middleware.CORSConfig{
    AllowOrigins:     []string{"https://yourdomain.com"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders:     []string{"Content-Type", "Authorization"},
    ExposeHeaders:    []string{"X-Request-ID"},
    AllowCredentials: true,
    MaxAge:           3600,
}
router.Use(middleware.CORS(corsConfig))
```

**Default Config:**
- Origins: `http://localhost:3000`, `http://localhost:8000`
- Methods: GET, POST, PUT, PATCH, DELETE, OPTIONS
- Headers: Origin, Content-Type, Accept, Authorization, X-Request-ID
- Credentials: Allowed
- Max Age: 3600 seconds

---

## 📦 Package: `health`

**Purpose:** Health checks for Kubernetes liveness/readiness probes and monitoring.

### Features

- Database connectivity checks
- Connection pool monitoring
- Latency measurement
- Status levels: healthy, degraded, unhealthy
- Kubernetes-compatible endpoints

### Usage Example

```go
import "github.com/joshbarros/golang-logistics-services/pkg/health"

// Create health checker
healthChecker := health.NewChecker("order-service", "v1.2.3", db)

// Register endpoints
router.GET("/health", healthChecker.Check)
router.GET("/ready", health.ReadinessHandler("order-service"))
router.GET("/live", health.LivenessHandler("order-service"))

// Configure connection pool
sqlDB, _ := db.DB()
health.ConfigureDBConnectionPool(sqlDB)
```

### Health Check Response

```json
{
  "status": "healthy",
  "service": "order-service",
  "version": "v1.2.3",
  "checks": {
    "api": {
      "status": "healthy",
      "message": "API is responding"
    },
    "database": {
      "status": "healthy",
      "message": "Database connection is healthy",
      "latency": "2.5ms"
    }
  }
}
```

### Status Codes

- **200 OK**: Service is healthy or degraded
- **503 Service Unavailable**: Service is unhealthy

### Kubernetes Integration

```yaml
# Deployment spec
livenessProbe:
  httpGet:
    path: /live
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5

# Detailed health check for monitoring
startup Probe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 0
  periodSeconds: 10
  failureThreshold: 30
```

### Connection Pool Configuration

The package configures optimal database connection pool settings:

```go
health.ConfigureDBConnectionPool(sqlDB)
```

Settings:
- Max open connections: 25
- Max idle connections: 5
- Connection max lifetime: 5 minutes
- Connection max idle time: 10 minutes

---

## 📦 Package: `server`

**Purpose:** HTTP server utilities with graceful shutdown and signal handling.

### Features

- Graceful shutdown on SIGINT/SIGTERM
- Configurable shutdown timeout
- Shutdown hooks for cleanup
- HTTP server lifecycle management

### Usage Example

```go
import "github.com/joshbarros/golang-logistics-services/pkg/server"

// Create HTTP server
srv := &http.Server{
    Addr:         ":8080",
    Handler:      router,
    ReadTimeout:  15 * time.Second,
    WriteTimeout: 15 * time.Second,
}

// Setup shutdown manager
shutdownManager := server.NewShutdownManager()

// Add cleanup hooks
shutdownManager.AddHook(func() error {
    log.Info("Closing database connections")
    return sqlDB.Close()
})

shutdownManager.AddHook(func() error {
    log.Info("Flushing metrics")
    return metricsClient.Flush()
})

// Run with graceful shutdown
shutdownConfig := server.DefaultShutdownConfig()
err := server.RunWithGracefulShutdown(srv, shutdownConfig, shutdownManager.Shutdown)
if err != nil {
    log.Fatal(err)
}
```

### Graceful Shutdown Flow

1. Signal received (SIGINT/SIGTERM)
2. Stop accepting new connections
3. Wait for active requests to complete (up to timeout)
4. Execute shutdown hooks in order
5. Exit cleanly

### Shutdown Configuration

```go
config := server.GracefulShutdownConfig{
    Timeout: 30 * time.Second,
    Signals: []os.Signal{syscall.SIGINT, syscall.SIGTERM},
}
```

---

## 🚀 Complete Service Example

Here's how all packages work together in a service:

```go
package main

import (
    "fmt"
    "net/http"

    "github.com/gin-gonic/gin"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"

    "github.com/joshbarros/golang-logistics-services/pkg/config"
    "github.com/joshbarros/golang-logistics-services/pkg/health"
    "github.com/joshbarros/golang-logistics-services/pkg/logger"
    "github.com/joshbarros/golang-logistics-services/pkg/middleware"
    "github.com/joshbarros/golang-logistics-services/pkg/server"
)

func main() {
    // 1. Initialize logger
    log := logger.New("myservice")
    log.SetLevel(logger.FromEnv())
    log.Info("Starting service", nil)

    // 2. Load configuration
    dbConfig := config.LoadDatabaseConfig("myservice")
    serverConfig := config.LoadServerConfig()
    version := config.GetServiceVersion()

    // 3. Connect to database
    db, err := gorm.Open(postgres.Open(dbConfig.DSN()), &gorm.Config{})
    if err != nil {
        log.Fatal("Database connection failed", logger.Fields{"error": err.Error()})
    }

    // 4. Configure connection pool
    sqlDB, _ := db.DB()
    health.ConfigureDBConnectionPool(sqlDB)

    // 5. Setup router with middleware
    router := gin.New()
    router.Use(middleware.RequestID())
    router.Use(middleware.Recovery(log))
    router.Use(log.GinMiddleware())
    router.Use(middleware.CORS(middleware.DefaultCORSConfig()))

    // 6. Add health checks
    healthChecker := health.NewChecker("myservice", version, db)
    router.GET("/health", healthChecker.Check)
    router.GET("/ready", health.ReadinessHandler("myservice"))
    router.GET("/live", health.LivenessHandler("myservice"))

    // 7. Add your routes
    router.GET("/api/v1/hello", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "Hello!"})
    })

    // 8. Create HTTP server
    srv := &http.Server{
        Addr:         fmt.Sprintf(":%s", serverConfig.Port),
        Handler:      router,
        ReadTimeout:  serverConfig.ReadTimeout,
        WriteTimeout: serverConfig.WriteTimeout,
    }

    // 9. Setup graceful shutdown
    shutdownManager := server.NewShutdownManager()
    shutdownManager.AddHook(func() error {
        log.Info("Closing database", nil)
        return sqlDB.Close()
    })

    // 10. Run server
    log.Info("Server starting", logger.Fields{"port": serverConfig.Port})
    shutdownConfig := server.DefaultShutdownConfig()
    err = server.RunWithGracefulShutdown(srv, shutdownConfig, shutdownManager.Shutdown)
    if err != nil {
        log.Fatal("Server error", logger.Fields{"error": err.Error()})
    }

    log.Info("Service stopped", nil)
}
```

---

## 🔐 Security Best Practices

### 1. CORS Configuration

**Development:**
```go
corsConfig := middleware.CORSConfig{
    AllowOrigins: []string{"http://localhost:3000"},
    AllowCredentials: true,
}
```

**Production:**
```go
corsConfig := middleware.CORSConfig{
    AllowOrigins: []string{"https://yourdomain.com"},
    AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},  // No OPTIONS if not needed
    AllowCredentials: true,
    MaxAge: 86400,  // 24 hours
}
```

**Never use in production:**
```go
AllowOrigins: []string{"*"}  // ❌ Insecure!
```

### 2. Database Configuration

Store credentials in Kubernetes secrets:

```yaml
# k8s/secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: postgres-secret
type: Opaque
data:
  username: <base64-encoded>
  password: <base64-encoded>
```

Reference in deployment:

```yaml
env:
  - name: DB_USER
    valueFrom:
      secretKeyRef:
        name: postgres-secret
        key: username
  - name: DB_PASSWORD
    valueFrom:
      secretKeyRef:
        name: postgres-secret
        key: password
```

### 3. Request ID Tracking

Always use request IDs for:
- Distributed tracing
- Log correlation
- Debugging across services
- Customer support tickets

---

## 📊 Observability

### Logging

All services log in structured format:
```
[timestamp] LEVEL service=name message="text" key1=value1 key2=value2
```

### Metrics (To Be Added)

Future enhancement: Prometheus metrics

```go
// Coming soon
metrics.RequestDuration.Observe(duration)
metrics.RequestsTotal.Inc()
metrics.ErrorsTotal.Inc()
```

### Tracing (To Be Added)

Future enhancement: OpenTelemetry integration

---

## 🧪 Testing

### Testing Logger

```go
func TestLogger(t *testing.T) {
    var buf bytes.Buffer
    log := logger.New("test-service")
    log.SetOutput(&buf)
    log.SetLevel(logger.INFO)

    log.Info("test message", logger.Fields{"key": "value"})

    output := buf.String()
    assert.Contains(t, output, "test message")
    assert.Contains(t, output, "key=value")
}
```

### Testing Middleware

```go
func TestRequestIDMiddleware(t *testing.T) {
    router := gin.New()
    router.Use(middleware.RequestID())
    router.GET("/test", func(c *gin.Context) {
        requestID := middleware.GetRequestID(c)
        c.JSON(200, gin.H{"request_id": requestID})
    })

    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/test", nil)
    router.ServeHTTP(w, req)

    assert.Equal(t, 200, w.Code)
    assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
}
```

---

## 📚 Additional Resources

- [Gin Web Framework](https://gin-gonic.com/)
- [GORM Documentation](https://gorm.io/)
- [Kubernetes Health Checks](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
- [Twelve-Factor App](https://12factor.net/)
- [Structured Logging Best Practices](https://www.loggly.com/ultimate-guide/go-logging-best-practices/)
