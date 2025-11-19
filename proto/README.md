# Protocol Buffers & gRPC

This directory contains Protocol Buffer definitions for all microservices in the logistics platform.

## Overview

Each service has its own proto package with versioned definitions:

- `order/v1` - Order management service
- `shipment/v1` - Shipment tracking service
- `inventory/v1` - Inventory management service
- `route/v1` - Route planning and optimization service
- `driver/v1` - Driver management service
- `notification/v1` - Notification service

## Prerequisites

Before generating code, you need to install:

```bash
# Install protoc (Protocol Buffer Compiler)
# macOS
brew install protobuf

# Ubuntu/Debian
sudo apt install protobuf-compiler

# Windows
# Download from https://github.com/protocolbuffers/protobuf/releases

# Install Go plugins for protoc
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Ensure plugins are in PATH
export PATH="$PATH:$(go env GOPATH)/bin"
```

## Generating Go Code

### Using Make (Recommended)

```bash
# Generate all proto files
make proto-gen

# Clean generated files
make proto-clean
```

### Using Script Directly

```bash
./scripts/generate-proto.sh
```

### Manual Generation

```bash
# Order Service
protoc \
    --go_out=. \
    --go_opt=paths=source_relative \
    --go-grpc_out=. \
    --go-grpc_opt=paths=source_relative \
    proto/order/v1/order.proto

# Repeat for other services...
```

## Generated Files

Each `.proto` file generates two Go files:

- `{service}.pb.go` - Message types and serialization code
- `{service}_grpc.pb.go` - gRPC server and client interfaces

## Service Definitions

### Order Service

**RPC Methods:**
- `CreateOrder` - Create a new order
- `GetOrder` - Retrieve order by ID
- `ListOrders` - List orders with pagination and filters
- `UpdateOrderStatus` - Update order status
- `CancelOrder` - Cancel an order

**Port:** 50051

### Shipment Service

**RPC Methods:**
- `CreateShipment` - Create new shipment
- `GetShipment` - Get shipment by ID
- `TrackShipment` - Track by tracking number
- `UpdateShipmentStatus` - Update status with location
- `ListShipments` - List with filters

**Port:** 50052

### Inventory Service

**RPC Methods:**
- `CreateItem` - Add inventory item
- `GetItem` - Get item by ID
- `ListItems` - List with pagination
- `UpdateStock` - Adjust stock levels
- `CheckAvailability` - Check product availability
- `GetLowStockItems` - Get items below reorder level

**Port:** 50053

### Route Service

**RPC Methods:**
- `OptimizeRoute` - Optimize delivery route
- `GetRoute` - Get route by ID
- `ListRoutes` - List routes with filters
- `CalculateDistance` - Calculate distance between points

**Port:** 50054

### Driver Service

**RPC Methods:**
- `CreateDriver` - Register new driver
- `GetDriver` - Get driver by ID
- `ListDrivers` - List drivers with filters
- `UpdateDriver` - Update driver info
- `AssignDriver` - Assign driver to route
- `GetAvailableDrivers` - Find available drivers nearby
- `UpdateDriverLocation` - Update real-time location

**Port:** 50055

### Notification Service

**RPC Methods:**
- `SendNotification` - Send single notification
- `GetNotification` - Get notification by ID
- `ListNotifications` - List with filters
- `SendBulkNotifications` - Send multiple notifications

**Port:** 50056

## Using gRPC Services

### Server Implementation

After generating code, implement the server interface:

```go
import (
    pb "github.com/joshbarros/golang-logistics-services/proto/order/v1"
    "google.golang.org/grpc"
)

type orderServer struct {
    pb.UnimplementedOrderServiceServer
    service *service.OrderService
}

func (s *orderServer) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
    // Convert proto request to domain model
    // Call service layer
    // Convert domain model to proto response
    return &pb.CreateOrderResponse{}, nil
}
```

### Client Usage

```go
import (
    pb "github.com/joshbarros/golang-logistics-services/proto/order/v1"
    "google.golang.org/grpc"
)

conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

client := pb.NewOrderServiceClient(conn)

resp, err := client.CreateOrder(context.Background(), &pb.CreateOrderRequest{
    CustomerId: "cust-123",
    CustomerName: "John Doe",
    // ... other fields
})
```

## Testing gRPC

### Using grpcurl

```bash
# Install grpcurl
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# List services
grpcurl -plaintext localhost:50051 list

# List methods
grpcurl -plaintext localhost:50051 list order.v1.OrderService

# Call method
grpcurl -plaintext \
  -d '{"customer_id": "cust-123", "customer_name": "John Doe"}' \
  localhost:50051 \
  order.v1.OrderService/CreateOrder
```

### Using BloomRPC

Download BloomRPC GUI client: https://github.com/bloomrpc/bloomrpc

1. Import proto files
2. Connect to localhost:50051
3. Test methods interactively

## Proto Style Guide

### Naming Conventions

- **Services:** PascalCase ending in "Service" (e.g., `OrderService`)
- **RPC Methods:** PascalCase verbs (e.g., `CreateOrder`, `GetOrder`)
- **Messages:** PascalCase (e.g., `CreateOrderRequest`)
- **Fields:** snake_case (e.g., `customer_id`, `total_amount`)

### Versioning

- All protos are in v1 packages
- Breaking changes require new version (v2, v3, etc.)
- Maintain backward compatibility when possible

### Field Numbers

- 1-15: Most frequently used fields (1 byte encoding)
- 16-2047: Less frequent fields (2 byte encoding)
- Never reuse field numbers

## gRPC vs REST

Both protocols are supported:

| Feature | gRPC | REST |
|---------|------|------|
| Protocol | HTTP/2 | HTTP/1.1 |
| Payload | Protocol Buffers | JSON |
| Performance | Faster (binary) | Slower (text) |
| Browser Support | Limited | Full |
| Streaming | Bidirectional | Server-sent only |
| Use Case | Service-to-service | Client-to-server |

### When to Use gRPC

- Internal microservice communication
- High-performance requirements
- Bidirectional streaming needed
- Strong typing required

### When to Use REST

- Public APIs
- Browser-based clients
- Simple CRUD operations
- Wide compatibility needed

## Next Steps

1. **Generate proto files:** `make proto-gen`
2. **Implement gRPC servers** in each service
3. **Add gRPC middleware** (logging, auth, metrics)
4. **Write gRPC tests**
5. **Add service reflection** for development
6. **Implement gRPC gateway** for REST-to-gRPC translation

## Resources

- [gRPC Go Quickstart](https://grpc.io/docs/languages/go/quickstart/)
- [Protocol Buffers Guide](https://developers.google.com/protocol-buffers/docs/proto3)
- [gRPC Best Practices](https://grpc.io/docs/guides/performance/)
- [Buf Build System](https://buf.build/)
