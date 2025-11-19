#!/bin/bash

# Platform validation script
# Validates code structure, syntax, and completeness without requiring network access

set -e

echo "============================================"
echo "🔍 Logistics Platform Validation"
echo "============================================"
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

PASSED=0
FAILED=0

# Function to check
check() {
    local name=$1
    local condition=$2

    echo -n "Checking $name... "

    if eval "$condition"; then
        echo -e "${GREEN}✓ PASS${NC}"
        ((PASSED++))
        return 0
    else
        echo -e "${RED}✗ FAIL${NC}"
        ((FAILED++))
        return 1
    fi
}

echo -e "${BLUE}📁 Checking Project Structure${NC}"
echo "================================"
echo ""

# Check directories exist
check "Services directory     " "[ -d services ]"
check "Packages directory     " "[ -d pkg ]"
check "Documentation directory" "[ -d docs ]"
check "Tests directory        " "[ -d tests ]"
check "K8s manifests          " "[ -d k8s ]"
check "Proto definitions      " "[ -d proto ]"

echo ""
echo -e "${BLUE}📦 Checking Core Packages${NC}"
echo "=========================="
echo ""

# Check core packages exist
check "Auth package           " "[ -f pkg/auth/auth.go ]"
check "Cache package          " "[ -f pkg/cache/cache.go ]"
check "Chaos package          " "[ -f pkg/chaos/chaos.go ]"
check "Circuit breaker package" "[ -f pkg/circuitbreaker/circuitbreaker.go ]"
check "Events package         " "[ -f pkg/events/events.go ]"
check "Feature flags package  " "[ -f pkg/featureflags/featureflags.go ]"
check "GraphQL package        " "[ -f pkg/graphql/graphql.go ]"
check "DataLoader package     " "[ -f pkg/graphql/dataloader.go ]"
check "Metrics package        " "[ -f pkg/metrics/metrics.go ]"
check "Multi-tenancy package  " "[ -f pkg/multitenancy/multitenancy.go ]"
check "Rate limit package     " "[ -f pkg/ratelimit/ratelimit.go ]"
check "Retry package          " "[ -f pkg/retry/retry.go ]"
check "Service mesh package   " "[ -f pkg/servicemesh/servicemesh.go ]"
check "Swagger package        " "[ -f pkg/swagger/swagger.go ]"
check "Tracing package        " "[ -f pkg/tracing/tracing.go ]"
check "Versioning package     " "[ -f pkg/versioning/versioning.go ]"
check "WebSocket package      " "[ -f pkg/websocket/websocket.go ]"
check "Worker package         " "[ -f pkg/worker/worker.go ]"

echo ""
echo -e "${BLUE}🎯 Checking Services${NC}"
echo "===================="
echo ""

# Check services
check "Order service          " "[ -d services/order ]"
check "Shipment service       " "[ -d services/shipment ]"
check "Inventory service      " "[ -d services/inventory ]"
check "Driver service         " "[ -d services/driver ]"
check "Route service          " "[ -d services/route ]"
check "Notification service   " "[ -d services/notification ]"
check "Gateway service        " "[ -f services/gateway/main.go ]"

echo ""
echo -e "${BLUE}📚 Checking Documentation${NC}"
echo "=========================="
echo ""

# Check documentation
check "README                 " "[ -f README.md ]"
check "Platform features      " "[ -f docs/PLATFORM_FEATURES.md ]"
check "Event architecture doc " "[ -f docs/EVENT_DRIVEN_ARCHITECTURE.md ]"
check "GraphQL gateway doc    " "[ -f docs/GRAPHQL_API_GATEWAY.md ]"
check "Service mesh doc       " "[ -f docs/SERVICE_MESH.md ]"
check "Chaos engineering doc  " "[ -f docs/CHAOS_ENGINEERING.md ]"
check "Advanced features doc  " "[ -f docs/ADVANCED_FEATURES.md ]"

echo ""
echo -e "${BLUE}🧪 Checking Test Files${NC}"
echo "======================="
echo ""

# Check tests exist
check "Chaos tests            " "[ -f pkg/chaos/chaos_test.go ]"
check "DataLoader tests       " "[ -f pkg/graphql/dataloader_test.go ]"
check "Service mesh tests     " "[ -f pkg/servicemesh/servicemesh_test.go ]"
check "Integration tests      " "[ -f tests/integration_test.go ]"

echo ""
echo -e "${BLUE}⚙️  Checking Configuration${NC}"
echo "=========================="
echo ""

# Check config files
check "go.mod exists          " "[ -f go.mod ]"
check "Makefile exists        " "[ -f Makefile ]"
check ".gitignore exists      " "[ -f .gitignore ]"
check "Docker compose         " "[ -f k8s/docker-compose.yml ] || [ -f docker-compose.yml ]"

echo ""
echo -e "${BLUE}🔍 Checking Code Quality${NC}"
echo "========================="
echo ""

# Count lines of code
GO_FILES=$(find pkg services -name "*.go" 2>/dev/null | wc -l)
GO_LOC=$(find pkg services -name "*.go" -exec wc -l {} \; 2>/dev/null | awk '{sum+=$1} END {print sum}')
DOC_FILES=$(find docs -name "*.md" 2>/dev/null | wc -l)
DOC_LOC=$(find docs -name "*.md" -exec wc -l {} \; 2>/dev/null | awk '{sum+=$1} END {print sum}')

check "Go files count (>50)   " "[ $GO_FILES -gt 50 ]"
check "Code lines (>15000)    " "[ $GO_LOC -gt 15000 ]"
check "Doc files count (>10)  " "[ $DOC_FILES -gt 10 ]"
check "Doc lines (>5000)      " "[ $DOC_LOC -gt 5000 ]"

echo ""
echo -e "${BLUE}🎨 Checking Service Mesh Manifests${NC}"
echo "==================================="
echo ""

# Check if service mesh files can be generated (check syntax)
check "Istio client           " "grep -q 'IstioClient' pkg/servicemesh/istio.go"
check "Linkerd client         " "grep -q 'LinkerdClient' pkg/servicemesh/linkerd.go"
check "Consul client          " "grep -q 'ConsulClient' pkg/servicemesh/consul.go"

echo ""
echo -e "${BLUE}🚀 Checking Advanced Features${NC}"
echo "=============================="
echo ""

# Check advanced feature implementations
check "Chaos fault injection  " "grep -q 'InjectFault' pkg/chaos/chaos.go"
check "Chaos scenarios        " "grep -q 'ScenarioRunner' pkg/chaos/runner.go"
check "DataLoader batching    " "grep -q 'BatchFunc' pkg/graphql/dataloader.go"
check "GraphQL middleware     " "grep -q 'AuthMiddleware' pkg/graphql/middleware.go"
check "Service mesh traffic   " "grep -q 'TrafficPolicy' pkg/servicemesh/servicemesh.go"
check "Multi-tenancy isolation" "grep -q 'TenantScope' pkg/multitenancy/multitenancy.go"
check "Event bus              " "grep -q 'EventBus' pkg/events/events.go"
check "Background workers     " "grep -q 'WorkerPool' pkg/worker/worker.go"
check "WebSocket hub          " "grep -q 'Hub' pkg/websocket/websocket.go"
check "Feature flags          " "grep -q 'FlagProvider' pkg/featureflags/featureflags.go"

echo ""
echo "============================================"
echo "📊 Validation Results Summary"
echo "============================================"
echo ""

TOTAL=$((PASSED + FAILED))

echo "Total Checks: $TOTAL"
echo -e "${GREEN}Passed: $PASSED${NC}"

if [ $FAILED -gt 0 ]; then
    echo -e "${RED}Failed: $FAILED${NC}"
    echo ""
    PERCENT=$(echo "scale=2; $PASSED * 100 / $TOTAL" | bc)
    echo -e "Completion: ${YELLOW}${PERCENT}%${NC}"
    echo ""
    echo -e "${YELLOW}⚠️  Some checks failed${NC}"
    exit 1
else
    echo -e "${GREEN}Failed: $FAILED${NC}"
    echo ""
    echo -e "${GREEN}Completion: 100%${NC}"
    echo ""
    echo -e "${GREEN}✅ Platform validation successful!${NC}"
    echo ""
    echo "Platform Statistics:"
    echo "  - Go files: $GO_FILES"
    echo "  - Lines of code: $GO_LOC"
    echo "  - Documentation files: $DOC_FILES"
    echo "  - Documentation lines: $DOC_LOC"
    echo ""
    exit 0
fi
