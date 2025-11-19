## Chaos Engineering

Chaos Engineering is the practice of intentionally injecting failures into systems to test their resilience and identify weaknesses before they cause outages in production.

## Features

- **Fault Injection**: Latency, errors, connection aborts, resource exhaustion
- **Experiment Management**: Create, start, stop, and monitor chaos experiments
- **Scenario Runner**: Execute multi-step chaos scenarios with validation
- **HTTP Middleware**: Easy integration with web services
- **Metrics Collection**: Track request rates, error rates, and latency during experiments
- **Validation Rules**: Define acceptance criteria for experiments

## Fault Types

### 1. Latency Injection
Adds delays to requests to simulate network issues or slow dependencies.

### 2. Error Injection
Returns HTTP error codes to simulate service failures.

### 3. Connection Abort
Abruptly terminates connections to simulate network issues.

### 4. Resource Exhaustion
Consumes CPU, memory, or disk I/O to simulate resource constraints.

### 5. Network Partition
Simulates network splits and packet loss.

## Usage

### Basic Chaos Injection

```go
import "golang-logistics-services/pkg/chaos"

// Create chaos manager
config := chaos.Config{
    Enabled:     true,
    Probability: 0.1, // 10% of requests
}
manager := chaos.NewManager(config)

// Add latency experiment
exp := chaos.NewLatencyExperiment(
    "order-service",
    "Test service degradation",
    5 * time.Minute,  // Run for 5 minutes
    2 * time.Second,  // Add 2 second delay
)
manager.AddExperiment(exp)
manager.StartExperiment(exp.ID)

// Inject chaos in your code
if err := manager.InjectFault(ctx, "order-service"); err != nil {
    log.Printf("Chaos injected: %v", err)
    return err
}
```

### HTTP Middleware

```go
// Add chaos middleware to HTTP handler
http.Handle("/api/orders",
    manager.HTTPMiddleware("order-service")(ordersHandler),
)

// Now 10% of requests will experience chaos
```

### Running Scenarios

```go
// Create runner
runner := chaos.NewRunner(manager)

// Add service degradation scenario
scenario := chaos.NewServiceDegradationScenario("order-service")
runner.AddScenario(scenario)

// Run scenario
result, err := runner.RunScenario(context.Background(), scenario.ID)
if err != nil {
    log.Fatal(err)
}

// Check results
fmt.Printf("Scenario: %s\n", result.ScenarioName)
fmt.Printf("Duration: %v\n", result.Duration)
fmt.Printf("Error Rate: %.2f%%\n", result.Metrics.ErrorRate*100)
fmt.Printf("Passed: %v\n", result.Passed)

for _, step := range result.Steps {
    fmt.Printf("  Step: %s - Duration: %v - Success: %v\n",
        step.StepName, step.Duration, step.Success)
}
```

## Predefined Scenarios

### 1. Service Degradation Test
Tests system behavior under increasing latency.

**Steps:**
1. Baseline (no latency)
2. Mild latency (100ms)
3. Moderate latency (500ms)
4. High latency (2s)

**Validation:**
- Max 5% error rate
- Max 3s latency
- Min 95% success rate

```go
scenario := chaos.NewServiceDegradationScenario("order-service")
```

### 2. Error Handling Test
Tests error handling under various failure modes.

**Steps:**
1. 5% error rate
2. 10% error rate
3. HTTP 503 errors

**Validation:**
- Max 15% error rate
- Min 85% success rate

```go
scenario := chaos.NewErrorHandlingScenario("inventory-service")
```

### 3. Failover Test
Tests failover from primary to backup service.

**Steps:**
1. Kill primary service (5 minutes)

**Validation:**
- Max 10% error rate (during failover)
- Min 90% success rate
- Custom: Verify traffic shifted to backup

```go
scenario := chaos.NewFailoverScenario("primary-db", "backup-db")
```

## Custom Experiments

### Latency Experiment

```go
exp := chaos.Experiment{
    ID:          "exp-001",
    Name:        "Network Degradation",
    Description: "Simulate slow network",
    Target: chaos.Target{
        Type: "service",
        Name: "shipment-service",
    },
    Fault: chaos.Fault{
        Type: "latency",
        Latency: &chaos.LatencyFault{
            Duration: 3 * time.Second,
            Jitter:   20, // 20% randomness
        },
    },
    Schedule: chaos.Schedule{
        StartTime: time.Now(),
        Duration:  10 * time.Minute,
    },
}

manager.AddExperiment(exp)
manager.StartExperiment(exp.ID)
```

### Error Experiment

```go
exp := chaos.Experiment{
    Name: "Database Outage",
    Target: chaos.Target{
        Type: "database",
        Name: "postgres",
    },
    Fault: chaos.Fault{
        Type: "error",
        Error: &chaos.ErrorFault{
            StatusCode: 503,
            Message:    "Database connection failed",
        },
    },
    Schedule: chaos.Schedule{
        StartTime: time.Now(),
        Duration:  5 * time.Minute,
    },
}
```

### Resource Exhaustion

```go
exp := chaos.Experiment{
    Name: "CPU Spike",
    Target: chaos.Target{
        Type: "service",
        Name: "analytics-service",
    },
    Fault: chaos.Fault{
        Type: "resource",
        Resource: &chaos.ResourceFault{
            CPUPercent:    80, // Consume 80% CPU
            MemoryMB:      512, // Consume 512MB RAM
            DiskIOPercent: 50,  // 50% disk I/O
        },
    },
    Schedule: chaos.Schedule{
        Duration: 5 * time.Minute,
    },
}
```

## Custom Scenarios

```go
scenario := chaos.Scenario{
    Name:        "Peak Load Failure",
    Description: "Simulate failure during peak load",
    Steps: []chaos.ScenarioStep{
        {
            Name: "Normal operation",
            Experiment: chaos.NewLatencyExperiment(
                "order-service", "Baseline", 2*time.Minute, 0,
            ),
            Duration: 2 * time.Minute,
        },
        {
            Name: "Peak load with degradation",
            Experiment: chaos.NewLatencyExperiment(
                "order-service", "Peak load", 5*time.Minute, 1*time.Second,
            ),
            Duration:   5 * time.Minute,
            WaitBefore: 30 * time.Second,
        },
        {
            Name: "Partial outage",
            Experiment: chaos.NewErrorExperiment(
                "order-service", "Partial outage", 3*time.Minute, 503, "Overloaded",
            ),
            Duration:   3 * time.Minute,
            WaitBefore: 30 * time.Second,
        },
    },
    Validation: chaos.ValidationRules{
        MaxErrorRate:   0.20, // 20% error rate acceptable
        MaxLatency:     5 * time.Second,
        MinSuccessRate: 0.80, // 80% success rate required
    },
    Timeout: 20 * time.Minute,
}

runner.AddScenario(scenario)
result, _ := runner.RunScenario(ctx, scenario.ID)
```

## Integration with Service Mesh

Chaos engineering integrates with service mesh fault injection:

```go
import (
    "golang-logistics-services/pkg/chaos"
    "golang-logistics-services/pkg/servicemesh"
)

// Use service mesh for infrastructure-level chaos
meshManager, _ := servicemesh.NewManager(servicemesh.DefaultConfig())

// Inject latency via service mesh
meshManager.InjectFault(
    ctx,
    "order-service",
    &servicemesh.Delay{
        Percentage: 10.0,
        FixedDelay: 5 * time.Second,
    },
    nil,
)

// Inject errors via service mesh
meshManager.InjectFault(
    ctx,
    "order-service",
    nil,
    &servicemesh.Abort{
        Percentage: 5.0,
        HTTPStatus: 503,
    },
)
```

## Best Practices

### 1. Start Small
Begin with low percentages and short durations:

```go
config := chaos.Config{
    Enabled:     true,
    Probability: 0.01, // 1% of requests
}
```

### 2. Gradual Increase
Progressively increase chaos intensity:

```
Week 1: 1% probability, 100ms latency
Week 2: 5% probability, 500ms latency
Week 3: 10% probability, 1s latency
Week 4: 10% probability, mixed faults
```

### 3. Run During Business Hours
Run experiments when team is available to respond:

```go
schedule := chaos.Schedule{
    DaysOfWeek: []int{1, 2, 3, 4, 5}, // Monday-Friday
    TimeOfDay:  "09:00-17:00",        // Business hours
}
```

### 4. Monitor Closely
Watch metrics during experiments:

```go
runner.OnComplete(func(result *chaos.ScenarioResult) {
    // Send to Slack
    slack.SendMessage(fmt.Sprintf(
        "Chaos experiment %s completed. Error rate: %.2f%%, Passed: %v",
        result.ScenarioName,
        result.Metrics.ErrorRate*100,
        result.Passed,
    ))

    // Send to metrics
    metrics.RecordChaosExperiment(result.ScenarioName, result.Passed)
})
```

### 5. Define Clear Success Criteria
Set validation rules upfront:

```go
validation := chaos.ValidationRules{
    MaxErrorRate:   0.05,
    MaxLatency:     2 * time.Second,
    MinSuccessRate: 0.95,
    CustomValidation: func(result *chaos.ScenarioResult) error {
        // Check that orders weren't lost
        if ordersLost > 0 {
            return fmt.Errorf("orders were lost during experiment")
        }
        return nil
    },
}
```

### 6. Automate Experiments
Run chaos as part of CI/CD:

```bash
# In CI pipeline
go test -tags=chaos ./tests/chaos/...
```

### 7. Document Learnings
Track what you learn from each experiment:

```go
runner.OnComplete(func(result *chaos.ScenarioResult) {
    // Log to incident management system
    incident.Create(&Incident{
        Type:        "Chaos Experiment",
        Description: result.ScenarioName,
        Learnings:   extractLearnings(result),
        Actions:     determineActions(result),
    })
})
```

## Safety Mechanisms

### Automatic Rollback
```go
// Stop experiment if error rate exceeds threshold
if result.Metrics.ErrorRate > 0.50 {
    manager.StopExperiment(exp.ID)
    alert.Send("Chaos experiment stopped due to high error rate")
}
```

### Circuit Breaker Integration
```go
// Chaos respects circuit breaker state
if circuitBreaker.State() == "open" {
    // Don't inject additional chaos
    return nil
}
```

### Rate Limiting
```go
// Limit chaos injection rate
config := chaos.Config{
    Enabled:     true,
    Probability: 0.10, // Max 10% of requests
}
```

## Metrics & Observability

### Collect Metrics
```go
type ExperimentMetrics struct {
    TotalRequests   int64
    FailedRequests  int64
    AverageLatency  time.Duration
    ErrorRate       float64
}
```

### Export to Prometheus
```
# Chaos experiment status
chaos_experiment_active{experiment="latency-test"} 1

# Chaos injection rate
chaos_injections_total{type="latency"} 1234

# Experiment error rate
chaos_experiment_error_rate{experiment="error-test"} 0.05
```

### Dashboard
Create Grafana dashboard showing:
- Active experiments
- Injection rate
- Error rate during experiments
- Latency distribution
- Pass/fail rates

## Example Scenarios

### Black Friday Load Test
```go
scenario := chaos.Scenario{
    Name: "Black Friday Simulation",
    Steps: []chaos.ScenarioStep{
        {
            Name: "Normal load",
            Experiment: chaos.NewLatencyExperiment(
                "order-service", "Baseline", 5*time.Minute, 0,
            ),
            Duration: 5 * time.Minute,
        },
        {
            Name: "10x load with degradation",
            Experiment: chaos.NewLatencyExperiment(
                "order-service", "Peak load", 10*time.Minute, 500*time.Millisecond,
            ),
            Duration: 10 * time.Minute,
        },
        {
            Name: "Database slowdown",
            Experiment: chaos.NewLatencyExperiment(
                "postgres", "DB slow", 5*time.Minute, 2*time.Second,
            ),
            Duration: 5 * time.Minute,
        },
    },
    Validation: chaos.ValidationRules{
        MaxErrorRate:   0.10,
        MinSuccessRate: 0.90,
    },
}
```

### Database Failover
```go
scenario := chaos.Scenario{
    Name: "Database Failover Test",
    Steps: []chaos.ScenarioStep{
        {
            Name: "Kill primary database",
            Experiment: chaos.NewErrorExperiment(
                "postgres-primary", "Primary down", 5*time.Minute, 503, "DB unavailable",
            ),
            Duration: 5 * time.Minute,
        },
    },
    Validation: chaos.ValidationRules{
        MaxErrorRate: 0.05, // Should failover quickly
        CustomValidation: func(result *chaos.ScenarioResult) error {
            // Verify failover happened
            if !verifyFailoverToReplica() {
                return errors.New("failover did not occur")
            }
            return nil
        },
    },
}
```

### Cascading Failure
```go
scenario := chaos.Scenario{
    Name: "Cascading Failure Test",
    Steps: []chaos.ScenarioStep{
        {
            Name: "Slow inventory service",
            Experiment: chaos.NewLatencyExperiment(
                "inventory-service", "Slow inventory", 3*time.Minute, 5*time.Second,
            ),
            Duration: 3 * time.Minute,
        },
        {
            Name: "Order service degradation (should be protected)",
            Experiment: chaos.NewLatencyExperiment(
                "order-service", "Order degradation", 3*time.Minute, 100*time.Millisecond,
            ),
            Duration:   3 * time.Minute,
            WaitBefore: 30 * time.Second,
        },
    },
    Validation: chaos.ValidationRules{
        MaxErrorRate:   0.15,
        MinSuccessRate: 0.85,
        CustomValidation: func(result *chaos.ScenarioResult) error {
            // Verify circuit breaker opened for inventory
            if !circuitBreakerOpened("inventory-service") {
                return errors.New("circuit breaker did not open")
            }
            return nil
        },
    },
}
```

## Troubleshooting

### Experiments Not Running
```bash
# Check if chaos is enabled
config.Enabled = true

# Check probability
config.Probability > 0

# Verify target matches
experiment.Target.Name == "your-service"
```

### High Error Rates
```bash
# Reduce probability
config.Probability = 0.05 // 5%

# Reduce fault severity
fault.Latency.Duration = 100 * time.Millisecond

# Stop experiment
manager.StopExperiment(exp.ID)
```

### Metrics Not Collected
```bash
# Ensure experiments are tracked
manager.StartExperiment(exp.ID)

# Check experiment is active
exp, _ := manager.GetExperiment(exp.ID)
fmt.Println(exp.Active)
```

## Resources

- [Principles of Chaos Engineering](https://principlesofchaos.org/)
- [Chaos Monkey](https://netflix.github.io/chaosmonkey/)
- [Gremlin Chaos Engineering](https://www.gremlin.com/)
- [AWS Fault Injection Simulator](https://aws.amazon.com/fis/)

## Summary

Chaos Engineering provides:

✅ **Proactive Testing** - Find weaknesses before production incidents
✅ **Resilience Validation** - Verify circuit breakers, retries, timeouts work
✅ **Confidence** - Deploy with confidence knowing system handles failures
✅ **Learning** - Understand system behavior under stress
✅ **Automation** - Run experiments as part of CI/CD pipeline

Perfect for validating that your distributed system can handle real-world failures gracefully.
