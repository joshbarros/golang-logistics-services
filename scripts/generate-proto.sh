#!/bin/bash

# Script to generate Go code from protocol buffer definitions
# Requires: protoc, protoc-gen-go, protoc-gen-go-grpc

set -e

echo "=================================="
echo "Generating gRPC code from protos"
echo "=================================="
echo ""

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo "❌ protoc not found. Please install:"
    echo "   brew install protobuf  # macOS"
    echo "   apt install protobuf-compiler  # Ubuntu/Debian"
    exit 1
fi

# Check if protoc-gen-go is installed
if ! command -v protoc-gen-go &> /dev/null; then
    echo "❌ protoc-gen-go not found. Installing..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

# Check if protoc-gen-go-grpc is installed
if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo "❌ protoc-gen-go-grpc not found. Installing..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

# Navigate to project root
cd "$(dirname "$0")/.."

# Create output directory
mkdir -p proto/gen

# Generate for each service
echo "Generating Order Service protos..."
protoc \
    --go_out=. \
    --go_opt=paths=source_relative \
    --go-grpc_out=. \
    --go-grpc_opt=paths=source_relative \
    proto/order/v1/order.proto

echo "Generating Shipment Service protos..."
protoc \
    --go_out=. \
    --go_opt=paths=source_relative \
    --go-grpc_out=. \
    --go-grpc_opt=paths=source_relative \
    proto/shipment/v1/shipment.proto

echo "Generating Inventory Service protos..."
protoc \
    --go_out=. \
    --go_opt=paths=source_relative \
    --go-grpc_out=. \
    --go-grpc_opt=paths=source_relative \
    proto/inventory/v1/inventory.proto

echo "Generating Route Service protos..."
protoc \
    --go_out=. \
    --go_opt=paths=source_relative \
    --go-grpc_out=. \
    --go-grpc_opt=paths=source_relative \
    proto/route/v1/route.proto

echo "Generating Driver Service protos..."
protoc \
    --go_out=. \
    --go_opt=paths=source_relative \
    --go-grpc_out=. \
    --go-grpc_opt=paths=source_relative \
    proto/driver/v1/driver.proto

echo "Generating Notification Service protos..."
protoc \
    --go_out=. \
    --go_opt=paths=source_relative \
    --go-grpc_out=. \
    --go-grpc_opt=paths=source_relative \
    proto/notification/v1/notification.proto

echo ""
echo "✅ All proto files generated successfully!"
echo ""
echo "Generated files:"
find proto -name "*.pb.go" -o -name "*_grpc.pb.go"
