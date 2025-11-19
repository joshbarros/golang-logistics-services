# gRPC Implementation Guide

## Current Status

✅ **Completed:**
- Protocol buffer definitions created for all 6 services
- Proto generation script created (`scripts/generate-proto.sh`)
- Makefile targets added (`make proto-gen`, `make proto-clean`)
- Comprehensive proto documentation

❌ **Requires Network Access:**
- Installing protoc compiler
- Installing Go gRPC plugins (`protoc-gen-go`, `protoc-gen-go-grpc`)
- Generating `.pb.go` and `_grpc.pb.go` files
- Implementing gRPC server interfaces

## Why gRPC Generation Cannot Be Done in Current Environment

The current environment has network restrictions that prevent:
1. Installing the Protocol Buffer compiler (`protoc`)
2. Running `go install` to get gRPC Go plugins
3. Downloading dependencies required for generated code

## What Needs to Be Done (In Environment With Network Access)

### Step 1: Install Prerequisites

```bash
# Install protoc
brew install protobuf  # macOS
# OR
sudo apt install protobuf-compiler  # Ubuntu

# Install Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Verify installation
protoc --version
which protoc-gen-go
which protoc-gen-go-grpc
```

### Step 2: Generate Proto Code

```bash
cd /path/to/golang-logistics-services

# Generate all proto files
make proto-gen

# Verify generation
find proto -name "*.pb.go"
```

Expected output:
```
proto/order/v1/order.pb.go
proto/order/v1/order_grpc.pb.go
proto/shipment/v1/shipment.pb.go
proto/shipment/v1/shipment_grpc.pb.go
proto/inventory/v1/inventory.pb.go
proto/inventory/v1/inventory_grpc.pb.go
proto/route/v1/route.pb.go
proto/route/v1/route_grpc.pb.go
proto/driver/v1/driver.pb.go
proto/driver/v1/driver_grpc.pb.go
proto/notification/v1/notification.pb.go
proto/notification/v1/notification_grpc.pb.go
```

### Step 3: Implement gRPC Servers

For each service, create a gRPC server implementation:

#### Order Service Example

Create `services/order-service/internal/grpc/server.go`:

```go
package grpc

import (
    "context"
    "time"

    pb "github.com/joshbarros/golang-logistics-services/proto/order/v1"
    "github.com/joshbarros/golang-logistics-services/services/order-service/internal/models"
    "github.com/joshbarros/golang-logistics-services/services/order-service/internal/service"
    "google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
    pb.UnimplementedOrderServiceServer
    orderService *service.OrderService
}

func NewServer(orderService *service.OrderService) *Server {
    return &Server{
        orderService: orderService,
    }
}

func (s *Server) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
    // Convert proto items to domain items
    items := make([]models.OrderItem, len(req.Items))
    for i, item := range req.Items {
        items[i] = models.OrderItem{
            ProductID:   item.ProductId,
            ProductName: item.ProductName,
            Quantity:    int(item.Quantity),
            UnitPrice:   item.UnitPrice,
        }
    }

    // Convert proto addresses to domain addresses
    shippingAddr := models.Address{
        Street:     req.ShippingAddress.Street,
        City:       req.ShippingAddress.City,
        State:      req.ShippingAddress.State,
        PostalCode: req.ShippingAddress.PostalCode,
        Country:    req.ShippingAddress.Country,
    }

    billingAddr := models.Address{
        Street:     req.BillingAddress.Street,
        City:       req.BillingAddress.City,
        State:      req.BillingAddress.State,
        PostalCode: req.BillingAddress.PostalCode,
        Country:    req.BillingAddress.Country,
    }

    // Call service layer
    order, err := s.orderService.CreateOrder(
        req.CustomerId,
        req.CustomerName,
        req.CustomerEmail,
        shippingAddr,
        billingAddr,
        items,
    )
    if err != nil {
        return nil, err
    }

    // Convert domain order to proto
    return &pb.CreateOrderResponse{
        Order: domainOrderToProto(order),
    }, nil
}

func (s *Server) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.GetOrderResponse, error) {
    order, err := s.orderService.GetOrder(req.Id)
    if err != nil {
        return nil, err
    }

    return &pb.GetOrderResponse{
        Order: domainOrderToProto(order),
    }, nil
}

func (s *Server) ListOrders(ctx context.Context, req *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
    orders, total, err := s.orderService.ListOrders(
        int(req.Page),
        int(req.PageSize),
        req.CustomerId,
        req.Status,
    )
    if err != nil {
        return nil, err
    }

    protoOrders := make([]*pb.Order, len(orders))
    for i, order := range orders {
        protoOrders[i] = domainOrderToProto(order)
    }

    return &pb.ListOrdersResponse{
        Orders:   protoOrders,
        Total:    int32(total),
        Page:     req.Page,
        PageSize: req.PageSize,
    }, nil
}

func (s *Server) CancelOrder(ctx context.Context, req *pb.CancelOrderRequest) (*pb.CancelOrderResponse, error) {
    order, err := s.orderService.CancelOrder(req.Id)
    if err != nil {
        return nil, err
    }

    return &pb.CancelOrderResponse{
        Order: domainOrderToProto(order),
    }, nil
}

// Helper function to convert domain order to proto
func domainOrderToProto(order *models.Order) *pb.Order {
    items := make([]*pb.OrderItem, len(order.Items))
    for i, item := range order.Items {
        items[i] = &pb.OrderItem{
            Id:          item.ID,
            OrderId:     item.OrderID,
            ProductId:   item.ProductID,
            ProductName: item.ProductName,
            Quantity:    int32(item.Quantity),
            UnitPrice:   item.UnitPrice,
            TotalPrice:  item.TotalPrice,
        }
    }

    return &pb.Order{
        Id:            order.ID,
        CustomerId:    order.CustomerID,
        CustomerName:  order.CustomerName,
        CustomerEmail: order.CustomerEmail,
        ShippingAddress: &pb.Address{
            Street:     order.ShippingAddress.Street,
            City:       order.ShippingAddress.City,
            State:      order.ShippingAddress.State,
            PostalCode: order.ShippingAddress.PostalCode,
            Country:    order.ShippingAddress.Country,
        },
        BillingAddress: &pb.Address{
            Street:     order.BillingAddress.Street,
            City:       order.BillingAddress.City,
            State:      order.BillingAddress.State,
            PostalCode: order.BillingAddress.PostalCode,
            Country:    order.BillingAddress.Country,
        },
        Items:          items,
        TotalAmount:    order.TotalAmount,
        Status:         order.Status,
        TrackingNumber: order.TrackingNumber,
        CreatedAt:      timestamppb.New(order.CreatedAt),
        UpdatedAt:      timestamppb.New(order.UpdatedAt),
    }
}
```

### Step 4: Update main.go

Update `services/order-service/cmd/main.go` to start gRPC server:

```go
package main

import (
    "log"
    "net"

    "github.com/gin-gonic/gin"
    "google.golang.org/grpc"
    "google.golang.org/grpc/reflection"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"

    grpcServer "github.com/joshbarros/golang-logistics-services/services/order-service/internal/grpc"
    "github.com/joshbarros/golang-logistics-services/services/order-service/internal/handlers"
    "github.com/joshbarros/golang-logistics-services/services/order-service/internal/repository"
    "github.com/joshbarros/golang-logistics-services/services/order-service/internal/service"
    pb "github.com/joshbarros/golang-logistics-services/proto/order/v1"
)

func main() {
    // Database setup
    dsn := "host=localhost user=postgres password=postgres dbname=order_db port=5432 sslmode=disable"
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }

    // Auto-migrate
    db.AutoMigrate(&models.Order{}, &models.OrderItem{})

    // Initialize layers
    repo := repository.NewPostgresOrderRepository(db)
    svc := service.NewOrderService(repo)
    handler := handlers.NewHTTPHandler(svc)

    // Start gRPC server in goroutine
    go startGRPCServer(svc)

    // Start HTTP server
    router := gin.Default()
    handlers.SetupRoutes(router, handler)

    log.Println("HTTP server starting on :8080")
    router.Run(":8080")
}

func startGRPCServer(svc *service.OrderService) {
    listener, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }

    grpcSrv := grpc.NewServer()
    pb.RegisterOrderServiceServer(grpcSrv, grpcServer.NewServer(svc))

    // Enable reflection for grpcurl
    reflection.Register(grpcSrv)

    log.Println("gRPC server starting on :50051")
    if err := grpcSrv.Serve(listener); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}
```

### Step 5: Add gRPC Dependencies

Update `go.mod` to include:

```go
require (
    google.golang.org/grpc v1.60.0
    google.golang.org/protobuf v1.31.0
)
```

Run:
```bash
go mod tidy
```

### Step 6: Test gRPC

```bash
# Install grpcurl
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# List services
grpcurl -plaintext localhost:50051 list

# Create order
grpcurl -plaintext \
  -d '{
    "customer_id": "cust-123",
    "customer_name": "John Doe",
    "customer_email": "john@example.com",
    "shipping_address": {
      "street": "123 Main St",
      "city": "Boston",
      "state": "MA",
      "postal_code": "02101",
      "country": "USA"
    },
    "billing_address": {
      "street": "123 Main St",
      "city": "Boston",
      "state": "MA",
      "postal_code": "02101",
      "country": "USA"
    },
    "items": [{
      "product_id": "prod-1",
      "product_name": "Widget",
      "quantity": 2,
      "unit_price": 10.00
    }]
  }' \
  localhost:50051 \
  order.v1.OrderService/CreateOrder
```

## Repeat for All Services

Apply the same pattern for:
- ✅ Order Service (port 50051)
- ⏳ Shipment Service (port 50052)
- ⏳ Inventory Service (port 50053)
- ⏳ Route Service (port 50054)
- ⏳ Driver Service (port 50055)
- ⏳ Notification Service (port 50056)

## gRPC Middleware (Recommended)

Add interceptors for:

### Logging Interceptor

```go
func LoggingInterceptor() grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        start := time.Now()

        resp, err := handler(ctx, req)

        log.Printf("gRPC %s took %v, error: %v", info.FullMethod, time.Since(start), err)

        return resp, err
    }
}
```

### Recovery Interceptor

```go
func RecoveryInterceptor() grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
        defer func() {
            if r := recover(); r != nil {
                log.Printf("Recovered from panic in %s: %v", info.FullMethod, r)
                err = status.Errorf(codes.Internal, "internal server error")
            }
        }()

        return handler(ctx, req)
    }
}
```

### Apply Interceptors

```go
grpcSrv := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        LoggingInterceptor(),
        RecoveryInterceptor(),
    ),
)
```

## Expected Benefits

Once implemented, gRPC will provide:

1. **Performance:** 7-10x faster than JSON/REST
2. **Type Safety:** Compile-time checking
3. **Streaming:** Bidirectional communication
4. **Code Generation:** Automatic client/server stubs
5. **Load Balancing:** Built-in support
6. **Interoperability:** Multiple language support

## Estimated Implementation Time

- Proto generation: 5 minutes
- Order Service gRPC: 2 hours
- Shipment Service gRPC: 1.5 hours
- Inventory Service gRPC: 1.5 hours
- Route Service gRPC: 1 hour
- Driver Service gRPC: 1 hour
- Notification Service gRPC: 1 hour
- Testing & debugging: 2 hours

**Total: ~10 hours**

## Next Steps After gRPC

1. Add gRPC authentication (JWT in metadata)
2. Implement gRPC-Gateway for REST-to-gRPC translation
3. Add OpenTelemetry tracing
4. Implement circuit breakers
5. Add rate limiting
6. Write gRPC integration tests
