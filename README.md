# Golang Logistics Services

A comprehensive microservices-based logistics platform built with Go, featuring order management, shipment tracking, inventory management, route planning, driver management, and notifications.

## Architecture

### Microservices
- **Order Service**: Manage customer orders and order lifecycle
- **Shipment Service**: Track shipments and delivery status
- **Inventory Service**: Warehouse and inventory management
- **Route Service**: Optimize delivery routes and planning
- **Driver Service**: Manage drivers and assignments
- **Notification Service**: Send notifications (email, SMS, push)
- **API Gateway**: Single entry point for external clients

### Technology Stack
- **Language**: Go 1.21+
- **Web Framework**: Gin
- **RPC**: gRPC (internal service communication)
- **API**: REST (external clients)
- **Database**: PostgreSQL (per service)
- **Service Mesh**: Istio
- **Orchestration**: Kubernetes
- **Local Development**: Tilt

## Project Structure

```
golang-logistics-services/
├── services/
│   ├── order-service/         # Order management
│   ├── shipment-service/      # Shipment tracking
│   ├── inventory-service/     # Inventory management
│   ├── route-service/         # Route planning
│   ├── driver-service/        # Driver management
│   ├── notification-service/  # Notifications
│   └── api-gateway/           # API Gateway
├── proto/                     # Shared protobuf definitions
├── k8s/                       # Kubernetes manifests
├── istio/                     # Istio configurations
├── Tiltfile                   # Tilt configuration
└── README.md
```

## Prerequisites

- Go 1.21 or higher
- Docker
- Kubernetes (local: Docker Desktop, minikube, or kind)
- Tilt (https://tilt.dev/)
- kubectl
- Istio (optional, for service mesh features)

## Getting Started

### 1. Clone the Repository
```bash
git clone https://github.com/joshbarros/golang-logistics-services.git
cd golang-logistics-services
```

### 2. Install Dependencies
```bash
go mod download
```

### 3. Set Up Local Kubernetes

#### Option A: Docker Desktop
- Enable Kubernetes in Docker Desktop settings

#### Option B: Minikube
```bash
minikube start --cpus=4 --memory=8192
```

#### Option C: Kind
```bash
kind create cluster --name logistics
```

### 4. Install Istio (Optional)
```bash
curl -L https://istio.io/downloadIstio | sh -
cd istio-*
export PATH=$PWD/bin:$PATH
istioctl install --set profile=demo -y
kubectl label namespace default istio-injection=enabled
```

### 5. Start with Tilt
```bash
tilt up
```

Open the Tilt UI at http://localhost:10350

## API Documentation

### API Gateway Endpoints

#### Orders
- `POST /api/v1/orders` - Create new order
- `GET /api/v1/orders/:id` - Get order details
- `GET /api/v1/orders` - List all orders
- `PUT /api/v1/orders/:id` - Update order
- `DELETE /api/v1/orders/:id` - Cancel order

#### Shipments
- `POST /api/v1/shipments` - Create shipment
- `GET /api/v1/shipments/:id` - Get shipment details
- `GET /api/v1/shipments/track/:tracking_number` - Track shipment
- `PUT /api/v1/shipments/:id/status` - Update shipment status

#### Inventory
- `POST /api/v1/inventory/items` - Add inventory item
- `GET /api/v1/inventory/items/:id` - Get item details
- `GET /api/v1/inventory/items` - List inventory
- `PUT /api/v1/inventory/items/:id` - Update item
- `PUT /api/v1/inventory/items/:id/stock` - Update stock level

#### Routes
- `POST /api/v1/routes/optimize` - Calculate optimal route
- `GET /api/v1/routes/:id` - Get route details
- `GET /api/v1/routes` - List routes

#### Drivers
- `POST /api/v1/drivers` - Add driver
- `GET /api/v1/drivers/:id` - Get driver details
- `GET /api/v1/drivers` - List drivers
- `PUT /api/v1/drivers/:id` - Update driver
- `PUT /api/v1/drivers/:id/assign` - Assign driver to route

#### Notifications
- `POST /api/v1/notifications/send` - Send notification

## Service Communication

Services communicate internally via gRPC for high performance:
- Order Service → Inventory Service (stock validation)
- Order Service → Notification Service (order confirmations)
- Shipment Service → Notification Service (shipping updates)
- Route Service → Driver Service (assignments)

## Database Schema

Each service has its own PostgreSQL database:
- `orders_db`
- `shipments_db`
- `inventory_db`
- `routes_db`
- `drivers_db`
- `notifications_db`

## Development

### Running Individual Services

```bash
# Order Service
cd services/order-service
go run cmd/main.go

# Shipment Service
cd services/shipment-service
go run cmd/main.go

# ... and so on
```

### Building Docker Images

```bash
# Build all services
docker build -t order-service:latest -f services/order-service/Dockerfile .
docker build -t shipment-service:latest -f services/shipment-service/Dockerfile .
# ... etc
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests for specific service
go test ./services/order-service/...
```

## Deployment

### Deploy to Kubernetes

```bash
# Apply all manifests
kubectl apply -f k8s/

# Apply Istio configurations
kubectl apply -f istio/
```

### Check Service Status

```bash
kubectl get pods
kubectl get services
kubectl get virtualservices
```

## Monitoring and Observability

With Istio enabled, access:
- **Kiali** (Service Graph): `istioctl dashboard kiali`
- **Jaeger** (Tracing): `istioctl dashboard jaeger`
- **Grafana** (Metrics): `istioctl dashboard grafana`
- **Prometheus**: `istioctl dashboard prometheus`

## Configuration

Each service can be configured via environment variables:

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=service_db

# Service Ports
HTTP_PORT=8080
GRPC_PORT=9090

# Service Discovery
ORDER_SERVICE_ADDR=order-service:9090
SHIPMENT_SERVICE_ADDR=shipment-service:9090
# ... etc
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License.

## Support

For issues and questions, please open an issue on GitHub.
