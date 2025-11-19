package chaos

import (
	"context"
	"testing"
	"time"
)

func TestChaosManager(t *testing.T) {
	// Create manager
	config := Config{
		Enabled:     true,
		Probability: 1.0, // 100% for testing
	}
	manager := NewManager(config)

	if manager == nil {
		t.Fatal("Failed to create chaos manager")
	}

	// Test adding experiment
	exp := NewLatencyExperiment(
		"test-service",
		"Test Latency",
		1*time.Minute,
		100*time.Millisecond,
	)

	manager.AddExperiment(exp)

	// Test starting experiment
	if err := manager.StartExperiment(exp.ID); err != nil {
		t.Fatalf("Failed to start experiment: %v", err)
	}

	// Test should inject
	if !manager.ShouldInject("test-service") {
		t.Error("Expected chaos to be injected")
	}

	// Test inject fault
	ctx := context.Background()
	start := time.Now()
	err := manager.InjectFault(ctx, "test-service")
	duration := time.Since(start)

	if err != nil {
		t.Logf("Fault injected: %v", err)
	}

	// Should have added latency
	if duration < 50*time.Millisecond {
		t.Errorf("Expected latency >= 50ms, got %v", duration)
	}

	// Test stopping experiment
	if err := manager.StopExperiment(exp.ID); err != nil {
		t.Fatalf("Failed to stop experiment: %v", err)
	}

	// Test should not inject after stop
	if manager.ShouldInject("test-service") {
		t.Error("Expected chaos not to be injected after stop")
	}
}

func TestNewLatencyExperiment(t *testing.T) {
	exp := NewLatencyExperiment(
		"order-service",
		"Test Latency Injection",
		5*time.Minute,
		2*time.Second,
	)

	if exp.Target.Name != "order-service" {
		t.Errorf("Expected target name 'order-service', got '%s'", exp.Target.Name)
	}

	if exp.Fault.Type != "latency" {
		t.Errorf("Expected fault type 'latency', got '%s'", exp.Fault.Type)
	}

	if exp.Fault.Latency.Duration != 2*time.Second {
		t.Errorf("Expected latency 2s, got %v", exp.Fault.Latency.Duration)
	}
}

func TestNewErrorExperiment(t *testing.T) {
	exp := NewErrorExperiment(
		"inventory-service",
		"Test Error Injection",
		5*time.Minute,
		503,
		"Service unavailable",
	)

	if exp.Fault.Type != "error" {
		t.Errorf("Expected fault type 'error', got '%s'", exp.Fault.Type)
	}

	if exp.Fault.Error.StatusCode != 503 {
		t.Errorf("Expected status code 503, got %d", exp.Fault.Error.StatusCode)
	}
}

func TestChaosRunner(t *testing.T) {
	manager := NewManager(Config{
		Enabled:     true,
		Probability: 1.0,
	})
	runner := NewRunner(manager)

	// Create simple scenario
	scenario := Scenario{
		ID:          "test-scenario",
		Name:        "Test Scenario",
		Description: "Simple test scenario",
		Steps: []ScenarioStep{
			{
				Name: "Step 1",
				Experiment: NewLatencyExperiment(
					"test-service",
					"Step 1 experiment",
					2*time.Second,
					50*time.Millisecond,
				),
				Duration: 2 * time.Second,
			},
		},
		Validation: ValidationRules{
			MaxErrorRate:   0.1,
			MinSuccessRate: 0.9,
		},
		Timeout: 30 * time.Second,
	}

	runner.AddScenario(scenario)

	// Run scenario
	ctx := context.Background()
	result, err := runner.RunScenario(ctx, scenario.ID)

	if err != nil {
		t.Fatalf("Failed to run scenario: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}

	if result.ScenarioID != scenario.ID {
		t.Errorf("Expected scenario ID '%s', got '%s'", scenario.ID, result.ScenarioID)
	}

	if len(result.Steps) != 1 {
		t.Errorf("Expected 1 step result, got %d", len(result.Steps))
	}

	t.Logf("Scenario completed in %v", result.Duration)
	t.Logf("Steps: %d, Passed: %v", len(result.Steps), result.Passed)
}

func TestServiceDegradationScenario(t *testing.T) {
	scenario := NewServiceDegradationScenario("order-service")

	if scenario.Name != "Service Degradation Test" {
		t.Errorf("Unexpected scenario name: %s", scenario.Name)
	}

	if len(scenario.Steps) != 4 {
		t.Errorf("Expected 4 steps, got %d", len(scenario.Steps))
	}

	// Verify step progression
	expectedLatencies := []time.Duration{0, 100 * time.Millisecond, 500 * time.Millisecond, 2 * time.Second}
	for i, step := range scenario.Steps {
		if step.Experiment.Fault.Latency.Duration != expectedLatencies[i] {
			t.Errorf("Step %d: expected latency %v, got %v",
				i, expectedLatencies[i], step.Experiment.Fault.Latency.Duration)
		}
	}
}

func TestErrorHandlingScenario(t *testing.T) {
	scenario := NewErrorHandlingScenario("shipment-service")

	if scenario.Name != "Error Handling Test" {
		t.Errorf("Unexpected scenario name: %s", scenario.Name)
	}

	if len(scenario.Steps) != 3 {
		t.Errorf("Expected 3 steps, got %d", len(scenario.Steps)  )
	}
}

func TestFailoverScenario(t *testing.T) {
	scenario := NewFailoverScenario("primary-db", "backup-db")

	if scenario.Name != "Failover Test" {
		t.Errorf("Unexpected scenario name: %s", scenario.Name)
	}

	if len(scenario.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(scenario.Steps))
	}
}
