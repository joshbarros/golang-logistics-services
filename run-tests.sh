#!/bin/bash

# Comprehensive test runner for logistics platform
# Tests all packages and integration scenarios

set -e

echo "================================"
echo "🧪 Logistics Platform Test Suite"
echo "================================"
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Track results
PASSED=0
FAILED=0

# Function to run tests
run_test() {
    local name=$1
    local path=$2

    echo -n "Testing $name... "

    if go test -v $path 2>&1 | grep -q "PASS\|ok"; then
        echo -e "${GREEN}✓ PASS${NC}"
        ((PASSED++))
        return 0
    else
        echo -e "${RED}✗ FAIL${NC}"
        ((FAILED++))
        return 1
    fi
}

echo "📦 Testing Core Packages"
echo "========================"
echo ""

# Test chaos engineering
run_test "Chaos Engineering     " "./pkg/chaos/..."

# Test GraphQL DataLoader
run_test "GraphQL DataLoader    " "./pkg/graphql/..."

# Test Service Mesh
run_test "Service Mesh          " "./pkg/servicemesh/..."

echo ""
echo "🔗 Testing Integration"
echo "======================"
echo ""

# Test integration scenarios
run_test "Integration Tests     " "./tests/..."

echo ""
echo "================================"
echo "📊 Test Results Summary"
echo "================================"
echo ""

TOTAL=$((PASSED + FAILED))

echo "Total Tests: $TOTAL"
echo -e "${GREEN}Passed: $PASSED${NC}"

if [ $FAILED -gt 0 ]; then
    echo -e "${RED}Failed: $FAILED${NC}"
    echo ""
    echo "❌ Some tests failed"
    exit 1
else
    echo -e "${GREEN}Failed: $FAILED${NC}"
    echo ""
    echo -e "${GREEN}✅ All tests passed!${NC}"
    exit 0
fi
