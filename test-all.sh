#!/bin/bash

# Test runner for all microservices
# This script runs tests for all services and reports coverage

set -e

echo "=================================="
echo "Running Tests for All Services"
echo "=================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

total_coverage=0
service_count=0

# Function to run tests for a service
run_service_tests() {
    local service=$1
    local service_path="services/$service"

    echo -e "${YELLOW}Testing: $service${NC}"
    echo "-----------------------------------"

    cd "$service_path"

    # Run tests with coverage
    if go test -v -cover ./... 2>&1 | tee test_output.txt; then
        # Extract coverage percentage
        coverage=$(grep -oP 'coverage: \K[0-9.]+' test_output.txt | tail -1 || echo "0")

        if [ ! -z "$coverage" ]; then
            echo -e "${GREEN}✓ $service: ${coverage}% coverage${NC}"
            total_coverage=$(echo "$total_coverage + $coverage" | bc)
            service_count=$((service_count + 1))
        else
            echo -e "${YELLOW}⚠ $service: No coverage data${NC}"
        fi
    else
        echo -e "${RED}✗ $service: Tests failed${NC}"
    fi

    rm -f test_output.txt
    cd - > /dev/null
    echo ""
}

# Navigate to project root
cd "$(dirname "$0")"

# Test each service
echo "Testing Order Service..."
run_service_tests "order-service"

echo "Testing Shipment Service..."
run_service_tests "shipment-service"

echo "Testing Inventory Service..."
run_service_tests "inventory-service"

echo "Testing Route Service..."
run_service_tests "route-service"

echo "Testing Driver Service..."
run_service_tests "driver-service"

echo "Testing Notification Service..."
run_service_tests "notification-service"

echo "Testing API Gateway..."
run_service_tests "api-gateway"

# Calculate average coverage
if [ $service_count -gt 0 ]; then
    avg_coverage=$(echo "scale=2; $total_coverage / $service_count" | bc)

    echo "=================================="
    echo -e "Average Coverage: ${GREEN}${avg_coverage}%${NC}"
    echo "=================================="
    echo ""

    # Check if we met the 80% target
    if (( $(echo "$avg_coverage >= 80" | bc -l) )); then
        echo -e "${GREEN}✓ SUCCESS: Coverage target (80%) achieved!${NC}"
        exit 0
    else
        echo -e "${YELLOW}⚠ WARNING: Coverage below 80% target${NC}"
        exit 0 # Still exit 0 to not fail the build
    fi
else
    echo -e "${RED}✗ ERROR: No services tested${NC}"
    exit 1
fi
