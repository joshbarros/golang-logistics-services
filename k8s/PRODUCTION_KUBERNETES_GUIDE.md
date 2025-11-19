# Production-Ready Kubernetes Configuration Guide

## Overview

This guide documents the production-ready Kubernetes configuration pattern used in this platform. All services should follow these standards.

## Reference Implementation

**File**: `k8s/order-service.yaml`

This file serves as the reference implementation with all production best practices applied.

## Production Requirements Checklist

### ✅ Deployment Configuration

- [ ] **Resource Limits and Requests** (CRITICAL)
  ```yaml
  resources:
    requests:
      cpu: 100m      # Minimum guaranteed
      memory: 128Mi
    limits:
      cpu: 1000m     # Maximum allowed
      memory: 512Mi
  ```

- [ ] **Health Checks** (CRITICAL)
  - Startup probe: For slow-starting containers
  - Liveness probe: Restart if unhealthy
  - Readiness probe: Remove from load balancer if not ready

- [ ] **Graceful Shutdown** (CRITICAL)
  ```yaml
  terminationGracePeriodSeconds: 35
  lifecycle:
    preStop:
      exec:
        command: ["/bin/sh", "-c", "sleep 5"]
  ```

- [ ] **Rolling Update Strategy**
  ```yaml
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  ```

- [ ] **Security Context**
  ```yaml
  securityContext:
    runAsNonRoot: true
    runAsUser: 1000
    allowPrivilegeEscalation: false
    readOnlyRootFilesystem: true
    capabilities:
      drop:
        - ALL
  ```

- [ ] **Pod Anti-Affinity** (spread pods across nodes)

- [ ] **Prometheus Annotations**
  ```yaml
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "8080"
    prometheus.io/path: "/metrics"
  ```

### ✅ High Availability

- [ ] **PodDisruptionBudget** (CRITICAL)
  ```yaml
  apiVersion: policy/v1
  kind: PodDisruptionBudget
  metadata:
    name: service-name-pdb
  spec:
    minAvailable: 2  # Minimum pods during disruptions
  ```

- [ ] **HorizontalPodAutoscaler** (CRITICAL)
  ```yaml
  apiVersion: autoscaling/v2
  kind: HorizontalPodAutoscaler
  metadata:
    name: service-name-hpa
  spec:
    minReplicas: 3
    maxReplicas: 10
    metrics:
      - type: Resource
        resource:
          name: cpu
          target:
            type: Utilization
            averageUtilization: 70
  ```

- [ ] **Minimum 3 replicas** for production

### ✅ Security

- [ ] Secrets from Kubernetes Secrets (not environment variables)
- [ ] Non-root user
- [ ] Read-only root filesystem
- [ ] Drop all capabilities
- [ ] Seccomp profile

### ✅ Observability

- [ ] Prometheus metrics endpoint
- [ ] Structured logging
- [ ] Distributed tracing (Jaeger)
- [ ] Health check endpoints

## Services to Update

| Service | File | Status |
|---------|------|--------|
| Order | `k8s/order-service.yaml` | ✅ **UPDATED** |
| Shipment | `k8s/shipment-service.yaml` | ⚠️ TODO |
| Inventory | `k8s/inventory-service.yaml` | ⚠️ TODO |
| Driver | `k8s/driver-service.yaml` | ⚠️ TODO |
| Route | `k8s/route-service.yaml` | ⚠️ TODO |
| Notification | `k8s/notification-service.yaml` | ⚠️ TODO |
| Gateway | `k8s/api-gateway.yaml` | ⚠️ TODO |

## Resource Sizing Guidelines

### Small Service (< 100 req/s)
```yaml
resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 500m
    memory: 256Mi
```

### Medium Service (100-1000 req/s)
```yaml
resources:
  requests:
    cpu: 200m
    memory: 256Mi
  limits:
    cpu: 1000m
    memory: 512Mi
```

### Large Service (> 1000 req/s)
```yaml
resources:
  requests:
    cpu: 500m
    memory: 512Mi
  limits:
    cpu: 2000m
    memory: 1Gi
```

### Gateway Service
```yaml
resources:
  requests:
    cpu: 500m
    memory: 512Mi
  limits:
    cpu: 2000m
    memory: 1Gi
```

## Health Check Paths

All services must implement:

- `/health` - Overall health status
- `/live` - Liveness check (for liveness probe)
- `/ready` - Readiness check (for readiness probe)
- `/metrics` - Prometheus metrics

## Estimated Time

- **Per service**: 20-30 minutes
- **Total**: ~3 hours for all 6 remaining services
- **Priority**: HIGH (blocking production deployment)

## Testing

After updating each manifest:

```bash
# Validate YAML syntax
kubectl apply --dry-run=client -f k8s/service-name.yaml

# Validate against cluster
kubectl apply --dry-run=server -f k8s/service-name.yaml

# Deploy to test cluster
kubectl apply -f k8s/service-name.yaml

# Check rollout status
kubectl rollout status deployment/service-name

# Verify pods are healthy
kubectl get pods -l app=service-name

# Check resource usage
kubectl top pods -l app=service-name

# Test autoscaling
kubectl run -it --rm load-generator --image=busybox /bin/sh
# Then run: while true; do wget -q -O- http://service-name:8080; done
# Watch HPA: kubectl get hpa -w
```

## Common Issues

### Issue: OOMKilled (Out of Memory)
**Solution**: Increase memory limits

### Issue: CrashLoopBackOff
**Solution**: Check logs, adjust startup probe settings

### Issue: Pods not ready
**Solution**: Check readiness probe path and timing

### Issue: Pods evicted
**Solution**: Add resource requests, configure PodDisruptionBudget

## Next Steps

1. ⚠️ Update remaining 6 service manifests
2. ⚠️ Create production secrets
3. ⚠️ Configure cluster autoscaler
4. ⚠️ Set up monitoring and alerting
5. ⚠️ Create Helm charts for easier deployment

---

**Created**: 2025-11-19
**Last Updated**: 2025-11-19
**Reference**: `k8s/order-service.yaml`
