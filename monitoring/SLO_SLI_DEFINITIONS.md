# SLO/SLI Definitions - Logistics Platform

## Overview

This document defines Service Level Indicators (SLIs), Service Level Objectives (SLOs), and error budgets for the logistics platform based on Google SRE best practices.

## Key Concepts

- **SLI (Service Level Indicator)**: A quantitative measure of service reliability
- **SLO (Service Level Objective)**: Target value or range for an SLI
- **SLA (Service Level Agreement)**: Contractual obligation (typically SLO - buffer)
- **Error Budget**: Allowed unreliability (100% - SLO)

## Platform-Wide SLOs

### Critical Services (Order, Shipment, Gateway)

| Metric | SLI | SLO | Error Budget | Measurement Window |
|--------|-----|-----|--------------|-------------------|
| **Availability** | Success rate of requests | 99.9% | 0.1% (43.2 min/month) | 30 days rolling |
| **Latency (P95)** | 95th percentile response time | < 1000ms | - | 5 minutes |
| **Latency (P99)** | 99th percentile response time | < 2000ms | - | 5 minutes |
| **Error Rate** | HTTP 5xx error rate | < 0.1% | 0.1% | 5 minutes |

### Standard Services (Inventory, Driver, Route, Notification)

| Metric | SLI | SLO | Error Budget | Measurement Window |
|--------|-----|-----|--------------|-------------------|
| **Availability** | Success rate of requests | 99.5% | 0.5% (3.6 hours/month) | 30 days rolling |
| **Latency (P95)** | 95th percentile response time | < 1500ms | - | 5 minutes |
| **Latency (P99)** | 99th percentile response time | < 3000ms | - | 5 minutes |
| **Error Rate** | HTTP 5xx error rate | < 0.5% | 0.5% | 5 minutes |

## Service-Specific SLIs

### Order Service

#### Availability SLI
```promql
sum(rate(http_requests_total{service="order-service", status!~"5.."}[5m]))
/
sum(rate(http_requests_total{service="order-service"}[5m]))
```
**Target**: 99.9% (43.2 minutes downtime per month)

#### Latency SLI (P95)
```promql
histogram_quantile(0.95,
  sum(rate(http_request_duration_seconds_bucket{service="order-service"}[5m])) by (le)
)
```
**Target**: < 1000ms

#### Error Rate SLI
```promql
sum(rate(http_requests_total{service="order-service", status=~"5.."}[5m]))
/
sum(rate(http_requests_total{service="order-service"}[5m]))
```
**Target**: < 0.1%

#### Order Processing Time
```promql
avg(order_processing_duration_seconds{service="order-service"})
```
**Target**: < 30 seconds

#### Order Creation Success Rate
```promql
sum(rate(orders_created_total{service="order-service"}[5m]))
/
sum(rate(order_creation_attempts_total{service="order-service"}[5m]))
```
**Target**: > 99.9%

### Shipment Service

#### Shipment Creation SLI
```promql
sum(rate(shipments_created_total{service="shipment-service"}[5m]))
/
sum(rate(shipment_creation_attempts_total{service="shipment-service"}[5m]))
```
**Target**: > 99.5%

#### Tracking Update Latency
```promql
histogram_quantile(0.95,
  sum(rate(shipment_tracking_update_duration_seconds_bucket[5m])) by (le)
)
```
**Target**: < 500ms

#### Shipment Status Accuracy
```promql
sum(shipment_status_corrections_total)
/
sum(shipment_status_updates_total)
```
**Target**: < 1% (less than 1% corrections needed)

### Inventory Service

#### Inventory Query Latency
```promql
histogram_quantile(0.95,
  sum(rate(inventory_query_duration_seconds_bucket[5m])) by (le)
)
```
**Target**: < 200ms

#### Stock Update Success Rate
```promql
sum(rate(inventory_updates_successful_total[5m]))
/
sum(rate(inventory_update_attempts_total[5m]))
```
**Target**: > 99.9%

#### Out-of-Stock Rate
```promql
count(inventory_quantity{quantity="0"})
/
count(inventory_quantity)
```
**Target**: < 5%

### Driver Service

#### Driver Assignment Time
```promql
avg(driver_assignment_duration_seconds)
```
**Target**: < 60 seconds

#### Driver Location Update Frequency
```promql
rate(driver_location_updates_total[1m])
```
**Target**: > 0.2/s (at least every 5 seconds per active driver)

#### Driver Utilization Rate
```promql
count(drivers{status="busy"})
/
count(drivers{status=~"busy|available"})
```
**Target**: 70-85% (optimal range)

### Route Service

#### Route Calculation Latency
```promql
histogram_quantile(0.95,
  sum(rate(route_calculation_duration_seconds_bucket[5m])) by (le)
)
```
**Target**: < 2000ms

#### Route Optimization Success Rate
```promql
sum(rate(routes_optimized_total[5m]))
/
sum(rate(route_optimization_attempts_total[5m]))
```
**Target**: > 99%

#### Route Efficiency
```promql
avg(route_actual_distance_km / route_optimal_distance_km)
```
**Target**: < 1.15 (within 15% of optimal)

### Notification Service

#### Notification Delivery Rate
```promql
sum(rate(notifications_delivered_total[5m]))
/
sum(rate(notifications_sent_total[5m]))
```
**Target**: > 99%

#### Notification Delivery Latency
```promql
histogram_quantile(0.95,
  sum(rate(notification_delivery_duration_seconds_bucket[5m])) by (le)
)
```
**Target**: < 5000ms

#### Notification Queue Depth
```promql
notification_queue_size
```
**Target**: < 10,000 messages

### Gateway Service

#### Gateway Availability
```promql
sum(rate(http_requests_total{service="gateway", status!~"5.."}[5m]))
/
sum(rate(http_requests_total{service="gateway"}[5m]))
```
**Target**: 99.95% (21.6 minutes downtime per month)

#### GraphQL Query Latency (P95)
```promql
histogram_quantile(0.95,
  sum(rate(graphql_query_duration_seconds_bucket[5m])) by (le)
)
```
**Target**: < 500ms

#### DataLoader Effectiveness
```promql
sum(rate(dataloader_cache_hits_total[5m]))
/
sum(rate(dataloader_requests_total[5m]))
```
**Target**: > 80% cache hit rate

## Infrastructure SLIs

### Database

#### Database Connection Pool Usage
```promql
sum(pg_stat_database_numbackends)
/
pg_settings_max_connections
```
**Target**: < 80%

#### Database Query Latency (P95)
```promql
histogram_quantile(0.95,
  sum(rate(pg_query_duration_seconds_bucket[5m])) by (le)
)
```
**Target**: < 100ms

#### Database Replication Lag
```promql
pg_replication_lag_seconds
```
**Target**: < 10 seconds

### Message Queue (RabbitMQ)

#### Message Processing Rate
```promql
rate(rabbitmq_queue_messages_published_total[5m])
```
**Target**: Match production rate

#### Queue Depth
```promql
rabbitmq_queue_messages
```
**Target**: < 5,000 messages per queue

#### Consumer Lag
```promql
rabbitmq_queue_messages
/
rate(rabbitmq_queue_messages_consumed_total[5m])
```
**Target**: < 60 seconds to drain at current rate

### Cache (Redis)

#### Cache Hit Rate
```promql
sum(rate(redis_keyspace_hits_total[5m]))
/
(sum(rate(redis_keyspace_hits_total[5m])) + sum(rate(redis_keyspace_misses_total[5m])))
```
**Target**: > 90%

#### Cache Memory Usage
```promql
redis_memory_used_bytes
/
redis_memory_max_bytes
```
**Target**: < 80%

#### Cache Eviction Rate
```promql
rate(redis_evicted_keys_total[5m])
```
**Target**: < 100 keys/second

## Error Budget Policy

### Budget Exhaustion Actions

#### Error Budget > 50% Remaining
- ✅ Full speed ahead on new features
- ✅ Normal deployment cadence
- ✅ Standard testing practices

#### Error Budget 25-50% Remaining
- ⚠️ Increase monitoring
- ⚠️ Reduce deployment frequency
- ⚠️ Enhanced pre-deployment testing

#### Error Budget 10-25% Remaining
- 🚨 Freeze new features
- 🚨 Focus on reliability improvements
- 🚨 Mandatory load testing before deployment
- 🚨 On-call escalation

#### Error Budget < 10% Remaining
- 🔴 Complete deployment freeze
- 🔴 Emergency reliability sprint
- 🔴 Executive escalation
- 🔴 Post-mortem required

### Budget Reset Schedule
- **Monthly**: Full error budget reset
- **Weekly**: Review and trend analysis
- **Daily**: Monitoring dashboard review

## Alerting Strategy

### Page-Worthy Alerts (Wake someone up)

1. **SLO violation**: Error budget burn rate > 5% per hour
2. **Service down**: No successful requests in 2 minutes
3. **Critical error rate**: > 10% HTTP 5xx errors
4. **Database down**: Primary database unavailable
5. **Data loss risk**: Replication lag > 5 minutes

### Ticket-Worthy Alerts (Create ticket, handle next day)

1. **Elevated error rate**: 1-5% HTTP 5xx errors for 15 minutes
2. **High latency**: P95 > 2x target for 10 minutes
3. **Resource pressure**: Memory/CPU > 85% for 15 minutes
4. **Queue backup**: Message queue depth > threshold for 30 minutes

### Info-Only Alerts (Dashboard, no action)

1. **Traffic spike**: 2x normal request rate
2. **Cache miss rate increase**: Hit rate < 80%
3. **Slow external API**: Third-party latency > normal

## Monitoring Dashboards

### Executive Dashboard (Business Metrics)
- Overall platform availability (30-day rolling)
- Current error budget status (all services)
- Total requests per second
- Orders processed today
- Active shipments

### Operations Dashboard (Technical Metrics)
- Service availability (per service)
- Latency percentiles (P50, P95, P99)
- Error rates (4xx, 5xx)
- Resource utilization (CPU, memory)
- Database performance

### On-Call Dashboard (Real-Time)
- Active alerts
- Recent deployments
- Traffic patterns (last 24h)
- Error budget burn rate
- Service dependencies health

## Reporting

### Daily
- Error budget consumption
- SLO compliance status
- Active incidents

### Weekly
- SLO achievement summary
- Reliability trends
- Deployment impact analysis
- On-call load

### Monthly
- Full SLO report card
- Error budget utilization
- Capacity planning metrics
- Improvement recommendations

## References

- [Google SRE Book - SLOs](https://sre.google/sre-book/service-level-objectives/)
- [Google SRE Workbook - Implementing SLOs](https://sre.google/workbook/implementing-slos/)
- [Implementing SLIs and SLOs](https://cloud.google.com/blog/products/management-tools/practical-guide-to-setting-slos)

---

**Created**: 2025-11-19
**Last Updated**: 2025-11-19
**Review Cycle**: Quarterly
**Owner**: Platform SRE Team
