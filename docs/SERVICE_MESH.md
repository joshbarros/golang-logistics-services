# Service Mesh Integration

Service mesh provides infrastructure-level traffic management, security, and observability for microservices without requiring application code changes.

## Supported Service Meshes

- **Istio** - Full-featured service mesh with advanced traffic management
- **Linkerd** - Lightweight, simple, and fast service mesh
- **Consul Connect** - HashiCorp's service mesh solution

## Features

### 1. Traffic Management

- **Load Balancing**: Round-robin, least-request, random, consistent hashing
- **Traffic Splitting**: Canary deployments, A/B testing, blue-green deployments
- **Circuit Breaking**: Automatic failure detection and isolation
- **Retries & Timeouts**: Configurable retry policies and request timeouts
- **Connection Pooling**: Limit concurrent connections and requests

### 2. Security

- **Mutual TLS (mTLS)**: Automatic encryption of service-to-service communication
- **Authentication**: Service identity and authentication
- **Authorization**: Fine-grained access control between services
- **Certificate Management**: Automatic certificate rotation

### 3. Observability

- **Distributed Tracing**: Request flow across services
- **Metrics**: Latency, traffic, errors, saturation (Golden Signals)
- **Service Graph**: Visual representation of service dependencies
- **Access Logs**: Detailed logs of all requests

## Usage

### Basic Setup

```go
import "golang-logistics-services/pkg/servicemesh"

// Initialize service mesh manager
config := servicemesh.DefaultConfig()
config.ServiceName = "order-service"
config.ServiceVersion = "v1.0.0"
config.Namespace = "production"
config.EnableMTLS = true
config.MeshType = "istio" // or "linkerd" or "consul"

manager, err := servicemesh.NewManager(config)
if err != nil {
    log.Fatal(err)
}

// Apply traffic policy
ctx := context.Background()
if err := manager.ApplyTrafficPolicy(ctx); err != nil {
    log.Printf("Failed to apply traffic policy: %v", err)
}

// Enable mTLS
if err := manager.EnableMTLS(ctx); err != nil {
    log.Printf("Failed to enable mTLS: %v", err)
}
```

### Canary Deployment

Deploy new version gradually:

```go
// Start with 10% traffic to canary
err := manager.CreateCanaryDeployment(
    ctx,
    "order-service",
    "v1.0.0",  // stable version
    "v1.1.0",  // canary version
    10,        // 10% traffic weight
)

// Monitor metrics...

// Increase to 25%
manager.CreateCanaryDeployment(ctx, "order-service", "v1.0.0", "v1.1.0", 25)

// Increase to 50%
manager.CreateCanaryDeployment(ctx, "order-service", "v1.0.0", "v1.1.0", 50)

// Full rollout (100%)
manager.CreateCanaryDeployment(ctx, "order-service", "v1.0.0", "v1.1.0", 100)
```

### Blue-Green Deployment

Instant switch between versions:

```go
// Deploy v2 (green) alongside v1 (blue)
// Route 100% traffic to blue
err := manager.CreateBlueGreenDeployment(ctx, "order-service", "v1")

// Test green deployment...

// Switch to green
err = manager.CreateBlueGreenDeployment(ctx, "order-service", "v2")

// Rollback to blue if needed
err = manager.CreateBlueGreenDeployment(ctx, "order-service", "v1")
```

### A/B Testing

Split traffic between multiple versions:

```go
// 50/50 split between v1 and v2
err := manager.CreateABTestDeployment(
    ctx,
    "order-service",
    map[string]int{
        "v1": 50,
        "v2": 50,
    },
)

// 70/30 split
err = manager.CreateABTestDeployment(
    ctx,
    "order-service",
    map[string]int{
        "v1": 70,
        "v2": 30,
    },
)

// Three-way split
err = manager.CreateABTestDeployment(
    ctx,
    "order-service",
    map[string]int{
        "v1": 40,
        "v2": 40,
        "v3": 20,
    },
)
```

### Fault Injection (Chaos Testing)

Test resilience by injecting faults:

```go
// Inject 5 second delay for 10% of requests
err := manager.InjectFault(
    ctx,
    "order-service",
    &servicemesh.Delay{
        Percentage: 10.0,
        FixedDelay: 5 * time.Second,
    },
    nil,
)

// Inject HTTP 503 errors for 5% of requests
err = manager.InjectFault(
    ctx,
    "order-service",
    nil,
    &servicemesh.Abort{
        Percentage: 5.0,
        HTTPStatus: 503,
    },
)

// Combined: delay AND errors
err = manager.InjectFault(
    ctx,
    "order-service",
    &servicemesh.Delay{
        Percentage: 10.0,
        FixedDelay: 3 * time.Second,
    },
    &servicemesh.Abort{
        Percentage: 5.0,
        HTTPStatus: 500,
    },
)
```

## Istio Configuration

### Generate Istio Manifests

```go
config := servicemesh.DefaultConfig()
config.ServiceName = "order-service"
config.MeshType = "istio"

manifests := servicemesh.GenerateIstioManifests(config)

// Write to files
for filename, content := range manifests {
    os.WriteFile(filepath.Join("k8s", filename), []byte(content), 0644)
}
```

### Apply Istio Resources

```bash
# Apply all manifests
kubectl apply -f k8s/

# Verify installation
kubectl get virtualservices
kubectl get destinationrules
kubectl get peerauthentications
kubectl get gateways
```

### Example Istio VirtualService

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: order-service
  namespace: production
spec:
  hosts:
  - order-service
  http:
  - match:
    - uri:
        prefix: /api/v2
    route:
    - destination:
        host: order-service
        subset: v2
      weight: 100
  - route:
    - destination:
        host: order-service
        subset: v1
      weight: 90
    - destination:
        host: order-service
        subset: v2
      weight: 10
```

### Example Istio DestinationRule

```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: order-service
  namespace: production
spec:
  host: order-service
  trafficPolicy:
    loadBalancer:
      simple: ROUND_ROBIN
    connectionPool:
      tcp:
        maxConnections: 100
        connectTimeout: 10s
      http:
        http1MaxPendingRequests: 100
        http2MaxRequests: 100
        maxRequestsPerConnection: 2
        maxRetries: 3
    outlierDetection:
      consecutiveErrors: 5
      interval: 30s
      baseEjectionTime: 30s
      maxEjectionPercent: 50
  subsets:
  - name: v1
    labels:
      version: v1.0.0
  - name: v2
    labels:
      version: v1.1.0
```

## Linkerd Configuration

### Generate Linkerd Manifests

```go
config := servicemesh.DefaultConfig()
config.ServiceName = "order-service"
config.MeshType = "linkerd"

manifests := servicemesh.GenerateLinkerdManifests(config)
```

### Linkerd Annotations

Add to Kubernetes deployment:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: order-service
  annotations:
    linkerd.io/inject: enabled
    config.linkerd.io/proxy-cpu-request: "100m"
    config.linkerd.io/proxy-memory-request: "128Mi"
spec:
  template:
    metadata:
      annotations:
        config.linkerd.io/skip-outbound-ports: "5432,3306"
```

### Example Linkerd TrafficSplit

```yaml
apiVersion: split.smi-spec.io/v1alpha2
kind: TrafficSplit
metadata:
  name: order-service
  namespace: production
spec:
  service: order-service
  backends:
  - service: order-service-v1
    weight: 90
  - service: order-service-v2
    weight: 10
```

### Example Linkerd ServiceProfile

```yaml
apiVersion: linkerd.io/v1alpha2
kind: ServiceProfile
metadata:
  name: order-service.production.svc.cluster.local
  namespace: production
spec:
  routes:
  - name: create_order
    condition:
      method: POST
      pathRegex: /api/orders
    isRetryable: true
    timeout: 10s
    retryBudget:
      retryRatio: 0.2
      minRetriesPerSecond: 10
      ttl: 10s
```

## Consul Connect Configuration

### Generate Consul Manifests

```go
config := servicemesh.DefaultConfig()
config.ServiceName = "order-service"
config.MeshType = "consul"

manifests := servicemesh.GenerateConsulManifests(config)
```

### Example Consul ServiceDefaults

```yaml
apiVersion: consul.hashicorp.com/v1alpha1
kind: ServiceDefaults
metadata:
  name: order-service
  namespace: production
spec:
  protocol: http
  meshGateway:
    mode: local
  upstreamConfig:
    defaults:
      connectTimeout: 10s
      limits:
        maxConnections: 100
        maxPendingRequests: 100
        maxConcurrentRequests: 100
```

### Example Consul ServiceSplitter

```yaml
apiVersion: consul.hashicorp.com/v1alpha1
kind: ServiceSplitter
metadata:
  name: order-service
  namespace: production
spec:
  splits:
  - weight: 90
    serviceSubset: v1
  - weight: 10
    serviceSubset: v2
```

## Traffic Patterns

### 1. Progressive Rollout

Gradually increase traffic to new version:

```
Deployment Timeline:
┌────────────────────────────────────────────┐
│ v1: 100%                                   │
├────────────────────────────────────────────┤
│ v1: 90%  | v2: 10%    (Week 1)            │
├────────────────────────────────────────────┤
│ v1: 75%  | v2: 25%    (Week 2)            │
├────────────────────────────────────────────┤
│ v1: 50%  | v2: 50%    (Week 3)            │
├────────────────────────────────────────────┤
│ v1: 25%  | v2: 75%    (Week 4)            │
├────────────────────────────────────────────┤
│ v2: 100%              (Week 5)            │
└────────────────────────────────────────────┘
```

### 2. User-Based Routing

Route based on user attributes:

```yaml
http:
- match:
  - headers:
      user-type:
        exact: beta-tester
  route:
  - destination:
      host: order-service
      subset: v2
- route:
  - destination:
      host: order-service
      subset: v1
```

### 3. Geographic Routing

Route based on user location:

```yaml
http:
- match:
  - headers:
      geo:
        prefix: us-
  route:
  - destination:
      host: order-service-us
- match:
  - headers:
      geo:
        prefix: eu-
  route:
  - destination:
      host: order-service-eu
```

## Load Balancing Algorithms

### Round Robin
Distributes requests evenly across all instances.

```go
config.TrafficPolicy.LoadBalancer.Type = "ROUND_ROBIN"
```

### Least Request
Routes to instance with fewest active requests.

```go
config.TrafficPolicy.LoadBalancer.Type = "LEAST_REQUEST"
```

### Consistent Hashing
Routes same user to same instance (session affinity).

```go
config.TrafficPolicy.LoadBalancer = LoadBalancerConfig{
    Type: "CONSISTENT_HASH",
    ConsistentHash: &ConsistentHashConfig{
        HTTPHeaderName: "user-id",
    },
}
```

## Circuit Breaking

Prevent cascading failures:

```go
config.TrafficPolicy.OutlierDetection = OutlierDetectionConfig{
    ConsecutiveErrors:  5,              // Eject after 5 errors
    Interval:           30 * time.Second, // Check every 30s
    BaseEjectionTime:   30 * time.Second, // Eject for 30s
    MaxEjectionPercent: 50,              // Max 50% instances ejected
}
```

**States:**
- **Closed**: Normal operation, requests flow through
- **Open**: Too many failures, fail fast
- **Half-Open**: Testing if service recovered

## Observability

### Metrics

Key metrics automatically collected:

```
# Request rate
istio_requests_total{destination_service="order-service"}

# Request duration (p50, p95, p99)
istio_request_duration_milliseconds{quantile="0.99"}

# Error rate
istio_requests_total{response_code="500"}

# Connection pool size
istio_tcp_connections_opened_total
```

### Tracing

View distributed traces in Jaeger:

```bash
# Forward Jaeger UI
kubectl port-forward -n istio-system svc/jaeger-query 16686:16686

# Access UI
open http://localhost:16686
```

### Service Graph

Visualize service dependencies:

```bash
# Kiali dashboard
kubectl port-forward -n istio-system svc/kiali 20001:20001

# Access UI
open http://localhost:20001
```

## Security Best Practices

### 1. Enable mTLS Everywhere

```go
config.EnableMTLS = true
```

### 2. Deny by Default

```yaml
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: deny-all
  namespace: production
spec:
  {}
```

### 3. Explicit Allow Lists

```yaml
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: order-service-policy
spec:
  selector:
    matchLabels:
      app: order-service
  rules:
  - from:
    - source:
        principals: ["cluster.local/ns/production/sa/api-gateway"]
    to:
    - operation:
        methods: ["GET", "POST"]
```

### 4. Rate Limiting

```yaml
apiVersion: networking.istio.io/v1beta1
kind: EnvoyFilter
metadata:
  name: rate-limit
spec:
  configPatches:
  - applyTo: HTTP_FILTER
    match:
      context: SIDECAR_INBOUND
    patch:
      operation: INSERT_BEFORE
      value:
        name: envoy.filters.http.local_ratelimit
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.http.local_ratelimit.v3.LocalRateLimit
          stat_prefix: http_local_rate_limiter
          token_bucket:
            max_tokens: 100
            tokens_per_fill: 100
            fill_interval: 60s
```

## Troubleshooting

### Check Sidecar Injection

```bash
# Verify sidecar is injected
kubectl get pod order-service-xxx -o jsonpath='{.spec.containers[*].name}'
# Should show: order-service istio-proxy

# Check sidecar logs
kubectl logs order-service-xxx -c istio-proxy
```

### Debug Traffic Issues

```bash
# Check virtual services
kubectl get virtualservices -n production

# Describe virtual service
kubectl describe virtualservice order-service -n production

# Check destination rules
kubectl get destinationrules -n production

# View Envoy config
istioctl proxy-config routes order-service-xxx
```

### mTLS Issues

```bash
# Check peer authentication
kubectl get peerauthentications -n production

# Verify certificates
istioctl proxy-config secret order-service-xxx

# Check mTLS status
istioctl authn tls-check order-service-xxx.production
```

## Comparison

| Feature | Istio | Linkerd | Consul |
|---------|-------|---------|--------|
| Complexity | High | Low | Medium |
| Performance | Good | Excellent | Good |
| Features | Extensive | Essential | Comprehensive |
| Learning Curve | Steep | Gentle | Moderate |
| Resource Usage | High | Low | Medium |
| mTLS | Yes | Yes | Yes |
| Traffic Splitting | Yes | Yes | Yes |
| Multi-cluster | Yes | Yes | Yes |

## When to Use Service Mesh

✅ **Use Service Mesh when:**
- Many microservices (10+)
- Need zero-trust security
- Complex traffic routing required
- Service-to-service encryption needed
- Want infrastructure-level observability

❌ **Don't Use Service Mesh when:**
- Simple monolith or few services
- Performance is critical (adds ~1ms latency)
- Team unfamiliar with Kubernetes
- Limited resources (CPU/memory)

## Resources

- [Istio Documentation](https://istio.io/latest/docs/)
- [Linkerd Documentation](https://linkerd.io/2/overview/)
- [Consul Connect Documentation](https://www.consul.io/docs/connect)
- [Service Mesh Comparison](https://servicemesh.es/)

## Summary

Service mesh provides:

✅ **Traffic Management** - Canary, A/B testing, circuit breaking
✅ **Security** - mTLS, authentication, authorization
✅ **Observability** - Tracing, metrics, service graph
✅ **Resilience** - Retries, timeouts, fault injection
✅ **Zero Code Changes** - Infrastructure-level features

Perfect for complex microservices architectures requiring advanced traffic management and security.
