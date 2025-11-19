# GraphQL API Gateway

The GraphQL API Gateway provides a unified, flexible API layer that aggregates data from all microservices in the logistics platform.

## Overview

The GraphQL gateway offers several advantages over traditional REST APIs:

- **Flexible Data Fetching**: Clients request exactly the data they need
- **Single Endpoint**: One endpoint for all operations
- **Strong Typing**: Self-documenting schema with type safety
- **Efficient Batching**: DataLoader solves N+1 query problems
- **Real-time Updates**: Built-in subscription support
- **Cross-Service Queries**: Fetch related data across multiple services in one request

## Architecture

```
┌─────────────┐
│   Clients   │
│ Web/Mobile  │
└──────┬──────┘
       │
       │ GraphQL Query
       ▼
┌──────────────────────┐
│  GraphQL Gateway     │
│  ┌────────────────┐  │
│  │  Query Engine  │  │
│  └────────────────┘  │
│  ┌────────────────┐  │
│  │  DataLoader    │  │ ← Batching & Caching
│  └────────────────┘  │
│  ┌────────────────┐  │
│  │ Circuit Breaker│  │ ← Resilience
│  └────────────────┘  │
└──────────────────────┘
       │
       │ gRPC
       ▼
┌────────────────────────────────────┐
│      Microservices Layer           │
│  ┌──────┐ ┌──────┐ ┌──────┐       │
│  │Order │ │Ship  │ │Inven │ ...   │
│  └──────┘ └──────┘ └──────┘       │
└────────────────────────────────────┘
```

## Features

### 1. Unified Schema

Single GraphQL schema spanning all services:

```graphql
type Query {
  # Orders
  order(id: ID!): Order
  orders(customerId: String, status: String, limit: Int, offset: Int): [Order!]!

  # Shipments
  shipment(id: ID!): Shipment
  shipments(orderId: String, status: String): [Shipment!]!

  # Inventory
  inventory(id: ID!): Inventory
  inventories(warehouseId: String): [Inventory!]!

  # Drivers
  driver(id: ID!): Driver
  drivers(status: String): [Driver!]!

  # Routes
  route(id: ID!): Route
  routes: [Route!]!
}

type Mutation {
  # Order mutations
  createOrder(customerId: ID!, items: [String!]!): Order!
  updateOrder(id: ID!, status: String): Order!
  cancelOrder(id: ID!): Boolean!

  # Shipment mutations
  createShipment(orderId: ID!): Shipment!
  updateShipment(id: ID!, status: String): Shipment!

  # Inventory mutations
  updateInventory(id: ID!, quantity: Int): Inventory!

  # Driver mutations
  assignDriver(shipmentId: ID!, driverId: ID!): Shipment!
}

type Subscription {
  # Real-time updates
  orderUpdates(orderId: String): Order!
  shipmentUpdates(shipmentId: String): Shipment!
  driverLocation(driverId: ID!): Location!
}
```

### 2. DataLoader for N+1 Optimization

Automatically batches and caches requests to prevent N+1 query problems:

```go
// Without DataLoader - N+1 problem
// Fetching 10 orders would make 1 + 10 = 11 database calls
for _, order := range orders {
    shipment := fetchShipment(order.ShipmentID) // 10 separate calls
}

// With DataLoader - Only 2 calls
// Automatically batches all shipment requests into one
loaders.ShipmentLoader.LoadMany(ctx, shipmentIDs)
```

**Benefits:**
- Reduces backend service calls by 90%+
- Lower latency for complex queries
- Automatic request deduplication
- Per-request caching

### 3. Circuit Breaker Protection

Protects against cascading failures when services are down:

```go
// Automatically wraps all service calls
orderCB := cbManager.GetOrCreate("order-service", config)

err := orderCB.Execute(func() error {
    return orderClient.GetOrder(ctx, id)
})

// Circuit states:
// - Closed: Normal operation
// - Open: Service unavailable, fail fast
// - Half-Open: Testing if service recovered
```

### 4. Query Complexity & Depth Limits

Prevents expensive queries from overloading the system:

```yaml
Complexity Limit: 1000 points
- Each field: 1 point
- Nested objects: 2 points
- Lists: 5 points multiplier

Depth Limit: 10 levels
- Prevents deeply nested queries
```

Example blocked query:
```graphql
# Too deep - 15 levels
query DeepQuery {
  order {
    shipment {
      driver {
        vehicle {
          ... # 15 levels deep
        }
      }
    }
  }
}
```

### 5. Rate Limiting

Per-user rate limiting protects against abuse:

```yaml
Default: 100 requests per minute per user
Burst: 20 requests
```

### 6. Authentication & Authorization

JWT-based authentication with role-based access:

```go
// Middleware extracts and validates JWT
graphqlGroup.Use(graphql.AuthMiddleware(jwtSecret))

// Resolvers can check user permissions
if !hasPermission(userID, "read:orders") {
    return nil, errors.New("unauthorized")
}
```

### 7. Distributed Tracing

OpenTelemetry integration tracks requests across services:

```
Trace: GET /graphql
├─ graphql.execute (10ms)
│  ├─ order-service.GetOrder (5ms)
│  ├─ shipment-service.GetShipment (3ms)
│  └─ driver-service.GetDriver (2ms)
└─ Total: 10ms
```

## Usage

### Starting the Gateway

```bash
# Development
cd services/gateway
go run main.go

# Production
docker run -p 8080:8080 \
  -e JWT_SECRET=your-secret \
  -e ORDER_SERVICE_URL=order:50051 \
  -e SHIPMENT_SERVICE_URL=shipment:50052 \
  logistics-gateway:latest
```

### Environment Variables

```bash
PORT=8080                          # HTTP port
JWT_SECRET=secret-key              # JWT signing key
ENABLE_PLAYGROUND=true             # Enable GraphQL Playground
REQUIRE_AUTH=false                 # Require authentication

# Service endpoints
ORDER_SERVICE_URL=localhost:50051
SHIPMENT_SERVICE_URL=localhost:50052
INVENTORY_SERVICE_URL=localhost:50053
DRIVER_SERVICE_URL=localhost:50054
ROUTE_SERVICE_URL=localhost:50055
NOTIFICATION_SERVICE_URL=localhost:50056

# Observability
JAEGER_ENDPOINT=http://localhost:14268/api/traces
METRICS_PORT=9090
```

## Example Queries

### 1. Simple Query

Get order details:

```graphql
query GetOrder {
  order(id: "order-123") {
    id
    customerId
    status
    totalAmount
    createdAt
  }
}
```

Response:
```json
{
  "data": {
    "order": {
      "id": "order-123",
      "customerId": "customer-456",
      "status": "pending",
      "totalAmount": 99.99,
      "createdAt": "2025-01-15T10:30:00Z"
    }
  }
}
```

### 2. Nested Query with Relationships

Get order with shipment and driver info:

```graphql
query GetOrderWithShipment {
  order(id: "order-123") {
    id
    totalAmount
    shipment {
      trackingNumber
      status
      estimatedDelivery
      driver {
        name
        phone
        currentLocation {
          latitude
          longitude
          timestamp
        }
        vehicleInfo {
          make
          model
          licensePlate
        }
      }
    }
  }
}
```

**Note**: This single query replaces 3 REST API calls!

### 3. Filtered List Query

Get customer's pending orders:

```graphql
query GetCustomerOrders {
  orders(
    customerId: "customer-456"
    status: "pending"
    limit: 10
    offset: 0
  ) {
    id
    status
    totalAmount
    createdAt
    items {
      productId
      quantity
      price
    }
  }
}
```

### 4. Aggregation Query

Dashboard data in one request:

```graphql
query Dashboard {
  activeOrders: orders(status: "active", limit: 5) {
    id
    totalAmount
    shipment {
      status
    }
  }

  availableDrivers: drivers(status: "available") {
    id
    name
    currentLocation {
      latitude
      longitude
    }
  }

  lowStockItems: inventories(warehouseId: "wh-001") {
    productId
    available
    reserved
  }
}
```

### 5. Create Order Mutation

```graphql
mutation CreateOrder {
  createOrder(
    customerId: "customer-789"
    items: ["product-1", "product-2"]
  ) {
    id
    status
    totalAmount
  }
}
```

### 6. Update Shipment Status

```graphql
mutation UpdateShipmentStatus {
  updateShipment(
    id: "shipment-456"
    status: "delivered"
  ) {
    id
    status
    trackingNumber
  }
}
```

### 7. Real-time Subscription

Subscribe to order updates:

```graphql
subscription WatchOrder {
  orderUpdates(orderId: "order-123") {
    id
    status
    updatedAt
  }
}
```

## Client Integration

### JavaScript/TypeScript

Using Apollo Client:

```typescript
import { ApolloClient, InMemoryCache, gql } from '@apollo/client';

const client = new ApolloClient({
  uri: 'http://localhost:8080/graphql',
  cache: new InMemoryCache(),
  headers: {
    Authorization: `Bearer ${token}`,
  },
});

// Query
const { data } = await client.query({
  query: gql`
    query GetOrder($id: ID!) {
      order(id: $id) {
        id
        status
        totalAmount
      }
    }
  `,
  variables: { id: 'order-123' },
});

// Mutation
const { data } = await client.mutate({
  mutation: gql`
    mutation CreateOrder($customerId: ID!, $items: [String!]!) {
      createOrder(customerId: $customerId, items: $items) {
        id
        status
      }
    }
  `,
  variables: {
    customerId: 'customer-456',
    items: ['item1', 'item2'],
  },
});
```

### React Hooks

```typescript
import { useQuery, useMutation } from '@apollo/client';

function OrderDetails({ orderId }) {
  const { loading, error, data } = useQuery(GET_ORDER_QUERY, {
    variables: { id: orderId },
  });

  if (loading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <div>
      <h2>Order {data.order.id}</h2>
      <p>Status: {data.order.status}</p>
      <p>Total: ${data.order.totalAmount}</p>
    </div>
  );
}
```

### Go Client

```go
import (
    "context"
    "github.com/machinebox/graphql"
)

client := graphql.NewClient("http://localhost:8080/graphql")

req := graphql.NewRequest(`
    query GetOrder($id: ID!) {
        order(id: $id) {
            id
            status
            totalAmount
        }
    }
`)

req.Var("id", "order-123")
req.Header.Set("Authorization", "Bearer "+token)

var response struct {
    Order struct {
        ID          string  `json:"id"`
        Status      string  `json:"status"`
        TotalAmount float64 `json:"totalAmount"`
    } `json:"order"`
}

if err := client.Run(context.Background(), req, &response); err != nil {
    log.Fatal(err)
}
```

### cURL

```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "query": "query { order(id: \"order-123\") { id status totalAmount } }"
  }'
```

## Performance Optimization

### 1. Use DataLoader for Related Data

**Bad** - N+1 queries:
```graphql
{
  orders {
    id
    shipment {  # Separate call for each order
      trackingNumber
    }
  }
}
```

**Good** - Batched with DataLoader:
```graphql
# Same query, but DataLoader batches all shipment requests
# 100 orders = 2 calls instead of 101 calls
```

### 2. Request Only Needed Fields

**Bad** - Over-fetching:
```graphql
{
  orders {
    id
    customerId
    status
    # ... 20 more fields you don't need
  }
}
```

**Good** - Minimal fields:
```graphql
{
  orders {
    id
    status  # Only what you need
  }
}
```

### 3. Use Pagination

**Bad** - Fetch all:
```graphql
{
  orders {  # Could be thousands
    id
    status
  }
}
```

**Good** - Paginated:
```graphql
{
  orders(limit: 20, offset: 0) {
    id
    status
  }
}
```

### 4. Enable Query Caching

For frequently accessed data:

```typescript
const { data } = useQuery(GET_ORDER_QUERY, {
  variables: { id: orderId },
  fetchPolicy: 'cache-first',  // Use cache if available
});
```

## Monitoring & Observability

### Metrics

Available at `/metrics`:

```
# GraphQL operation counts
graphql_operations_total{operation="GetOrder",status="success"} 1234

# Operation duration
graphql_operation_duration_seconds{operation="GetOrder",quantile="0.99"} 0.045

# Query complexity
graphql_query_complexity{operation="Dashboard"} 250

# Circuit breaker states
circuit_breaker_state{service="order-service",state="closed"} 1
```

### Tracing

View traces in Jaeger:

```bash
# Access Jaeger UI
http://localhost:16686

# Filter by service
Service: gateway

# View trace spans
graphql.execute
├─ order-service.GetOrder
├─ shipment-service.GetShipment
└─ driver-service.GetDriver
```

### Logs

Structured logging for all operations:

```json
{
  "timestamp": "2025-01-15T10:30:00Z",
  "level": "info",
  "operation": "GetOrder",
  "userId": "user-123",
  "duration": "45ms",
  "status": "success"
}
```

## Error Handling

GraphQL errors are returned in standardized format:

```json
{
  "errors": [
    {
      "message": "Order not found",
      "path": ["order"],
      "extensions": {
        "code": "NOT_FOUND",
        "service": "order-service"
      }
    }
  ],
  "data": {
    "order": null
  }
}
```

### Error Codes

- `UNAUTHORIZED`: Invalid or missing authentication
- `FORBIDDEN`: Insufficient permissions
- `NOT_FOUND`: Resource doesn't exist
- `BAD_REQUEST`: Invalid input
- `INTERNAL_ERROR`: Server error
- `SERVICE_UNAVAILABLE`: Backend service down (circuit breaker open)
- `RATE_LIMIT_EXCEEDED`: Too many requests
- `QUERY_TOO_COMPLEX`: Exceeded complexity limit

## Security Best Practices

### 1. Authentication Required

```go
// Enforce authentication for all operations
router.Use(graphql.AuthMiddleware(jwtSecret))
```

### 2. Query Depth & Complexity Limits

```go
// Prevent expensive queries
router.Use(graphql.ComplexityMiddleware(1000))
router.Use(graphql.DepthMiddleware(10))
```

### 3. Rate Limiting

```go
// Per-user rate limiting
router.Use(graphql.RateLimitMiddleware(100))
```

### 4. Input Validation

```graphql
# Strong typing prevents injection
mutation CreateOrder($customerId: ID!, $items: [String!]!) {
  createOrder(customerId: $customerId, items: $items) {
    id
  }
}
```

### 5. Field-Level Authorization

```go
// Resolver checks permissions
if !hasPermission(userID, "read:order") {
    return nil, errors.New("unauthorized")
}
```

## Comparison: GraphQL vs REST

| Feature | REST | GraphQL |
|---------|------|---------|
| Endpoints | Multiple (one per resource) | Single endpoint |
| Over-fetching | Common | Eliminated |
| Under-fetching | Multiple requests needed | Single request |
| Versioning | URL versioning (v1, v2) | Schema evolution |
| Documentation | Swagger/OpenAPI | Self-documenting |
| Caching | HTTP caching | Custom per-query |
| Learning Curve | Lower | Higher |

### When to Use GraphQL

✅ **Use GraphQL when:**
- Client needs flexible data fetching
- Multiple related resources in one request
- Mobile apps (minimize network calls)
- Real-time subscriptions needed
- Complex, nested data structures

❌ **Use REST when:**
- Simple CRUD operations
- File uploads/downloads
- Public APIs for third parties
- Team unfamiliar with GraphQL
- HTTP caching is critical

## Troubleshooting

### High Query Complexity

**Symptom**: Queries rejected with "Query too complex"

**Solution**: Break into multiple queries or increase limit:
```go
config.MaxComplexity = 2000  // Increase limit
```

### N+1 Query Problems

**Symptom**: Slow queries with many related resources

**Solution**: Ensure DataLoader is used in resolvers:
```go
loaders := GetDataLoader(p.Context)
return loaders.OrderLoader.Load(p.Context, id)
```

### Circuit Breaker Open

**Symptom**: Errors with "service unavailable"

**Solution**: Check service health and wait for circuit to close:
```bash
curl http://localhost:8080/health
# Check circuit_breakers.order status
```

### Rate Limit Exceeded

**Symptom**: "Rate limit exceeded" errors

**Solution**: Implement exponential backoff or request limit increase

## Advanced Features

### Custom Directives

```graphql
directive @auth(requires: Role!) on FIELD_DEFINITION

type Query {
  orders: [Order!]! @auth(requires: ADMIN)
}
```

### Query Batching

Send multiple queries in one HTTP request:

```json
[
  { "query": "{ order(id: \"1\") { id } }" },
  { "query": "{ order(id: \"2\") { id } }" }
]
```

### Persisted Queries

Pre-register queries for security and performance:

```json
{
  "id": "abc123",
  "variables": { "orderId": "order-123" }
}
```

## Resources

- [GraphQL Playground](http://localhost:8080/playground) - Interactive query editor
- [Health Check](http://localhost:8080/health) - Service health status
- [Metrics](http://localhost:8080/metrics) - Prometheus metrics
- [GraphQL Spec](https://spec.graphql.org/) - Official specification
- [Apollo Docs](https://www.apollographql.com/docs/) - Client library documentation

## Summary

The GraphQL API Gateway provides:

✅ **Unified API** - Single endpoint for all services
✅ **Flexible Queries** - Fetch exactly what you need
✅ **High Performance** - DataLoader prevents N+1 queries
✅ **Resilient** - Circuit breakers protect against failures
✅ **Secure** - Authentication, rate limiting, complexity limits
✅ **Observable** - Tracing, metrics, structured logs
✅ **Real-time** - WebSocket subscriptions for live updates

Perfect for modern web and mobile applications requiring efficient, flexible data access across multiple microservices.
