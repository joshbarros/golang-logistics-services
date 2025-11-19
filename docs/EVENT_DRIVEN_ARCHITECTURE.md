# Event-Driven Architecture Guide

This document provides comprehensive guidance on using event-driven architecture, asynchronous processing, real-time communication, feature flags, and multi-tenancy in the Logistics Services platform.

## Table of Contents

1. [Event-Driven Architecture](#event-driven-architecture)
2. [Background Job Processing](#background-job-processing)
3. [WebSocket Real-Time Communication](#websocket-real-time-communication)
4. [Feature Flags & A/B Testing](#feature-flags--ab-testing)
5. [Multi-Tenancy](#multi-tenancy)
6. [Integration Patterns](#integration-patterns)
7. [Best Practices](#best-practices)

---

## Event-Driven Architecture

**Location:** `pkg/events/`

Event-driven architecture enables loosely coupled microservices through asynchronous event publishing and subscription.

### Core Concepts

- **Event:** A notification that something happened in the system
- **Publisher:** Service that emits events
- **Subscriber:** Service that listens to and handles events
- **Event Bus:** Message broker that routes events (RabbitMQ in production)

### Event Types

```go
const (
    // Order events
    EventOrderCreated   = "order.created"
    EventOrderUpdated   = "order.updated"
    EventOrderCancelled = "order.cancelled"
    EventOrderCompleted = "order.completed"

    // Shipment events
    EventShipmentCreated   = "shipment.created"
    EventShipmentPickedUp  = "shipment.picked_up"
    EventShipmentInTransit = "shipment.in_transit"
    EventShipmentDelivered = "shipment.delivered"

    // Inventory events
    EventInventoryReserved = "inventory.reserved"
    EventInventoryReleased = "inventory.released"
    EventLowStockAlert     = "inventory.low_stock"

    // Driver events
    EventDriverAssigned  = "driver.assigned"
    EventLocationUpdated = "driver.location_updated"

    // Notification events
    EventNotificationSent = "notification.sent"
)
```

### Setup

**Development (In-Memory):**

```go
import "github.com/joshbarros/golang-logistics-services/pkg/events"

// Create in-memory event bus
eventBus := events.NewInMemoryEventBus()
defer eventBus.Close()
```

**Production (RabbitMQ):**

```go
// Create RabbitMQ event bus
eventBus, err := events.NewRabbitMQEventBus(events.Config{
    BrokerURL:     "amqp://guest:guest@localhost:5672/",
    Exchange:      "logistics-events",
    ExchangeType:  "topic",
    QueuePrefix:   "order-service",
    ConsumerTag:   "order-service-consumer",
    Durable:       true,
    PrefetchCount: 10,
    RetryAttempts: 3,
    RetryDelay:    2 * time.Second,
    OnError: func(err error, event events.Event) {
        log.Printf("Event error: %v", err)
    },
})
if err != nil {
    log.Fatal(err)
}
defer eventBus.Close()
```

### Publishing Events

```go
// Build event
event := events.NewEventBuilder(events.EventOrderCreated, "order-service").
    WithData(events.OrderCreatedData{
        OrderID:     "order-123",
        CustomerID:  "customer-456",
        TotalAmount: 99.99,
        ItemCount:   3,
        CreatedAt:   time.Now(),
    }).
    WithMetadata("correlation_id", "req-789").
    WithMetadata("user_id", "user-101").
    Build()

// Publish
if err := eventBus.Publish(context.Background(), event); err != nil {
    log.Printf("Failed to publish event: %v", err)
}

// Batch publish
events := []events.Event{event1, event2, event3}
eventBus.PublishBatch(context.Background(), events)
```

### Subscribing to Events

**Single Event Type:**

```go
err := eventBus.Subscribe(context.Background(), events.EventOrderCreated,
    func(ctx context.Context, event events.Event) error {
        log.Printf("Received order.created: %+v", event)

        // Parse event data
        data, ok := event.Data.(events.OrderCreatedData)
        if !ok {
            return errors.New("invalid event data")
        }

        // Process event
        return processOrderCreated(ctx, data)
    })
```

**Multiple Event Types:**

```go
eventBus.SubscribeMultiple(
    context.Background(),
    []string{events.EventOrderCreated, events.EventOrderUpdated},
    func(ctx context.Context, event events.Event) error {
        switch event.Type {
        case events.EventOrderCreated:
            return handleOrderCreated(ctx, event)
        case events.EventOrderUpdated:
            return handleOrderUpdated(ctx, event)
        }
        return nil
    })
```

**Pattern Matching (RabbitMQ):**

```go
// Subscribe to all order events
eventBus.Subscribe(ctx, "order.*", orderEventHandler)

// Subscribe to all events
eventBus.Subscribe(ctx, "#", allEventsHandler)
```

### Event Sourcing

```go
// Create event store
eventStore := events.NewInMemoryEventStore()

// Save events
event := events.NewEventBuilder("order.created", "order-service").
    WithData(orderData).
    Build()

eventStore.Save(ctx, event)

// Retrieve events
orderEvents, _ := eventStore.GetByType(ctx, "order.created", 100)
recentEvents, _ := eventStore.GetSince(ctx, time.Now().Add(-24*time.Hour), 1000)
```

### Use Cases

**1. Order Processing Pipeline:**

```go
// Order Service publishes order.created
eventBus.Publish(ctx, orderCreatedEvent)

// Inventory Service subscribes and reserves stock
eventBus.Subscribe(ctx, "order.created", func(ctx context.Context, e events.Event) error {
    data := e.Data.(events.OrderCreatedData)
    return inventoryService.ReserveStock(ctx, data.OrderID)
})

// Payment Service subscribes and processes payment
eventBus.Subscribe(ctx, "order.created", func(ctx context.Context, e events.Event) error {
    data := e.Data.(events.OrderCreatedData)
    return paymentService.ProcessPayment(ctx, data.OrderID, data.TotalAmount)
})

// Notification Service sends confirmation
eventBus.Subscribe(ctx, "order.created", func(ctx context.Context, e events.Event) error {
    data := e.Data.(events.OrderCreatedData)
    return notificationService.SendOrderConfirmation(ctx, data.CustomerID, data.OrderID)
})
```

**2. Saga Pattern:**

```go
// Distributed transaction using events
// Step 1: Order created
eventBus.Publish(ctx, orderCreatedEvent)

// Step 2: Inventory reserved (success -> publish, fail -> compensate)
if err := inventoryService.Reserve(orderID); err != nil {
    eventBus.Publish(ctx, inventoryReservationFailedEvent)
    // Compensate: cancel order
} else {
    eventBus.Publish(ctx, inventoryReservedEvent)
}

// Step 3: Payment processed
if err := paymentService.Process(orderID); err != nil {
    // Compensate: release inventory
    eventBus.Publish(ctx, paymentFailedEvent)
} else {
    eventBus.Publish(ctx, paymentSucceededEvent)
}
```

---

## Background Job Processing

**Location:** `pkg/worker/`

Background job processing offloads long-running tasks to worker pools for asynchronous execution.

### Setup

```go
import "github.com/joshbarros/golang-logistics-services/pkg/worker"

// Create queue
queue := worker.NewInMemoryQueue(1000)
defer queue.Close()

// Create worker
w := worker.NewWorker("worker-1", queue, worker.Config{
    Concurrency:  10,              // 10 concurrent workers
    PollInterval: 1 * time.Second, // Poll every second
    RetryDelay:   5 * time.Second, // Wait 5s between retries
    OnJobStart: func(job worker.Job) {
        log.Printf("Starting job %s (type: %s)", job.ID, job.Type)
    },
    OnJobComplete: func(job worker.Job, err error) {
        if err != nil {
            log.Printf("Job %s failed: %v", job.ID, err)
        } else {
            log.Printf("Job %s completed", job.ID)
        }
    },
})

// Register handlers
w.RegisterHandler(worker.JobSendEmail, handleSendEmail)
w.RegisterHandler(worker.JobSendSMS, handleSendSMS)
w.RegisterHandler(worker.JobProcessPayment, handleProcessPayment)
w.RegisterHandler(worker.JobGenerateReport, handleGenerateReport)

// Start worker pool
w.Start()
defer w.Stop()
```

### Job Handlers

```go
func handleSendEmail(ctx context.Context, job worker.Job) error {
    payload, ok := job.Payload.(worker.SendEmailPayload)
    if !ok {
        return worker.ErrInvalidJob
    }

    return emailService.Send(ctx, payload.To, payload.Subject, payload.Body)
}

func handleGenerateReport(ctx context.Context, job worker.Job) error {
    // Long-running task
    report, err := reportService.Generate(ctx)
    if err != nil {
        return err
    }

    // Upload to S3
    return s3Service.Upload(ctx, report)
}
```

### Enqueueing Jobs

**Simple Job:**

```go
job := worker.NewJobBuilder(worker.JobSendEmail).
    WithPayload(worker.SendEmailPayload{
        To:      "customer@example.com",
        Subject: "Order Confirmation",
        Body:    "Your order has been confirmed",
    }).
    WithMaxRetries(3).
    Build()

queue.Enqueue(context.Background(), job)
```

**Priority Job:**

```go
urgentJob := worker.NewJobBuilder(worker.JobSendEmail).
    WithPayload(emailPayload).
    WithPriority(10). // Higher priority
    Build()
```

**Scheduled Job:**

```go
// Schedule for future execution
reminderJob := worker.NewJobBuilder(worker.JobSendEmail).
    WithPayload(reminderPayload).
    WithDelay(24 * time.Hour). // Execute in 24 hours
    Build()

queue.Enqueue(ctx, reminderJob)
```

**Batch Jobs:**

```go
jobs := []worker.Job{job1, job2, job3}
queue.EnqueueBatch(context.Background(), jobs)
```

### Common Job Types

```go
const (
    JobSendEmail             = "send_email"
    JobSendSMS               = "send_sms"
    JobProcessPayment        = "process_payment"
    JobGenerateReport        = "generate_report"
    JobSyncInventory         = "sync_inventory"
    JobCalculateRouteMetrics = "calculate_route_metrics"
    JobSendNotification      = "send_notification"
    JobCleanupOldData        = "cleanup_old_data"
)
```

### Use Cases

**1. Email Campaigns:**

```go
// Enqueue bulk emails
for _, customer := range customers {
    job := worker.NewJobBuilder(worker.JobSendEmail).
        WithPayload(worker.SendEmailPayload{
            To:      customer.Email,
            Subject: "Special Offer",
            Body:    emailTemplate,
        }).
        Build()

    queue.Enqueue(ctx, job)
}
```

**2. Report Generation:**

```go
// Generate daily reports asynchronously
job := worker.NewJobBuilder(worker.JobGenerateReport).
    WithPayload(map[string]interface{}{
        "report_type": "daily_sales",
        "date":        time.Now().Format("2006-01-02"),
    }).
    WithMaxRetries(1).
    Build()

queue.Enqueue(ctx, job)
```

**3. Data Cleanup:**

```go
// Schedule nightly cleanup
cleanupJob := worker.NewJobBuilder(worker.JobCleanupOldData).
    WithPayload(map[string]interface{}{
        "retention_days": 90,
    }).
    WithScheduledAt(time.Now().Add(nextMidnight())).
    Build()
```

---

## WebSocket Real-Time Communication

**Location:** `pkg/websocket/`

WebSocket support enables bidirectional real-time communication between server and clients.

### Setup

```go
import "github.com/joshbarros/golang-logistics-services/pkg/websocket"

// Create WebSocket hub
wsHub := websocket.NewHub(websocket.Config{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    MaxMessageSize:  512 * 1024, // 512 KB
    PongWait:        60 * time.Second,
    PingPeriod:      54 * time.Second,
    OnConnect: func(client *websocket.Client) {
        log.Printf("Client connected: %s (user: %s)", client.ID, client.UserID)
    },
    OnDisconnect: func(client *websocket.Client) {
        log.Printf("Client disconnected: %s", client.ID)
    },
    OnMessage: func(client *websocket.Client, message websocket.Message) error {
        log.Printf("Received from %s: %+v", client.ID, message)
        return handleClientMessage(client, message)
    },
})

// Start hub
go wsHub.Run()

// Setup WebSocket endpoint
router.GET("/ws", websocket.HandleWebSocket(wsHub, websocket.DefaultConfig()))
```

### Broadcasting Messages

**To All Clients:**

```go
message := websocket.Message{
    Type:      websocket.MessageTypeSystemAlert,
    Data:      map[string]string{"message": "System maintenance in 10 minutes"},
    Timestamp: time.Now(),
}

wsHub.Broadcast(message)
```

**To Specific Client:**

```go
wsHub.SendToClient("client-123", websocket.Message{
    Type: websocket.MessageTypeNotification,
    Data: notificationData,
    Timestamp: time.Now(),
})
```

**To Specific User (All Connections):**

```go
wsHub.SendToUser("user-456", websocket.Message{
    Type: websocket.MessageTypeOrderUpdate,
    Data: orderUpdate,
    Timestamp: time.Now(),
})
```

### Message Types

```go
const (
    MessageTypeOrderUpdate    = "order.update"
    MessageTypeShipmentUpdate = "shipment.update"
    MessageTypeDriverLocation = "driver.location"
    MessageTypeNotification   = "notification"
    MessageTypeChat           = "chat"
    MessageTypeSystemAlert    = "system.alert"
)
```

### Use Cases

**1. Real-Time Order Tracking:**

```go
// When order status changes
eventBus.Subscribe(ctx, "order.updated", func(ctx context.Context, event events.Event) error {
    data := event.Data.(OrderUpdatedData)

    // Push update to customer via WebSocket
    wsHub.SendToUser(data.CustomerID, websocket.Message{
        Type: websocket.MessageTypeOrderUpdate,
        Data: map[string]interface{}{
            "order_id": data.OrderID,
            "status":   data.NewStatus,
            "eta":      data.ETA,
        },
        Timestamp: time.Now(),
    })

    return nil
})
```

**2. Live Driver Location:**

```go
// Driver sends location updates
router.POST("/api/v1/drivers/location", func(c *gin.Context) {
    var location DriverLocation
    c.BindJSON(&location)

    // Store location
    driverService.UpdateLocation(c.Request.Context(), location)

    // Broadcast to customers tracking this driver
    wsHub.Broadcast(websocket.Message{
        Type: websocket.MessageTypeDriverLocation,
        Data: location,
    })
})
```

**3. Real-Time Notifications:**

```go
// Push notifications to connected users
wsHub.SendToUser(userID, websocket.Message{
    Type: websocket.MessageTypeNotification,
    Data: map[string]string{
        "title": "Delivery Complete",
        "body":  "Your package has been delivered",
        "icon":  "success",
    },
})
```

**4. Admin Dashboard Updates:**

```go
// Real-time metrics for admin dashboard
go func() {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        metrics := metricsService.GetRealTimeMetrics()

        wsHub.Broadcast(websocket.Message{
            Type: "dashboard.metrics",
            Data: metrics,
        })
    }
}()
```

---

## Feature Flags & A/B Testing

**Location:** `pkg/featureflags/`

Feature flags enable dynamic feature toggles, gradual rollouts, and A/B testing without code deployment.

### Setup

```go
import "github.com/joshbarros/golang-logistics-services/pkg/featureflags"

// Create provider
provider := featureflags.NewInMemoryProvider()
```

### Creating Feature Flags

**Simple Toggle:**

```go
flag := featureflags.NewFlagBuilder("new-ui", "New UI Design").
    WithDescription("Enable new user interface").
    WithEnabled(false). // Default off
    Build()

provider.UpdateFlag(context.Background(), flag)
```

**Gradual Rollout:**

```go
flag := featureflags.NewFlagBuilder("premium-features", "Premium Features").
    WithDescription("Premium tier features").
    WithEnabled(false).
    WithRule(featureflags.Rule{
        ID:                "gradual-rollout",
        Enabled:           true,
        Conditions:        []featureflags.Condition{},
        RolloutPercentage: 10, // 10% of users
    }).
    Build()
```

**Targeted Rollout:**

```go
flag := featureflags.NewFlagBuilder("beta-features", "Beta Features").
    WithEnabled(false).
    WithRule(featureflags.Rule{
        ID:      "beta-users",
        Enabled: true,
        Conditions: []featureflags.Condition{
            {
                Attribute: "group",
                Operator:  "in",
                Values:    []string{"beta-testers", "internal"},
            },
        },
    }).
    Build()
```

**A/B Testing:**

```go
flag := featureflags.NewFlagBuilder("checkout-flow", "Checkout Flow").
    WithDescription("A/B test checkout UI").
    WithEnabled(true).
    WithVariations([]featureflags.Variation{
        {Key: "control", Name: "Control", Value: "old-ui", Weight: 50},
        {Key: "variant-a", Name: "Variant A", Value: "new-ui-v1", Weight: 25},
        {Key: "variant-b", Name: "Variant B", Value: "new-ui-v2", Weight: 25},
    }).
    WithDefaultVariation("control").
    Build()
```

### Evaluating Flags

```go
// Build evaluation context
evalCtx := featureflags.EvalContext{
    UserID: "user-123",
    Email:  "user@example.com",
    Groups: []string{"beta-testers"},
    Attributes: map[string]interface{}{
        "country": "US",
        "plan":    "premium",
        "age":     25,
    },
}

// Check if enabled
enabled, _ := provider.IsEnabled(ctx, "new-ui", evalCtx)
if enabled {
    // Show new UI
}

// Get variation for A/B test
variation, _ := provider.GetVariation(ctx, "checkout-flow", evalCtx)
switch variation {
case "control":
    // Show old checkout
case "variant-a":
    // Show variant A
case "variant-b":
    // Show variant B
}

// Get variation value
value, _ := provider.GetVariationValue(ctx, "checkout-flow", evalCtx)
uiVersion := value.(string)
```

### Use Cases

**1. Kill Switch:**

```go
// Disable feature in emergency
flag.Enabled = false
provider.UpdateFlag(ctx, flag)

// All users immediately see old behavior
```

**2. Beta Testing:**

```go
// Enable for internal team first
flag := featureflags.NewFlagBuilder("experimental-feature", "Experimental Feature").
    WithRule(featureflags.Rule{
        Conditions: []featureflags.Condition{
            {Attribute: "email", Operator: "contains", Values: []string{"@company.com"}},
        },
    }).
    Build()
```

**3. Regional Rollout:**

```go
// Enable for specific countries
flag := featureflags.NewFlagBuilder("localized-feature", "Localized Feature").
    WithRule(featureflags.Rule{
        Conditions: []featureflags.Condition{
            {Attribute: "country", Operator: "in", Values: []string{"US", "CA", "UK"}},
        },
    }).
    Build()
```

---

## Multi-Tenancy

**Location:** `pkg/multitenancy/`

Multi-tenancy enables a single application instance to serve multiple isolated tenants.

### Setup

```go
import "github.com/joshbarros/golang-logistics-services/pkg/multitenancy"

// Create tenant manager
manager := multitenancy.NewManager(multitenancy.Config{
    Strategy:      "header",      // or "subdomain", "path", "custom"
    HeaderName:    "X-Tenant-ID",
    RequireTenant: true,
})

// Add tenants
manager.AddTenant(&multitenancy.Tenant{
    ID:     "tenant-1",
    Name:   "Acme Corp",
    Slug:   "acme",
    Domain: "acme.example.com",
    Active: true,
})

manager.AddTenant(&multitenancy.Tenant{
    ID:     "tenant-2",
    Name:   "Beta Inc",
    Slug:   "beta",
    Domain: "beta.example.com",
    Active: true,
})

// Add middleware
router.Use(manager.Middleware())

// Register GORM callbacks for automatic tenant scoping
multitenancy.RegisterCallbacks(db)
```

### Tenant-Aware Models

```go
type Order struct {
    ID        string    `json:"id" gorm:"primaryKey"`
    TenantID  string    `json:"tenant_id" gorm:"index;not null"`
    Customer  string    `json:"customer"`
    Total     float64   `json:"total"`
    CreatedAt time.Time `json:"created_at"`
}

func (o *Order) GetTenantID() string { return o.TenantID }
func (o *Order) SetTenantID(id string) { o.TenantID = id }
```

### Querying with Tenant Isolation

**Automatic (via callbacks):**

```go
router.GET("/orders", func(c *gin.Context) {
    ctx := c.Request.Context()

    var orders []Order
    // Automatically filtered by tenant_id
    db.WithContext(ctx).Find(&orders)

    c.JSON(200, orders)
})
```

**Manual scoping:**

```go
ctx := c.Request.Context()
tenantID, _ := multitenancy.FromContext(ctx)

var orders []Order
db.Where("tenant_id = ?", tenantID).Find(&orders)
```

**Using TenantDB wrapper:**

```go
ctx := c.Request.Context()
tdb := multitenancy.NewTenantDB(db, ctx)

// All queries automatically scoped
var orders []Order
tdb.Find(&orders)
```

### Tenant Identification Strategies

**1. Header-Based:**
```http
GET /api/v1/orders
X-Tenant-ID: tenant-1
```

**2. Subdomain-Based:**
```
https://acme.example.com/api/v1/orders
https://beta.example.com/api/v1/orders
```

**3. Path-Based:**
```
GET /tenants/tenant-1/api/v1/orders
GET /tenants/tenant-2/api/v1/orders
```

---

## Integration Patterns

### Pattern 1: Event-Driven + Background Jobs

```go
// Subscribe to events and enqueue background jobs
eventBus.Subscribe(ctx, "order.created", func(ctx context.Context, event events.Event) error {
    data := event.Data.(events.OrderCreatedData)

    // Enqueue email job
    emailJob := worker.NewJobBuilder(worker.JobSendEmail).
        WithPayload(worker.SendEmailPayload{
            To:      data.CustomerEmail,
            Subject: "Order Confirmation",
            Body:    fmt.Sprintf("Order %s confirmed", data.OrderID),
        }).
        Build()

    return queue.Enqueue(ctx, emailJob)
})
```

### Pattern 2: WebSocket + Events

```go
// Push event updates to WebSocket clients
eventBus.Subscribe(ctx, "shipment.*", func(ctx context.Context, event events.Event) error {
    // Broadcast to all connected clients
    wsHub.Broadcast(websocket.Message{
        Type:      websocket.MessageTypeShipmentUpdate,
        Data:      event.Data,
        Timestamp: event.Timestamp,
    })
    return nil
})
```

### Pattern 3: Feature Flags + A/B Testing

```go
router.GET("/api/v1/ui-config", func(c *gin.Context) {
    evalCtx := buildEvalContext(c)

    // Get UI version from A/B test
    uiVersion, _ := flags.GetVariationValue(ctx, "ui-version", evalCtx)

    // Check if new features enabled
    newFeatures, _ := flags.IsEnabled(ctx, "new-features", evalCtx)

    c.JSON(200, gin.H{
        "ui_version":   uiVersion,
        "new_features": newFeatures,
    })
})
```

### Pattern 4: Multi-Tenancy + Events

```go
// Publish tenant-scoped events
func (s *OrderService) CreateOrder(ctx context.Context, order *Order) error {
    tenantID, _ := multitenancy.FromContext(ctx)

    // Create order (tenant_id auto-set)
    if err := s.repo.Create(ctx, order); err != nil {
        return err
    }

    // Publish event with tenant context
    event := events.NewEventBuilder("order.created", "order-service").
        WithData(order).
        WithMetadata("tenant_id", tenantID).
        Build()

    return s.eventBus.Publish(ctx, event)
}
```

---

## Best Practices

### Event-Driven

1. **Idempotent Handlers** - Handle duplicate events gracefully
2. **Event Versioning** - Include version in event schema
3. **Correlation IDs** - Track requests across services
4. **Dead Letter Queues** - Handle failed events
5. **Schema Registry** - Validate event payloads

### Background Jobs

1. **Job Timeouts** - Set reasonable timeouts
2. **Retry Strategy** - Exponential backoff for retries
3. **Job Monitoring** - Track job success/failure rates
4. **Queue Metrics** - Monitor queue depth
5. **Worker Scaling** - Auto-scale based on queue size

### WebSocket

1. **Authentication** - Validate connections
2. **Rate Limiting** - Prevent message spam
3. **Heartbeats** - Detect dead connections
4. **Message Size Limits** - Prevent memory issues
5. **Graceful Shutdown** - Close connections cleanly

### Feature Flags

1. **Flag Cleanup** - Remove old flags
2. **Default Values** - Always provide defaults
3. **Testing** - Test all variations
4. **Monitoring** - Track flag usage
5. **Documentation** - Document flag purpose

### Multi-Tenancy

1. **Data Isolation** - Ensure tenant data separation
2. **Performance** - Index tenant_id columns
3. **Tenant Limits** - Enforce resource quotas
4. **Testing** - Test cross-tenant isolation
5. **Migrations** - Handle schema changes carefully

---

**For more information, see:**
- [ADVANCED_FEATURES.md](ADVANCED_FEATURES.md) - Circuit breaker, caching, tracing
- [PRODUCTION_DEPLOYMENT_GUIDE.md](PRODUCTION_DEPLOYMENT_GUIDE.md) - Deployment procedures
- [API.md](API.md) - API documentation
