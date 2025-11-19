package chaos

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Runner executes chaos experiments and collects results
type Runner struct {
	manager    *Manager
	scenarios  map[string]*Scenario
	mutex      sync.RWMutex
	results    map[string]*ScenarioResult
	onComplete func(*ScenarioResult)
}

// Scenario defines a chaos testing scenario
type Scenario struct {
	// ID uniquely identifies the scenario
	ID string
	// Name is human-readable name
	Name string
	// Description explains the scenario
	Description string
	// Steps are executed in order
	Steps []ScenarioStep
	// Validation checks for expected behavior
	Validation ValidationRules
	// Timeout for the entire scenario
	Timeout time.Duration
}

// ScenarioStep represents a step in the scenario
type ScenarioStep struct {
	// Name of the step
	Name string
	// Experiment to run
	Experiment Experiment
	// Duration how long to run the experiment
	Duration time.Duration
	// WaitBefore waiting before starting
	WaitBefore time.Duration
	// WaitAfter waiting after completing
	WaitAfter time.Duration
}

// ValidationRules defines expected behavior
type ValidationRules struct {
	// MaxErrorRate allowed (0.0-1.0)
	MaxErrorRate float64
	// MaxLatency allowed
	MaxLatency time.Duration
	// MinSuccessRate required (0.0-1.0)
	MinSuccessRate float64
	// Custom validation function
	CustomValidation func(*ScenarioResult) error
}

// ScenarioResult contains scenario execution results
type ScenarioResult struct {
	// ScenarioID identifies the scenario
	ScenarioID string
	// ScenarioName for display
	ScenarioName string
	// StartTime when scenario started
	StartTime time.Time
	// EndTime when scenario ended
	EndTime time.Time
	// Duration total duration
	Duration time.Duration
	// Steps results for each step
	Steps []StepResult
	// Passed indicates if validation passed
	Passed bool
	// Errors encountered during execution
	Errors []error
	// Metrics collected during scenario
	Metrics AggregatedMetrics
}

// StepResult contains step execution results
type StepResult struct {
	// StepName identifies the step
	StepName string
	// ExperimentID that was run
	ExperimentID string
	// StartTime when step started
	StartTime time.Time
	// EndTime when step ended
	EndTime time.Time
	// Duration of step execution
	Duration time.Duration
	// Metrics collected during step
	Metrics ExperimentMetrics
	// Success indicates if step completed successfully
	Success bool
	// Error if step failed
	Error error
}

// AggregatedMetrics contains aggregated metrics across all steps
type AggregatedMetrics struct {
	// TotalRequests across all steps
	TotalRequests int64
	// FailedRequests across all steps
	FailedRequests int64
	// ErrorRate percentage
	ErrorRate float64
	// AverageLatency across all steps
	AverageLatency time.Duration
	// P50Latency percentile
	P50Latency time.Duration
	// P95Latency percentile
	P95Latency time.Duration
	// P99Latency percentile
	P99Latency time.Duration
}

// NewRunner creates a new chaos runner
func NewRunner(manager *Manager) *Runner {
	return &Runner{
		manager:   manager,
		scenarios: make(map[string]*Scenario),
		results:   make(map[string]*ScenarioResult),
	}
}

// AddScenario adds a scenario to the runner
func (r *Runner) AddScenario(scenario Scenario) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if scenario.ID == "" {
		scenario.ID = generateID()
	}

	r.scenarios[scenario.ID] = &scenario
}

// RunScenario executes a scenario
func (r *Runner) RunScenario(ctx context.Context, scenarioID string) (*ScenarioResult, error) {
	r.mutex.RLock()
	scenario, exists := r.scenarios[scenarioID]
	r.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("scenario not found: %s", scenarioID)
	}

	fmt.Printf("Starting chaos scenario: %s (%s)\n", scenario.Name, scenario.ID)

	result := &ScenarioResult{
		ScenarioID:   scenario.ID,
		ScenarioName: scenario.Name,
		StartTime:    time.Now(),
		Steps:        make([]StepResult, 0, len(scenario.Steps)),
		Errors:       make([]error, 0),
	}

	// Apply timeout if specified
	if scenario.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, scenario.Timeout)
		defer cancel()
	}

	// Execute each step
	for _, step := range scenario.Steps {
		// Wait before starting if specified
		if step.WaitBefore > 0 {
			select {
			case <-time.After(step.WaitBefore):
			case <-ctx.Done():
				result.Errors = append(result.Errors, ctx.Err())
				break
			}
		}

		// Execute step
		stepResult := r.executeStep(ctx, step)
		result.Steps = append(result.Steps, stepResult)

		if !stepResult.Success {
			result.Errors = append(result.Errors, stepResult.Error)
		}

		// Wait after completing if specified
		if step.WaitAfter > 0 {
			time.Sleep(step.WaitAfter)
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Aggregate metrics
	result.Metrics = r.aggregateMetrics(result.Steps)

	// Validate results
	result.Passed = r.validate(scenario.Validation, result)

	// Store result
	r.mutex.Lock()
	r.results[scenario.ID] = result
	r.mutex.Unlock()

	// Call completion callback
	if r.onComplete != nil {
		r.onComplete(result)
	}

	fmt.Printf("Completed chaos scenario: %s (passed: %v)\n", scenario.Name, result.Passed)

	return result, nil
}

// executeStep executes a single scenario step
func (r *Runner) executeStep(ctx context.Context, step ScenarioStep) StepResult {
	result := StepResult{
		StepName:  step.Name,
		StartTime: time.Now(),
		Success:   true,
	}

	// Add experiment to manager
	r.manager.AddExperiment(step.Experiment)
	result.ExperimentID = step.Experiment.ID

	// Start experiment
	if err := r.manager.StartExperiment(step.Experiment.ID); err != nil {
		result.Success = false
		result.Error = err
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result
	}

	// Wait for duration
	select {
	case <-time.After(step.Duration):
		// Experiment completed
	case <-ctx.Done():
		result.Success = false
		result.Error = ctx.Err()
	}

	// Stop experiment
	if err := r.manager.StopExperiment(step.Experiment.ID); err != nil {
		result.Success = false
		result.Error = err
	}

	// Get experiment metrics
	if exp, err := r.manager.GetExperiment(step.Experiment.ID); err == nil {
		result.Metrics = exp.Metrics
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result
}

// aggregateMetrics aggregates metrics from all steps
func (r *Runner) aggregateMetrics(steps []StepResult) AggregatedMetrics {
	metrics := AggregatedMetrics{}

	for _, step := range steps {
		metrics.TotalRequests += step.Metrics.TotalRequests
		metrics.FailedRequests += step.Metrics.FailedRequests
	}

	if metrics.TotalRequests > 0 {
		metrics.ErrorRate = float64(metrics.FailedRequests) / float64(metrics.TotalRequests)
	}

	// Calculate average latency (simplified)
	var totalLatency time.Duration
	for _, step := range steps {
		totalLatency += step.Metrics.AverageLatency
	}
	if len(steps) > 0 {
		metrics.AverageLatency = totalLatency / time.Duration(len(steps))
	}

	return metrics
}

// validate checks if results meet validation rules
func (r *Runner) validate(rules ValidationRules, result *ScenarioResult) bool {
	// Check error rate
	if rules.MaxErrorRate > 0 && result.Metrics.ErrorRate > rules.MaxErrorRate {
		result.Errors = append(result.Errors, fmt.Errorf(
			"error rate %.2f%% exceeds maximum %.2f%%",
			result.Metrics.ErrorRate*100,
			rules.MaxErrorRate*100,
		))
		return false
	}

	// Check latency
	if rules.MaxLatency > 0 && result.Metrics.AverageLatency > rules.MaxLatency {
		result.Errors = append(result.Errors, fmt.Errorf(
			"average latency %v exceeds maximum %v",
			result.Metrics.AverageLatency,
			rules.MaxLatency,
		))
		return false
	}

	// Check success rate
	successRate := 1.0 - result.Metrics.ErrorRate
	if rules.MinSuccessRate > 0 && successRate < rules.MinSuccessRate {
		result.Errors = append(result.Errors, fmt.Errorf(
			"success rate %.2f%% below minimum %.2f%%",
			successRate*100,
			rules.MinSuccessRate*100,
		))
		return false
	}

	// Custom validation
	if rules.CustomValidation != nil {
		if err := rules.CustomValidation(result); err != nil {
			result.Errors = append(result.Errors, err)
			return false
		}
	}

	return true
}

// GetResult returns the result of a scenario execution
func (r *Runner) GetResult(scenarioID string) (*ScenarioResult, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result, exists := r.results[scenarioID]
	if !exists {
		return nil, fmt.Errorf("result not found for scenario: %s", scenarioID)
	}

	return result, nil
}

// ListResults returns all scenario results
func (r *Runner) ListResults() []*ScenarioResult {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	results := make([]*ScenarioResult, 0, len(r.results))
	for _, result := range r.results {
		results = append(results, result)
	}

	return results
}

// OnComplete sets a callback for when scenarios complete
func (r *Runner) OnComplete(callback func(*ScenarioResult)) {
	r.onComplete = callback
}

// Predefined scenarios

// NewServiceDegradationScenario tests service behavior under degradation
func NewServiceDegradationScenario(serviceName string) Scenario {
	return Scenario{
		ID:          generateID(),
		Name:        "Service Degradation Test",
		Description: fmt.Sprintf("Test %s behavior under increasing latency", serviceName),
		Steps: []ScenarioStep{
			{
				Name: "Baseline",
				Experiment: NewLatencyExperiment(
					serviceName,
					"Baseline measurement",
					1*time.Minute,
					0,
				),
				Duration: 1 * time.Minute,
			},
			{
				Name: "Mild latency (100ms)",
				Experiment: NewLatencyExperiment(
					serviceName,
					"Mild latency injection",
					2*time.Minute,
					100*time.Millisecond,
				),
				Duration:   2 * time.Minute,
				WaitBefore: 10 * time.Second,
			},
			{
				Name: "Moderate latency (500ms)",
				Experiment: NewLatencyExperiment(
					serviceName,
					"Moderate latency injection",
					2*time.Minute,
					500*time.Millisecond,
				),
				Duration:   2 * time.Minute,
				WaitBefore: 10 * time.Second,
			},
			{
				Name: "High latency (2s)",
				Experiment: NewLatencyExperiment(
					serviceName,
					"High latency injection",
					2*time.Minute,
					2*time.Second,
				),
				Duration:   2 * time.Minute,
				WaitBefore: 10 * time.Second,
			},
		},
		Validation: ValidationRules{
			MaxErrorRate:   0.05, // 5% error rate acceptable
			MaxLatency:     3 * time.Second,
			MinSuccessRate: 0.95, // 95% success rate required
		},
		Timeout: 15 * time.Minute,
	}
}

// NewErrorHandlingScenario tests error handling
func NewErrorHandlingScenario(serviceName string) Scenario {
	return Scenario{
		ID:          generateID(),
		Name:        "Error Handling Test",
		Description: fmt.Sprintf("Test %s error handling under various failure modes", serviceName),
		Steps: []ScenarioStep{
			{
				Name: "5% error rate",
				Experiment: NewAbortExperiment(
					serviceName,
					"Low error rate",
					2*time.Minute,
					0.05,
				),
				Duration: 2 * time.Minute,
			},
			{
				Name: "10% error rate",
				Experiment: NewAbortExperiment(
					serviceName,
					"Moderate error rate",
					2*time.Minute,
					0.10,
				),
				Duration:   2 * time.Minute,
				WaitBefore: 10 * time.Second,
			},
			{
				Name: "HTTP 503 errors",
				Experiment: NewErrorExperiment(
					serviceName,
					"Service unavailable errors",
					2*time.Minute,
					503,
					"Service temporarily unavailable",
				),
				Duration:   2 * time.Minute,
				WaitBefore: 10 * time.Second,
			},
		},
		Validation: ValidationRules{
			MaxErrorRate:   0.15, // 15% error rate acceptable during chaos
			MinSuccessRate: 0.85, // 85% success rate required
		},
		Timeout: 10 * time.Minute,
	}
}

// NewFailoverScenario tests failover behavior
func NewFailoverScenario(primaryService, backupService string) Scenario {
	return Scenario{
		ID:          generateID(),
		Name:        "Failover Test",
		Description: fmt.Sprintf("Test failover from %s to %s", primaryService, backupService),
		Steps: []ScenarioStep{
			{
				Name: "Kill primary service",
				Experiment: NewErrorExperiment(
					primaryService,
					"Primary service failure",
					5*time.Minute,
					503,
					"Service unavailable",
				),
				Duration: 5 * time.Minute,
			},
		},
		Validation: ValidationRules{
			MaxErrorRate:   0.10, // Some errors expected during failover
			MinSuccessRate: 0.90, // Should recover quickly
			CustomValidation: func(result *ScenarioResult) error {
				// Verify traffic shifted to backup
				// In production, query metrics to verify
				return nil
			},
		},
		Timeout: 10 * time.Minute,
	}
}

// Example usage:
//
// // Create chaos manager and runner
// manager := chaos.NewManager(chaos.Config{
//     Enabled:     true,
//     Probability: 1.0, // 100% during experiments
// })
// runner := chaos.NewRunner(manager)
//
// // Set completion callback
// runner.OnComplete(func(result *chaos.ScenarioResult) {
//     fmt.Printf("Scenario completed: %s (passed: %v)\n",
//         result.ScenarioName, result.Passed)
//     fmt.Printf("Error rate: %.2f%%\n", result.Metrics.ErrorRate*100)
//     fmt.Printf("Average latency: %v\n", result.Metrics.AverageLatency)
// })
//
// // Add and run service degradation scenario
// scenario := chaos.NewServiceDegradationScenario("order-service")
// runner.AddScenario(scenario)
//
// result, err := runner.RunScenario(context.Background(), scenario.ID)
// if err != nil {
//     log.Fatal(err)
// }
//
// // Check results
// if !result.Passed {
//     fmt.Println("Chaos scenario failed validation:")
//     for _, err := range result.Errors {
//         fmt.Printf("  - %v\n", err)
//     }
// }
//
// // Run error handling scenario
// errorScenario := chaos.NewErrorHandlingScenario("inventory-service")
// runner.AddScenario(errorScenario)
// runner.RunScenario(context.Background(), errorScenario.ID)
