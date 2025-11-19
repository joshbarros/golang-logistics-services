.PHONY: help build run test clean docker-build k8s-deploy k8s-delete tilt-up tilt-down

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build all services
	@echo "Building all services..."
	@cd services/order-service && go build -o ../../bin/order-service ./cmd/main.go
	@cd services/shipment-service && go build -o ../../bin/shipment-service ./cmd/main.go
	@cd services/inventory-service && go build -o ../../bin/inventory-service ./cmd/main.go
	@cd services/route-service && go build -o ../../bin/route-service ./cmd/main.go
	@cd services/driver-service && go build -o ../../bin/driver-service ./cmd/main.go
	@cd services/notification-service && go build -o ../../bin/notification-service ./cmd/main.go
	@cd services/api-gateway && go build -o ../../bin/api-gateway ./cmd/main.go
	@echo "✅ All services built successfully!"

test: ## Run tests for all services
	@echo "Running tests..."
	@./test-all.sh

test-verbose: ## Run tests with verbose output
	@echo "Running tests (verbose)..."
	@cd services/order-service && go test -v ./...
	@cd services/shipment-service && go test -v ./...
	@cd services/inventory-service && go test -v ./...
	@cd services/route-service && go test -v ./...
	@cd services/driver-service && go test -v ./...
	@cd services/notification-service && go test -v ./...
	@cd services/api-gateway && go test -v ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@cd services/order-service && go test -v -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html
	@cd services/shipment-service && go test -v -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html
	@cd services/inventory-service && go test -v -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html
	@cd services/route-service && go test -v -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html
	@cd services/driver-service && go test -v -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html
	@cd services/notification-service && go test -v -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html
	@cd services/api-gateway && go test -v -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage reports generated (coverage.html in each service directory)"

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -rf dist/
	@echo "✅ Clean complete!"

docker-build: ## Build Docker images for all services
	@echo "Building Docker images..."
	@docker build -t order-service:latest -f services/order-service/Dockerfile .
	@docker build -t shipment-service:latest -f services/shipment-service/Dockerfile .
	@docker build -t inventory-service:latest -f services/inventory-service/Dockerfile .
	@docker build -t route-service:latest -f services/route-service/Dockerfile .
	@docker build -t driver-service:latest -f services/driver-service/Dockerfile .
	@docker build -t notification-service:latest -f services/notification-service/Dockerfile .
	@docker build -t api-gateway:latest -f services/api-gateway/Dockerfile .
	@echo "✅ All Docker images built!"

k8s-deploy: ## Deploy to Kubernetes
	@echo "Deploying to Kubernetes..."
	@kubectl apply -f k8s/
	@echo "✅ Deployed to Kubernetes!"

k8s-delete: ## Delete from Kubernetes
	@echo "Deleting from Kubernetes..."
	@kubectl delete -f k8s/
	@echo "✅ Deleted from Kubernetes!"

istio-deploy: ## Deploy Istio configurations
	@echo "Deploying Istio configurations..."
	@kubectl apply -f istio/
	@echo "✅ Istio configurations deployed!"

istio-delete: ## Delete Istio configurations
	@echo "Deleting Istio configurations..."
	@kubectl delete -f istio/
	@echo "✅ Istio configurations deleted!"

tilt-up: ## Start Tilt development environment
	@echo "Starting Tilt..."
	@tilt up

tilt-down: ## Stop Tilt development environment
	@echo "Stopping Tilt..."
	@tilt down

logs-order: ## View order service logs
	@kubectl logs -f -l app=order-service

logs-shipment: ## View shipment service logs
	@kubectl logs -f -l app=shipment-service

logs-inventory: ## View inventory service logs
	@kubectl logs -f -l app=inventory-service

logs-gateway: ## View API gateway logs
	@kubectl logs -f -l app=api-gateway

status: ## Check status of all services
	@echo "Checking service status..."
	@kubectl get pods
	@kubectl get services
	@echo ""
	@echo "API Gateway: http://localhost:8000/health"
	@curl -s http://localhost:8000/health | jq . || echo "API Gateway not available"
