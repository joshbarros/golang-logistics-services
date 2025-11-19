.PHONY: help build run test clean docker-build k8s-deploy k8s-delete tilt-up tilt-down proto-gen proto-clean \
	lint fmt vet security-scan coverage validate-platform quality-gates pre-commit-install ci-local \
	migrate-up migrate-down migrate-create helm-package helm-install helm-upgrade

# Configuration
GO_VERSION := 1.21
GOLANGCI_LINT_VERSION := v1.55.2
SERVICES := order-service shipment-service inventory-service driver-service route-service notification-service gateway
BIN_DIR := bin
COVERAGE_THRESHOLD := 50

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# =============================================================================
# Development
# =============================================================================

deps: ## Download dependencies
	@echo "📦 Downloading Go dependencies..."
	@go mod download
	@go mod verify
	@echo "✅ Dependencies downloaded and verified"

fmt: ## Format code
	@echo "🎨 Formatting Go code..."
	@gofmt -w -s .
	@goimports -w -local github.com/joshbarros/golang-logistics-services .
	@echo "✅ Code formatted"

vet: ## Run go vet
	@echo "🔍 Running go vet..."
	@go vet ./...
	@echo "✅ Go vet passed"

# =============================================================================
# Code Quality
# =============================================================================

lint: ## Run golangci-lint
	@echo "🔍 Running golangci-lint..."
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
	}
	@golangci-lint run --config=.golangci.yml --timeout=5m
	@echo "✅ Linting passed"

lint-fix: ## Run golangci-lint with auto-fix
	@echo "🔧 Running golangci-lint with auto-fix..."
	@golangci-lint run --config=.golangci.yml --timeout=5m --fix
	@echo "✅ Linting complete with fixes applied"

# =============================================================================
# Security
# =============================================================================

security-scan: ## Run security scanners
	@echo "🔒 Running security scans..."
	@echo "Running gosec..."
	@command -v gosec >/dev/null 2>&1 || go install github.com/securego/gosec/v2/cmd/gosec@latest
	@gosec -fmt=json -out=gosec-report.json ./... || true
	@echo ""
	@echo "Running govulncheck..."
	@command -v govulncheck >/dev/null 2>&1 || go install golang.org/x/vuln/cmd/govulncheck@latest
	@govulncheck ./...
	@echo "✅ Security scans complete"

# =============================================================================
# Testing
# =============================================================================

coverage: ## Run tests with coverage
	@echo "📊 Running tests with coverage..."
	@go test -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -func=coverage.out
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print substr($$3, 1, length($$3)-1)}'); \
	echo ""; \
	echo "Total coverage: $${COVERAGE}%"; \
	if [ "$$(echo "$${COVERAGE} < $(COVERAGE_THRESHOLD)" | bc -l)" -eq 1 ]; then \
		echo "❌ Coverage $${COVERAGE}% is below minimum $(COVERAGE_THRESHOLD)%"; \
		exit 1; \
	else \
		echo "✅ Coverage $${COVERAGE}% meets minimum threshold"; \
	fi

coverage-html: coverage ## Generate HTML coverage report
	@echo "📊 Generating HTML coverage report..."
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

integration-test: ## Run integration tests
	@echo "🧪 Running integration tests..."
	@go test -v -tags=integration ./tests/...
	@echo "✅ Integration tests passed"

benchmark: ## Run benchmarks
	@echo "⚡ Running benchmarks..."
	@go test -bench=. -benchmem ./...
	@echo "✅ Benchmarks complete"

# =============================================================================
# Validation
# =============================================================================

validate-platform: ## Run platform validation script
	@echo "✅ Running platform validation..."
	@chmod +x validate-platform.sh
	@./validate-platform.sh

# =============================================================================
# Quality Gates (CI/CD)
# =============================================================================

quality-gates: fmt vet lint security-scan test coverage validate-platform ## Run all quality gates
	@echo ""
	@echo "================================"
	@echo "🎯 Quality Gates Summary"
	@echo "================================"
	@echo "✅ Format check"
	@echo "✅ Go vet"
	@echo "✅ Linting"
	@echo "✅ Security scan"
	@echo "✅ Tests"
	@echo "✅ Coverage"
	@echo "✅ Platform validation"
	@echo ""
	@echo "🎉 All quality gates passed!"

ci-local: quality-gates build ## Run full CI pipeline locally
	@echo ""
	@echo "================================"
	@echo "🚀 Local CI Complete"
	@echo "================================"
	@echo "✅ All checks passed"
	@echo "✅ All services built"
	@echo ""
	@echo "Ready to commit and push!"

# =============================================================================
# Pre-commit
# =============================================================================

pre-commit-install: ## Install pre-commit hooks
	@echo "🪝 Installing pre-commit hooks..."
	@command -v pre-commit >/dev/null 2>&1 || { \
		echo "Installing pre-commit..."; \
		pip install pre-commit || pip3 install pre-commit; \
	}
	@pre-commit install
	@pre-commit install --hook-type commit-msg
	@echo "✅ Pre-commit hooks installed"

pre-commit-run: ## Run pre-commit on all files
	@echo "🪝 Running pre-commit hooks..."
	@pre-commit run --all-files

# =============================================================================
# Database Migrations
# =============================================================================

migrate-install: ## Install golang-migrate
	@echo "📦 Installing golang-migrate..."
	@go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "✅ golang-migrate installed"

migrate-create: ## Create new migration (usage: make migrate-create NAME=create_users_table)
	@echo "📝 Creating migration: $(NAME)"
	@migrate create -ext sql -dir migrations -seq $(NAME)
	@echo "✅ Migration files created"

migrate-up: ## Run database migrations
	@echo "⬆️  Running migrations..."
	@migrate -path migrations -database "$(DATABASE_URL)" up
	@echo "✅ Migrations complete"

migrate-down: ## Rollback last migration
	@echo "⬇️  Rolling back migration..."
	@migrate -path migrations -database "$(DATABASE_URL)" down 1
	@echo "✅ Rollback complete"

migrate-version: ## Show current migration version
	@migrate -path migrations -database "$(DATABASE_URL)" version

# =============================================================================
# Protocol Buffers
# =============================================================================

proto-gen: ## Generate Go code from protocol buffers
	@echo "Generating gRPC code from proto files..."
	@./scripts/generate-proto.sh

proto-clean: ## Clean generated proto files
	@echo "Cleaning generated proto files..."
	@find proto -name "*.pb.go" -delete
	@find proto -name "*_grpc.pb.go" -delete
	@echo "✅ Proto files cleaned!"

build: proto-gen ## Build all services
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
