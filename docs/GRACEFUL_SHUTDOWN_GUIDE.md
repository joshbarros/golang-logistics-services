# Graceful Shutdown Implementation Guide

## Overview

This guide provides instructions for implementing comprehensive graceful shutdown in all microservices using the `pkg/shutdown` package.

## Why Graceful Shutdown is Critical

Without proper graceful shutdown:
- ❌ In-flight requests are terminated mid-process
- ❌ Database connections close abruptly (potential data corruption)
- ❌ Message queue messages are lost
- ❌ Partial writes leave data in inconsistent state
- ❌ Clients receive 5xx errors instead of graceful degradation
- ❌ Load balancers send traffic to terminating pods

With proper graceful shutdown:
- ✅ New requests are rejected immediately
- ✅ In-flight requests complete (with timeout)
- ✅ Database connections drain gracefully
- ✅ Message queues flush pending messages
- ✅ Resources are released properly
- ✅ Zero downtime deployments possible

## The Shutdown Package

Location: `pkg/shutdown/shutdown.go`

### Key Features

1. **Priority-Based Shutdown**: Components shut down in the correct order
2. **Timeout Protection**: Won't hang forever (default 30s)
3. **Signal Handling**: Responds to SIGINT, SIGTERM
4. **Comprehensive Logging**: Tracks shutdown progress
5. **Error Aggregation**: Reports all errors, not just the first
6. **Helper Methods**: For common components (HTTP, gRPC, DB, etc.)

### Priority Order

Lower number = higher priority (shuts down first):

```
Priority 5:  Worker pools (stop accepting new jobs)
Priority 10: HTTP/gRPC servers (stop accepting new requests)
Priority 15: Message brokers (flush pending messages)
Priority 20: Databases (drain connections)
Priority 25: Caches (flush writes)
Priority 30: Custom cleanup
```

## Implementation Pattern

### Basic Setup (3 steps)

#### Step 1: Import the package

```go
import (
    "github.com/sirupsen/logrus"
    "github.com/joshbarros/golang-logistics-services/pkg/shutdown"
)
```

#### Step 2: Create shutdown manager

```go
logger := logrus.New()
logger.SetFormatter(&logrus.JSONFormatter{})

shutdownMgr := shutdown.NewManager(shutdown.Config{
    Logger:          logger,
    ShutdownTimeout: 30 * time.Second, // Customize if needed
})
```

#### Step 3: Register components

```go
// HTTP Server
shutdownMgr.RegisterHTTPServer("my-http-server", httpServer)

// Database
shutdownMgr.RegisterDatabase("postgres", sqlDB)

// Message Broker
shutdownMgr.RegisterMessageBroker("rabbitmq", rabbitmqConn)

// Cache
shutdownMgr.RegisterCache("redis", redisClient)

// Worker Pool
shutdownMgr.RegisterWorkerPool("background-workers", workerPool)

// Custom cleanup
shutdownMgr.RegisterCustom("custom-component", 30, func(ctx context.Context) error {
    // Your cleanup logic here
    return nil
})
```

#### Step 4: Wait for shutdown

```go
// Start server in background
go func() {
    if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatalf("Failed to start server: %v", err)
    }
}()

// Block until shutdown signal received
if err := shutdownMgr.Wait(); err != nil {
    log.Fatalf("Shutdown error: %v", err)
}

log.Println("Service exited gracefully")
```

## Service-Specific Examples

### Gateway Service (HTTP Only)

**File**: `services/gateway/main.go`

```go
func main() {
    // ... setup code ...

    srv := &http.Server{
        Addr:    ":8080",
        Handler: router,
    }

    logger := logrus.New()
    shutdownMgr := shutdown.New(logger)

    // Register components
    shutdownMgr.RegisterHTTPServer("gateway-http", srv)
    shutdownMgr.Register("metrics", 15, func(ctx context.Context) error {
        return metricsManager.Close()
    })
    shutdownMgr.Register("tracing", 20, func(ctx context.Context) error {
        return tracingManager.Close()
    })

    // Start and wait
    go func() {
        log.Printf("Starting gateway on port 8080")
        srv.ListenAndServe()
    }()

    if err := shutdownMgr.Wait(); err != nil {
        log.Fatalf("Shutdown error: %v", err)
    }
}
```

### Order Service (HTTP + Database + Message Broker)

**File**: `services/order-service/cmd/main.go`

```go
func main() {
    // ... setup code ...

    // Database
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    sqlDB, _ := db.DB()

    // Message broker
    rabbitmqConn, err := amqp.Dial("amqp://localhost:5672")
    rabbitmqCh, err := rabbitmqConn.Channel()

    // Redis cache
    redisClient := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })

    // HTTP server
    srv := &http.Server{
        Addr:    ":8081",
        Handler: router,
    }

    // Graceful shutdown manager
    logger := logrus.New()
    shutdownMgr := shutdown.New(logger)

    // Register all components (priority order)
    shutdownMgr.RegisterHTTPServer("order-http", srv)
    shutdownMgr.RegisterMessageBroker("rabbitmq", rabbitmqConn)
    shutdownMgr.RegisterDatabase("postgres", sqlDB)
    shutdownMgr.RegisterCache("redis", redisClient)

    // Custom cleanup for channel
    shutdownMgr.Register("rabbitmq-channel", 16, func(ctx context.Context) error {
        return rabbitmqCh.Close()
    })

    // Start server
    go func() {
        log.Printf("Starting order service on port 8081")
        srv.ListenAndServe()
    }()

    // Wait for shutdown
    if err := shutdownMgr.Wait(); err != nil {
        log.Fatalf("Shutdown error: %v", err)
    }

    log.Println("Order service exited gracefully")
}
```

### Notification Service (HTTP + Database + Worker Pool)

**File**: `services/notification-service/cmd/main.go`

```go
func main() {
    // ... setup code ...

    // Database
    db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    sqlDB, _ := db.DB()

    // Background worker pool for sending notifications
    workerPool := worker.NewPool(worker.Config{
        NumWorkers:  10,
        QueueSize:   1000,
    })
    workerPool.Start()

    // HTTP server
    srv := &http.Server{
        Addr:    ":8085",
        Handler: router,
    }

    // Graceful shutdown manager
    logger := logrus.New()
    shutdownMgr := shutdown.New(logger)

    // Register components
    shutdownMgr.RegisterWorkerPool("notification-workers", workerPool) // Priority 5
    shutdownMgr.RegisterHTTPServer("notification-http", srv)           // Priority 10
    shutdownMgr.RegisterDatabase("postgres", sqlDB)                    // Priority 20

    // Start server
    go func() {
        log.Printf("Starting notification service on port 8085")
        srv.ListenAndServe()
    }()

    // Wait for shutdown
    if err := shutdownMgr.Wait(); err != nil {
        log.Fatalf("Shutdown error: %v", err)
    }

    log.Println("Notification service exited gracefully")
}
```

## Migration Checklist

For each service:

### 1. Update Imports
- [ ] Add `"github.com/sirupsen/logrus"`
- [ ] Add `"github.com/joshbarros/golang-logistics-services/pkg/shutdown"`
- [ ] Remove manual `"os/signal"` and `"syscall"` if using basic shutdown

### 2. Create Shutdown Manager
- [ ] Initialize logger
- [ ] Create shutdown manager with timeout
- [ ] Register HTTP/gRPC server
- [ ] Register database connections
- [ ] Register message brokers
- [ ] Register caches
- [ ] Register worker pools
- [ ] Add custom cleanup if needed

### 3. Replace Existing Shutdown Code
- [ ] Remove manual signal handling
- [ ] Remove manual `server.Shutdown()` calls
- [ ] Remove manual `context.WithTimeout()` for shutdown
- [ ] Replace with `shutdownMgr.Wait()`

### 4. Test
- [ ] Service starts normally
- [ ] Service responds to requests
- [ ] Send SIGTERM: `kill -TERM <pid>`
- [ ] Verify logs show orderly shutdown
- [ ] Verify no errors in shutdown logs
- [ ] Verify all resources released

## Testing Graceful Shutdown

### Manual Testing

```bash
# Start service
go run services/order-service/cmd/main.go

# In another terminal, get PID
ps aux | grep order-service

# Send SIGTERM
kill -TERM <pid>

# Check logs for:
# ✅ "Received shutdown signal"
# ✅ "Shutting down component..." for each component
# ✅ "Component shut down successfully"
# ✅ "Graceful shutdown completed successfully"
```

### Automated Testing

```bash
# Test script
./scripts/test-graceful-shutdown.sh order-service
```

### Load Testing During Shutdown

```bash
# Start load test
k6 run tests/load/order-service.js &

# Wait 10 seconds
sleep 10

# Trigger shutdown
kill -TERM <pid>

# Check results:
# ✅ 0 failed requests (all in-flight completed)
# ✅ New requests after shutdown receive 503
# ✅ No database errors
```

## Kubernetes Integration

### Proper Pod Spec

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: order-service
spec:
  containers:
  - name: order-service
    image: order-service:latest
    lifecycle:
      preStop:
        exec:
          command: ["/bin/sh", "-c", "sleep 5"]  # Allow time for load balancer to update
    terminationGracePeriodSeconds: 35  # Must be > shutdown timeout (30s) + preStop sleep (5s)

  # Liveness probe (restart if unhealthy)
  livenessProbe:
    httpGet:
      path: /health
      port: 8081
    initialDelaySeconds: 30
    periodSeconds: 10
    failureThreshold: 3

  # Readiness probe (remove from load balancer if not ready)
  readinessProbe:
    httpGet:
      path: /ready
      port: 8081
    initialDelaySeconds: 5
    periodSeconds: 5
    failureThreshold: 2
```

### Deployment Strategy

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: order-service
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1         # Add 1 extra pod during update
      maxUnavailable: 0   # Never have fewer than desired replicas

  # Pod template spec from above
```

## Common Pitfalls

### ❌ Wrong: No timeout

```go
// Hangs forever if component doesn't respond
srv.Shutdown(context.Background())
```

### ✅ Right: With timeout

```go
// Shutdown manager automatically applies timeout
shutdownMgr.RegisterHTTPServer("my-server", srv)
```

### ❌ Wrong: Wrong order

```go
// Database closes before HTTP server finishes requests!
db.Close()
srv.Shutdown(ctx)
```

### ✅ Right: Priority-based

```go
// HTTP server stops first (priority 10), database closes after (priority 20)
shutdownMgr.RegisterHTTPServer("http", srv)      // Priority 10
shutdownMgr.RegisterDatabase("postgres", sqlDB)  // Priority 20
```

### ❌ Wrong: Silent failures

```go
if err := db.Close(); err != nil {
    // Error swallowed, never logged
}
```

### ✅ Right: Error aggregation

```go
// Shutdown manager logs all errors and aggregates them
shutdownMgr.RegisterDatabase("postgres", sqlDB)
// Errors automatically logged and returned
```

### ❌ Wrong: No signal handling

```go
func main() {
    srv.ListenAndServe()  // Blocks forever, can't be gracefully stopped
}
```

### ✅ Right: Signal handling

```go
func main() {
    go srv.ListenAndServe()
    shutdownMgr.Wait()  // Waits for SIGINT/SIGTERM
}
```

## Monitoring Graceful Shutdown

### Metrics to Track

```go
// Add to pkg/metrics package
type ShutdownMetrics struct {
    shutdownDuration prometheus.Histogram
    shutdownErrors   prometheus.Counter
}

// Record in shutdown manager
func (m *Manager) Shutdown() error {
    start := time.Now()
    defer func() {
        duration := time.Since(start)
        metrics.RecordShutdownDuration(duration)
    }()

    // ... shutdown logic ...
}
```

### Alerts to Configure

```yaml
# Prometheus alerts
groups:
  - name: shutdown
    rules:
      - alert: SlowGracefulShutdown
        expr: shutdown_duration_seconds > 25
        for: 1m
        annotations:
          summary: "Graceful shutdown taking too long (>25s)"

      - alert: ShutdownErrors
        expr: increase(shutdown_errors_total[5m]) > 0
        annotations:
          summary: "Errors during graceful shutdown"
```

## Performance Impact

### Overhead
- **Memory**: ~1KB per service (negligible)
- **Latency**: 0ms during normal operation (only runs on shutdown)
- **CPU**: 0% during normal operation

### Benefits
- **Zero downtime deployments**: Prevents dropped requests
- **Data integrity**: Prevents partial writes
- **Customer experience**: No 5xx errors during deployments
- **Operational safety**: Rollouts can be done anytime

## Services to Update

### Current Status

| Service | Location | Database | Message Broker | Cache | Worker Pool | Status |
|---------|----------|----------|----------------|-------|-------------|--------|
| Gateway | `services/gateway/main.go` | ❌ | ❌ | ❌ | ❌ | ✅ **UPDATED** |
| Order | `services/order-service/cmd/main.go` | ✅ | ✅ | ✅ | ❌ | ⚠️ TODO |
| Shipment | `services/shipment-service/cmd/main.go` | ✅ | ✅ | ✅ | ❌ | ⚠️ TODO |
| Inventory | `services/inventory-service/cmd/main.go` | ✅ | ❌ | ✅ | ❌ | ⚠️ TODO |
| Driver | `services/driver-service/cmd/main.go` | ✅ | ❌ | ✅ | ❌ | ⚠️ TODO |
| Route | `services/route-service/cmd/main.go` | ✅ | ❌ | ✅ | ❌ | ⚠️ TODO |
| Notification | `services/notification-service/cmd/main.go` | ✅ | ❌ | ✅ | ✅ | ⚠️ TODO |

### Estimated Time

- **Per service**: 15-20 minutes
- **Total**: ~2 hours for all 6 remaining services
- **Priority**: HIGH (blocking production deployment)

## Resources

- [Google SRE Book - Graceful Degradation](https://sre.google/sre-book/handling-overload/)
- [Kubernetes Pod Lifecycle](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/)
- [Go HTTP Server Shutdown](https://pkg.go.dev/net/http#Server.Shutdown)
- [Handling Signals in Go](https://gobyexample.com/signals)

## Next Steps

1. ✅ Review this guide
2. ⚠️ Update all 6 remaining services
3. ⚠️ Test each service individually
4. ⚠️ Update Kubernetes manifests with proper termination settings
5. ⚠️ Add monitoring and alerts
6. ⚠️ Document in runbooks

---

**Created**: 2025-11-19
**Last Updated**: 2025-11-19
**Status**: Draft - Ready for implementation
