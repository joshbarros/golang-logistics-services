# Tiltfile for Golang Logistics Services
# This file orchestrates the local development environment with Kubernetes + Tilt

# Allow Kubernetes context (update based on your setup)
allow_k8s_contexts(['docker-desktop', 'minikube', 'kind-logistics'])

# Load Kubernetes manifests
k8s_yaml('k8s/postgres.yaml')
k8s_yaml('k8s/order-service.yaml')
k8s_yaml('k8s/shipment-service.yaml')
k8s_yaml('k8s/inventory-service.yaml')
k8s_yaml('k8s/route-service.yaml')
k8s_yaml('k8s/driver-service.yaml')
k8s_yaml('k8s/notification-service.yaml')
k8s_yaml('k8s/api-gateway.yaml')

# Load Istio configurations (optional - comment out if Istio not installed)
# k8s_yaml('istio/gateway.yaml')
# k8s_yaml('istio/virtual-service.yaml')
# k8s_yaml('istio/destination-rules.yaml')
# k8s_yaml('istio/service-entry.yaml')

# PostgreSQL databases - no build needed
k8s_resource('postgres-orders', port_forwards='5432:5432')
k8s_resource('postgres-shipments', port_forwards='5433:5432')
k8s_resource('postgres-inventory', port_forwards='5434:5432')

# Order Service
docker_build(
    'order-service',
    context='.',
    dockerfile='services/order-service/Dockerfile',
    live_update=[
        sync('services/order-service', '/app'),
        run('cd /app && go build -o main ./cmd/main.go'),
        restart_container(),
    ]
)
k8s_resource(
    'order-service',
    port_forwards='8080:8080',
    resource_deps=['postgres-orders']
)

# Shipment Service
docker_build(
    'shipment-service',
    context='.',
    dockerfile='services/shipment-service/Dockerfile',
    live_update=[
        sync('services/shipment-service', '/app'),
        run('cd /app && go build -o main ./cmd/main.go'),
        restart_container(),
    ]
)
k8s_resource(
    'shipment-service',
    port_forwards='8081:8081',
    resource_deps=['postgres-shipments']
)

# Inventory Service
docker_build(
    'inventory-service',
    context='.',
    dockerfile='services/inventory-service/Dockerfile',
    live_update=[
        sync('services/inventory-service', '/app'),
        run('cd /app && go build -o main ./cmd/main.go'),
        restart_container(),
    ]
)
k8s_resource(
    'inventory-service',
    port_forwards='8082:8082',
    resource_deps=['postgres-inventory']
)

# Route Service
docker_build(
    'route-service',
    context='.',
    dockerfile='services/route-service/Dockerfile',
    live_update=[
        sync('services/route-service', '/app'),
        run('cd /app && go build -o main ./cmd/main.go'),
        restart_container(),
    ]
)
k8s_resource(
    'route-service',
    port_forwards='8083:8083'
)

# Driver Service
docker_build(
    'driver-service',
    context='.',
    dockerfile='services/driver-service/Dockerfile',
    live_update=[
        sync('services/driver-service', '/app'),
        run('cd /app && go build -o main ./cmd/main.go'),
        restart_container(),
    ]
)
k8s_resource(
    'driver-service',
    port_forwards='8084:8084'
)

# Notification Service
docker_build(
    'notification-service',
    context='.',
    dockerfile='services/notification-service/Dockerfile',
    live_update=[
        sync('services/notification-service', '/app'),
        run('cd /app && go build -o main ./cmd/main.go'),
        restart_container(),
    ]
)
k8s_resource(
    'notification-service',
    port_forwards='8085:8085'
)

# API Gateway
docker_build(
    'api-gateway',
    context='.',
    dockerfile='services/api-gateway/Dockerfile',
    live_update=[
        sync('services/api-gateway', '/app'),
        run('cd /app && go build -o main ./cmd/main.go'),
        restart_container(),
    ]
)
k8s_resource(
    'api-gateway',
    port_forwards='8000:8000',
    resource_deps=[
        'order-service',
        'shipment-service',
        'inventory-service',
        'route-service',
        'driver-service',
        'notification-service'
    ]
)

# Print helpful information
print("""
╔═══════════════════════════════════════════════════════════════╗
║       Golang Logistics Services - Tilt Development            ║
╠═══════════════════════════════════════════════════════════════╣
║                                                               ║
║  🚀 Services Running:                                         ║
║                                                               ║
║  📦 API Gateway:         http://localhost:8000                ║
║  📋 Order Service:       http://localhost:8080                ║
║  🚚 Shipment Service:    http://localhost:8081                ║
║  📊 Inventory Service:   http://localhost:8082                ║
║  🗺️  Route Service:       http://localhost:8083                ║
║  👤 Driver Service:      http://localhost:8084                ║
║  📧 Notification Svc:    http://localhost:8085                ║
║                                                               ║
║  🗄️  PostgreSQL DBs:                                          ║
║     Orders DB:           localhost:5432                       ║
║     Shipments DB:        localhost:5433                       ║
║     Inventory DB:        localhost:5434                       ║
║                                                               ║
║  📊 Tilt UI:             http://localhost:10350               ║
║                                                               ║
║  💡 Tips:                                                     ║
║  - Press 'space' to open Tilt UI in browser                  ║
║  - Edit code and see live updates!                           ║
║  - Check logs in Tilt UI for debugging                       ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝
""")
