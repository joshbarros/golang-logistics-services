```markdown
# Advanced Production Features

This document details the advanced production-grade features available in the Logistics Services platform beyond the core Tier 1-3 features.

## Table of Contents

1. [Circuit Breaker Pattern](#circuit-breaker-pattern)
2. [Distributed Caching](#distributed-caching)
3. [OpenTelemetry Tracing](#opentelemetry-tracing)
4. [Retry Mechanism](#retry-mechanism)
5. [API Versioning](#api-versioning)
6. [Swagger/OpenAPI Documentation](#swaggeropenapi-documentation)
7. [Load Testing](#load-testing)

---

## Circuit Breaker Pattern

**Location:** `pkg/circuitbreaker/`

The circuit breaker pattern prevents cascading failures when downstream services are experiencing issues.

### Features

- **Three States:** Closed, Open, Half-Open
- **Automatic State Transitions** based on failure rates
- **Configurable Thresholds** for tripping
- **HTTP Client Wrapper** for external API calls
- **Manager** for multiple circuit breakers

### Basic Usage

```go
import "github.com/joshbarros/golang-logistics-services/pkg/circuitbreaker"

// Create circuit breaker
cb := circuitbreaker.NewCircuitBreaker("external-api", circuitbreaker.Config{
    MaxRequests: 3,                    // Max requests in half-open state
    Interval:    60 * time.Second,     // Reset interval
    Timeout:     30 * time.Second,     // Open state timeout
    ReadyToTrip: func(counts circuitbreaker.Counts) bool {
        return counts.ConsecutiveFailures >= 5
    },
    OnStateChange: func(name string, from, to circuitbreaker.State) {
        log.Printf("Circuit %s: %s -> %s", name, from, to)
    },
})

// Execute with circuit breaker
err := cb.Execute(func() error {
    return apiClient.DoSomething()
})

if err == circuitbreaker.ErrCircuitOpen {
    // Circuit is open, use fallback
    return useCachedData()
}
```

### HTTP Client with Circuit Breaker

```go
// Create HTTP client with circuit breaker protection
client := circuitbreaker.NewHTTPClient(
    "payment-gateway",
    circuitbreaker.Config{
        MaxRequests: 3,
        Timeout:     30 * time.Second,
        ReadyToTrip: func(counts circuitbreaker.Counts) bool {
            failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
            return counts.Requests >= 10 && failureRatio >= 0.5
        },
    },
    10*time.Second, // HTTP timeout
)

// Use like normal http.Client
resp, err := client.Get("https://api.payment.com/process")
if err != nil {
    if errors.Is(err, circuitbreaker.ErrCircuitOpen) {
        log.Println("Payment gateway circuit is open")
        return handlePaymentUnavailable()
    }
    return err
}
```

### Circuit Breaker Manager

```go
// Create manager for multiple services
manager := circuitbreaker.NewCircuitBreakerManager(circuitbreaker.DefaultConfig())

// Get circuit breaker for a service (auto-creates if needed)
paymentCB := manager.GetCircuitBreaker("payment-service")
inventoryCB := manager.GetCircuitBreaker("inventory-service")

// Get all circuit breaker states
states := manager.GetAllStates()
// Returns: map[string]State{"payment-service": StateOpen, "inventory-service": StateClosed}

// Get all counts
counts := manager.GetAllCounts()
```

### Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| MaxRequests | Max requests allowed in half-open state | 1 |
| Interval | Time window for counting failures in closed state | 60s |
| Timeout | Duration before transitioning from open to half-open | 60s |
| ReadyToTrip | Function to determine when to open circuit | 5 consecutive failures |
| OnStateChange | Callback when state changes | nil |

### Best Practices

1. **Set Appropriate Thresholds** - Balance between sensitivity and stability
2. **Monitor Circuit States** - Expose circuit breaker metrics
3. **Implement Fallbacks** - Always have a fallback when circuit opens
4. **Use Per-Service Circuits** - Isolate failures to specific services
5. **Log State Changes** - Track when circuits open/close

---

## Distributed Caching

**Location:** `pkg/cache/`

High-performance caching layer with Redis and in-memory fallback.

### Features

- **Redis-based** distributed caching
- **In-memory fallback** when Redis unavailable
- **JSON serialization** for complex types
- **TTL support** for expiration
- **Pattern-based invalidation**
- **Cache-aside pattern** helper

### Basic Usage

```go
import "github.com/joshbarros/golang-logistics-services/pkg/cache"

// Create Redis cache
redisCache, err := cache.NewRedisCache(
    "localhost:6379",  // Redis address
    "",                // Password
    "order-service",   // Key prefix
    0,                 // DB number
)
if err != nil {
    // Fallback to in-memory cache
    redisCache = cache.NewInMemoryCache(5 * time.Minute)
}
defer redisCache.Close()

// Store in cache
order := &Order{ID: "123", Total: 99.99}
err = redisCache.Set(ctx, "order:123", order, 5*time.Minute)

// Retrieve from cache
var cachedOrder Order
err = redisCache.Get(ctx, "order:123", &cachedOrder)
if err == cache.ErrCacheMiss {
    // Cache miss, fetch from database
}

// Delete from cache
err = redisCache.Delete(ctx, "order:123")

// Check existence
count, err := redisCache.Exists(ctx, "order:123", "order:456")

// Invalidate pattern
err = redisCache.Invalidate(ctx, "order:*")
```

### Cache-Aside Pattern

```go
wrapper := cache.NewCacheWrapper(redisCache)

// Get from cache or execute function
var order Order
err := wrapper.GetOrSet(
    ctx,
    fmt.Sprintf("order:%s", orderID),
    &order,
    5*time.Minute,
    func() (interface{}, error) {
        // This function is only called on cache miss
        return orderRepo.GetByID(orderID)
    },
)
```

### In-Memory Cache

```go
// Create in-memory cache with cleanup interval
memCache := cache.NewInMemoryCache(5 * time.Minute)
defer memCache.Close()

// Use same interface as Redis cache
memCache.Set(ctx, "key", value, 10*time.Minute)
memCache.Get(ctx, "key", &dest)
```

### Caching Strategies

#### 1. Cache-Aside (Lazy Loading)

```go
func (s *OrderService) GetOrder(ctx context.Context, id string) (*Order, error) {
    var order Order

    // Try cache first
    err := s.cache.Get(ctx, fmt.Sprintf("order:%s", id), &order)
    if err == nil {
        return &order, nil
    }

    // Cache miss, fetch from DB
    order, err = s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }

    // Store in cache (fire and forget)
    go s.cache.Set(context.Background(), fmt.Sprintf("order:%s", id), order, 5*time.Minute)

    return &order, nil
}
```

#### 2. Write-Through

```go
func (s *OrderService) UpdateOrder(ctx context.Context, order *Order) error {
    // Update database
    err := s.repo.Update(ctx, order)
    if err != nil {
        return err
    }

    // Update cache
    err = s.cache.Set(ctx, fmt.Sprintf("order:%s", order.ID), order, 5*time.Minute)
    if err != nil {
        log.Printf("Failed to update cache: %v", err)
        // Continue - cache update failure shouldn't fail the operation
    }

    return nil
}
```

#### 3. Write-Behind (Write-Back)

```go
func (s *OrderService) CreateOrder(ctx context.Context, order *Order) error {
    // Write to cache immediately
    err := s.cache.Set(ctx, fmt.Sprintf("order:%s", order.ID), order, 5*time.Minute)
    if err != nil {
        return err
    }

    // Async write to database
    go func() {
        if err := s.repo.Create(context.Background(), order); err != nil {
            log.Printf("Failed to persist order to DB: %v", err)
            // Handle failure (retry, alert, etc.)
        }
    }()

    return nil
}
```

### Cache Invalidation Patterns

```go
// 1. Time-based expiration (set TTL)
cache.Set(ctx, "order:123", order, 5*time.Minute)

// 2. Event-based invalidation
func (s *OrderService) UpdateOrder(ctx context.Context, order *Order) error {
    err := s.repo.Update(ctx, order)
    if err != nil {
        return err
    }

    // Invalidate cache on update
    s.cache.Delete(ctx, fmt.Sprintf("order:%s", order.ID))

    return nil
}

// 3. Pattern-based invalidation
func (s *OrderService) InvalidateUserOrders(ctx context.Context, userID string) error {
    return s.cache.Invalidate(ctx, fmt.Sprintf("order:user:%s:*", userID))
}
```

### Performance Considerations

- **Cache Key Design** - Use hierarchical keys: `service:entity:id`
- **TTL Strategy** - Short TTL for frequently changing data, long TTL for static data
- **Cache Size** - Monitor memory usage, set appropriate limits
- **Serialization** - JSON is flexible but slower than msgpack/protobuf
- **Network Latency** - Redis adds network hop, only cache expensive operations

---

## OpenTelemetry Tracing

**Location:** `pkg/tracing/`

Distributed tracing with OpenTelemetry and Jaeger integration.

### Features

- **Automatic HTTP tracing** via Gin middleware
- **Custom span creation** for business logic
- **Context propagation** across services
- **Jaeger exporter** for visualization
- **Sampling support** to control overhead
- **Attribute tagging** for rich traces

### Setup

```go
import "github.com/joshbarros/golang-logistics-services/pkg/tracing"

// Initialize tracing
tracingMgr, err := tracing.NewTracingManager(tracing.Config{
    ServiceName:    "order-service",
    ServiceVersion: "2.0.0",
    Environment:    "production",
    JaegerEndpoint: "http://localhost:14268/api/traces",
    SamplingRate:   0.1,  // Sample 10% of traces
    Enabled:        true,
})
if err != nil {
    log.Fatal(err)
}
defer tracingMgr.Shutdown(context.Background())

// Add middleware to Gin
router := gin.New()
router.Use(tracingMgr.Middleware())
```

### Creating Custom Spans

```go
func (s *OrderService) ProcessOrder(ctx context.Context, order *Order) error {
    // Create span for business logic
    ctx, span := s.tracing.StartSpan(ctx, "process_order")
    defer span.End()

    // Add attributes
    tracing.SetAttributes(ctx,
        attribute.String("order.id", order.ID),
        attribute.Float64("order.total", order.Total),
        attribute.Int("order.item_count", len(order.Items)),
    )

    // Validate order
    ctx, validateSpan := s.tracing.StartSpan(ctx, "validate_order")
    if err := s.validateOrder(ctx, order); err != nil {
        tracing.RecordError(ctx, err)
        validateSpan.End()
        return err
    }
    validateSpan.End()

    // Calculate pricing
    ctx, pricingSpan := s.tracing.StartSpan(ctx, "calculate_pricing")
    total, err := s.calculateTotal(ctx, order)
    pricingSpan.End()
    if err != nil {
        tracing.RecordError(ctx, err)
        return err
    }

    // Add event
    tracing.AddEvent(ctx, "pricing_calculated",
        attribute.Float64("total", total),
    )

    return nil
}
```

### Tracing Database Queries

```go
func (r *OrderRepository) GetByID(ctx context.Context, id string) (*Order, error) {
    ctx, span := tracing.TraceDBQuery(ctx, "SELECT * FROM orders WHERE id = $1", id)
    defer span.End()

    var order Order
    err := r.db.WithContext(ctx).First(&order, "id = ?", id).Error
    if err != nil {
        tracing.RecordError(ctx, err)
        return nil, err
    }

    return &order, nil
}
```

### Tracing gRPC Calls

```go
func (c *ShipmentClient) CreateShipment(ctx context.Context, req *pb.CreateShipmentRequest) (*pb.Shipment, error) {
    ctx, span := tracing.TraceGRPCCall(ctx, "shipment.ShipmentService/CreateShipment")
    defer span.End()

    resp, err := c.client.CreateShipment(ctx, req)
    if err != nil {
        tracing.RecordError(ctx, err)
        return nil, err
    }

    return resp, nil
}
```

### Viewing Traces

1. **Start Jaeger** (local development):
```bash
docker run -d --name jaeger \
  -p 5775:5775/udp \
  -p 6831:6831/udp \
  -p 6832:6832/udp \
  -p 5778:5778 \
  -p 16686:16686 \
  -p 14268:14268 \
  -p 14250:14250 \
  -p 9411:9411 \
  jaegertracing/all-in-one:latest
```

2. **Access Jaeger UI**: http://localhost:16686

3. **Search Traces**:
   - Filter by service name
   - Filter by operation
   - Filter by tags (order.id, user.id, etc.)
   - View trace timeline and spans

### Trace IDs in Logs

```go
// Include trace ID in log messages
func (h *OrderHandler) CreateOrder(c *gin.Context) {
    ctx := c.Request.Context()
    traceID := tracing.TraceID(ctx)

    log.Printf("[trace_id=%s] Creating order", traceID)

    // ... handler logic
}
```

---

## Retry Mechanism

**Location:** `pkg/retry/`

Robust retry mechanism with exponential backoff and jitter.

### Features

- **Exponential backoff** to avoid overwhelming downstream services
- **Jitter** to prevent thundering herd
- **Configurable retry attempts**
- **Context support** for cancellation
- **Selective retry** based on error types
- **Retry callbacks** for logging

### Basic Usage

```go
import "github.com/joshbarros/golang-logistics-services/pkg/retry"

// Simple retry
err := retry.Do(ctx, retry.Config{
    MaxAttempts:  5,
    InitialDelay: 100 * time.Millisecond,
    MaxDelay:     5 * time.Second,
    Multiplier:   2.0,
    Jitter:       true,
    OnRetry: func(attempt int, err error, delay time.Duration) {
        log.Printf("Retry %d after error: %v (waiting %v)", attempt, err, delay)
    },
}, func() error {
    return apiClient.DoOperation()
})
```

### Retry with Return Value

```go
order, err := retry.DoWithValue(ctx, retry.DefaultConfig(), func() (*Order, error) {
    return orderRepo.GetByID("order-123")
})
```

### Selective Retry

```go
// Only retry specific errors
err := retry.Do(ctx, retry.Config{
    MaxAttempts:     3,
    InitialDelay:    100 * time.Millisecond,
    MaxDelay:        1 * time.Second,
    Multiplier:      2.0,
    RetryableErrors: []error{
        retry.ErrTemporaryFailure,
        retry.ErrTimeout,
        retry.ErrConnectionFailed,
    },
}, func() error {
    return database.Query()
})
```

### Reusable Retrier

```go
// Create retrier once, reuse multiple times
retrier := retry.NewRetrier(retry.Config{
    MaxAttempts:  3,
    InitialDelay: 50 * time.Millisecond,
    MaxDelay:     1 * time.Second,
    Multiplier:   2.0,
    Jitter:       true,
})

// Use in multiple places
err1 := retrier.Do(ctx, func() error {
    return operation1()
})

result, err2 := retrier.DoWithValue(ctx, func() (*Result, error) {
    return operation2()
})
```

### Retry with Circuit Breaker

```go
// Combine retry and circuit breaker
cb := circuitbreaker.NewCircuitBreaker("api", circuitbreaker.DefaultConfig())
retrier := retry.NewRetrier(retry.DefaultConfig())

err := cb.Execute(func() error {
    return retrier.Do(ctx, func() error {
        return externalAPI.Call()
    })
})
```

### Backoff Calculation

With `Multiplier: 2.0` and `InitialDelay: 100ms`:

| Attempt | Delay (without jitter) | Delay (with 10% jitter) |
|---------|------------------------|-------------------------|
| 1 | 100ms | 100-110ms |
| 2 | 200ms | 200-220ms |
| 3 | 400ms | 400-440ms |
| 4 | 800ms | 800-880ms |
| 5 | 1600ms | 1600-1760ms |

---

## API Versioning

**Location:** `pkg/versioning/`

Flexible API versioning with multiple strategies.

### Features

- **Multiple strategies**: Path, Header, Query Parameter, Accept Header
- **Version validation** with min/max support
- **Deprecation warnings** for old versions
- **Automatic version detection**
- **Versioned router groups**

### Path-Based Versioning

```go
import "github.com/joshbarros/golang-logistics-services/pkg/versioning"

router := gin.New()

versionedRouter := versioning.NewVersionedRouter(router, versioning.Config{
    Strategy:           versioning.StrategyPath,
    DefaultVersion:     2,
    MinVersion:         1,
    MaxVersion:         2,
    DeprecatedVersions: []int{1},
    OnDeprecated: func(c *gin.Context, version int) {
        log.Printf("Deprecated version %d used by %s", version, c.ClientIP())
    },
})

// V1 endpoints (deprecated)
v1 := versionedRouter.Version(1)
v1.GET("/orders", getOrdersV1)
v1.POST("/orders", createOrderV1)

// V2 endpoints (current)
v2 := versionedRouter.Version(2)
v2.GET("/orders", getOrdersV2)
v2.POST("/orders", createOrderV2)

// URLs:
// GET /api/v1/orders  -> getOrdersV1 (with deprecation warning)
// POST /api/v2/orders -> createOrdersV2
```

### Header-Based Versioning

```go
router.Use(versioning.Middleware(versioning.Config{
    Strategy:       versioning.StrategyHeader,
    HeaderName:     "X-API-Version",
    DefaultVersion: 2,
    MinVersion:     1,
    MaxVersion:     2,
}))

// Client request:
// GET /api/orders
// X-API-Version: 2
```

### Query Parameter Versioning

```go
router.Use(versioning.Middleware(versioning.Config{
    Strategy:       versioning.StrategyQueryParam,
    QueryParamName: "version",
    DefaultVersion: 1,
    MinVersion:     1,
    MaxVersion:     2,
}))

// Client request:
// GET /api/orders?version=2
```

### Accept Header Versioning

```go
router.Use(versioning.Middleware(versioning.Config{
    Strategy:       versioning.StrategyAcceptHeader,
    DefaultVersion: 1,
    MinVersion:     1,
    MaxVersion:     2,
}))

// Client request:
// GET /api/orders
// Accept: application/vnd.api.v2+json
```

### Getting Version in Handlers

```go
func getOrders(c *gin.Context) {
    version := versioning.GetVersion(c)

    if version >= 2 {
        // Use new response format
        c.JSON(200, OrderResponseV2{...})
    } else {
        // Use legacy format
        c.JSON(200, OrderResponseV1{...})
    }
}
```

### Multi-Version Handlers

```go
// Register same handler for multiple versions
versionedRouter.RegisterMultiVersion(
    []int{1, 2, 3},
    "GET",
    "/health",
    healthCheckHandler,
)
```

### Deprecation Response Headers

When accessing deprecated versions:
```
X-API-Deprecated: true
X-API-Deprecation-Info: Version 1 is deprecated. Please upgrade to version 2
X-API-Version: 1
```

---

## Swagger/OpenAPI Documentation

**Location:** `pkg/swagger/`

Automatic API documentation generation using Swagger/OpenAPI.

### Setup

1. **Install swag CLI**:
```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

2. **Add annotations to code**:

```go
// main.go
// @title Order Service API
// @version 2.0.0
// @description Enterprise-grade order management microservice
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.example.com/support
// @contact.email support@example.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
    router := gin.New()

    // Setup Swagger endpoint
    swagger.SetupSwagger(router, swagger.Config{
        Title:       "Order Service API",
        Description: "Enterprise order management",
        Version:     "2.0.0",
        Host:        "localhost:8080",
        BasePath:    "/api/v1",
    })

    // Swagger UI available at: http://localhost:8080/swagger/index.html
}
```

3. **Annotate handlers**:

```go
// CreateOrder godoc
// @Summary Create a new order
// @Description Create a new order with the provided details
// @Tags orders
// @Accept json
// @Produce json
// @Param order body CreateOrderRequest true "Order details"
// @Success 201 {object} CreateOrderResponse
// @Failure 400 {object} swagger.ErrorResponse
// @Failure 401 {object} swagger.ErrorResponse "Unauthorized"
// @Failure 403 {object} swagger.ErrorResponse "Forbidden"
// @Failure 500 {object} swagger.ErrorResponse
// @Security BearerAuth
// @Router /orders [post]
func (h *HTTPHandler) CreateOrder(c *gin.Context) {
    // implementation
}

// GetOrder godoc
// @Summary Get order by ID
// @Description Get detailed information about an order
// @Tags orders
// @Produce json
// @Param id path string true "Order ID"
// @Success 200 {object} Order
// @Failure 404 {object} swagger.ErrorResponse
// @Security BearerAuth
// @Router /orders/{id} [get]
func (h *HTTPHandler) GetOrder(c *gin.Context) {
    // implementation
}
```

4. **Generate documentation**:
```bash
swag init -g services/order-service/cmd/main.go -o services/order-service/docs
```

5. **Access Swagger UI**:
- Open browser: http://localhost:8080/swagger/index.html
- Test endpoints directly from the UI
- View request/response schemas

### Common Annotations

```go
// @Summary      Short description
// @Description  Longer description
// @Tags         category
// @Accept       json
// @Produce      json
// @Param        name  path      string  true   "Description"
// @Param        body  body      Type    true   "Description"
// @Param        query query     string  false  "Description"
// @Success      200   {object}  Type
// @Failure      400   {object}  ErrorType
// @Router       /path [method]
// @Security     BearerAuth
```

---

## Load Testing

**Location:** `tests/load/`

Comprehensive load testing with K6.

### Running Tests

```bash
# Install K6
brew install k6  # macOS
# or: sudo apt-get install k6  # Linux

# Run basic load test
k6 run tests/load/basic-load-test.js

# Run against specific environment
k6 run --env BASE_URL=http://staging.example.com tests/load/basic-load-test.js

# Run with authentication
k6 run --env JWT_TOKEN=your-token tests/load/basic-load-test.js
```

### Test Scenarios

1. **Smoke Test** - Quick validation
2. **Load Test** - Normal traffic (10-100 VUs)
3. **Stress Test** - Find breaking point (up to 500 VUs)
4. **Spike Test** - Sudden traffic spike
5. **Soak Test** - Sustained load (4 hours)

### Metrics

- **http_req_duration** - Request latency (p95, p99)
- **http_req_failed** - Error rate
- **order_creation_duration** - Custom business metric
- **Thresholds** - Automatic pass/fail criteria

See `tests/load/README.md` for detailed usage.

---

## Integration Examples

### Complete Service Example

```go
package main

import (
    "context"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/joshbarros/golang-logistics-services/pkg/cache"
    "github.com/joshbarros/golang-logistics-services/pkg/circuitbreaker"
    "github.com/joshbarros/golang-logistics-services/pkg/retry"
    "github.com/joshbarros/golang-logistics-services/pkg/swagger"
    "github.com/joshbarros/golang-logistics-services/pkg/tracing"
    "github.com/joshbarros/golang-logistics-services/pkg/versioning"
)

func main() {
    // Initialize tracing
    tracingMgr, _ := tracing.NewTracingManager(tracing.Config{
        ServiceName:    "order-service",
        ServiceVersion: "2.0.0",
        JaegerEndpoint: "http://localhost:14268/api/traces",
        SamplingRate:   0.1,
        Enabled:        true,
    })
    defer tracingMgr.Shutdown(context.Background())

    // Initialize cache
    cache, _ := cache.NewRedisCache("localhost:6379", "", "order", 0)
    defer cache.Close()

    // Initialize circuit breaker manager
    cbManager := circuitbreaker.NewCircuitBreakerManager(circuitbreaker.DefaultConfig())

    // Initialize retry
    retrier := retry.NewRetrier(retry.DefaultConfig())

    // Setup router
    router := gin.New()

    // Add tracing middleware
    router.Use(tracingMgr.Middleware())

    // Setup versioning
    versionedRouter := versioning.NewVersionedRouter(router, versioning.Config{
        Strategy:       versioning.StrategyPath,
        DefaultVersion: 2,
        MinVersion:     1,
        MaxVersion:     2,
    })

    // Setup Swagger
    swagger.SetupSwagger(router, swagger.Config{
        Title:   "Order Service API",
        Version: "2.0.0",
    })

    // Register handlers
    v2 := versionedRouter.Version(2)
    v2.GET("/orders/:id", getOrderHandler)

    router.Run(":8080")
}
```

---

## Performance Impact

| Feature | Latency Overhead | Memory Overhead | CPU Overhead |
|---------|------------------|-----------------|--------------|
| Circuit Breaker | < 1ms | Minimal | Minimal |
| Caching (Redis) | 1-5ms | None (external) | Minimal |
| Tracing (10% sample) | < 1ms | Low | Low |
| Retry | Depends on retries | Minimal | Minimal |
| Versioning | < 0.1ms | Minimal | Minimal |
| Swagger | None (runtime) | Low | None |

---

## Recommended Combinations

### High Availability Setup
- Circuit Breaker + Retry + Caching
- Protects against downstream failures
- Reduces load on failing services
- Provides fast fallback responses

### Debugging Setup
- Tracing + Logging + Metrics
- Full visibility into request flow
- Correlation between logs and traces
- Performance bottleneck identification

### Production Setup
- All features enabled
- Monitoring circuit breaker states
- Cache hit rate tracking
- Distributed tracing sampling
- Load testing in staging

---

## Next Steps

1. **Enable Gradually** - Start with one feature, add more as needed
2. **Monitor Impact** - Track performance metrics
3. **Tune Configuration** - Adjust based on traffic patterns
4. **Document Usage** - Add team-specific guidelines
5. **Train Team** - Ensure everyone understands the features

For questions or issues, see the main README.md or create an issue in the repository.
```
