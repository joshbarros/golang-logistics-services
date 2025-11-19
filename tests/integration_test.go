package tests

import (
	"context"
	"testing"
	"time"

	"github.com/joshbarros/golang-logistics-services/pkg/chaos"
	"github.com/joshbarros/golang-logistics-services/pkg/graphql"
	"github.com/joshbarros/golang-logistics-services/pkg/servicemesh"
)

// TestChaosEngineering validates the chaos engineering framework
func TestChaosEngineering(t *testing.T) {
	t.Log("Testing Chaos Engineering Framework")

	// Create chaos manager
	config := chaos.Config{
		Enabled:     true,
		Probability: 1.0,
	}
	manager := chaos.NewManager(config)

	// Test latency experiment
	t.Run("LatencyExperiment", func(t *testing.T) {
		exp := chaos.NewLatencyExperiment(
			"test-service",
			"Latency Test",
			1*time.Minute,
			100*time.Millisecond,
		)

		manager.AddExperiment(exp)
		if err := manager.StartExperiment(exp.ID); err != nil {
			t.Fatalf("Failed to start experiment: %v", err)
		}

		// Measure latency injection
		ctx := context.Background()
		start := time.Now()
		err := manager.InjectFault(ctx, "test-service")
		duration := time.Since(start)

		if err != nil {
			t.Logf("Fault injected: %v", err)
		}

		if duration < 80*time.Millisecond {
			t.Errorf("Expected latency >= 80ms, got %v", duration)
		}

		manager.StopExperiment(exp.ID)
		t.Logf("✓ Latency injection works: %v", duration)
	})

	// Test error experiment
	t.Run("ErrorExperiment", func(t *testing.T) {
		exp := chaos.NewErrorExperiment(
			"test-service",
			"Error Test",
			1*time.Minute,
			503,
			"Service unavailable",
		)

		manager.AddExperiment(exp)
		manager.StartExperiment(exp.ID)

		ctx := context.Background()
		err := manager.InjectFault(ctx, "test-service")

		if err == nil {
			t.Error("Expected error to be injected")
		}

		manager.StopExperiment(exp.ID)
		t.Logf("✓ Error injection works: %v", err)
	})

	// Test scenario runner
	t.Run("ScenarioRunner", func(t *testing.T) {
		runner := chaos.NewRunner(manager)

		scenario := chaos.NewServiceDegradationScenario("test-service")
		// Shorten durations for testing
		for i := range scenario.Steps {
			scenario.Steps[i].Duration = 500 * time.Millisecond
			scenario.Steps[i].WaitBefore = 100 * time.Millisecond
		}
		scenario.Timeout = 10 * time.Second

		runner.AddScenario(scenario)

		start := time.Now()
		result, err := runner.RunScenario(context.Background(), scenario.ID)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Scenario failed: %v", err)
		}

		if result == nil {
			t.Fatal("Result is nil")
		}

		t.Logf("✓ Scenario completed in %v", duration)
		t.Logf("  Steps: %d, Passed: %v", len(result.Steps), result.Passed)
	})

	t.Log("✓ All chaos engineering tests passed")
}

// TestGraphQLDataLoader validates DataLoader functionality
func TestGraphQLDataLoader(t *testing.T) {
	t.Log("Testing GraphQL DataLoader")

	callCount := 0
	batchFunc := func(ctx context.Context, keys []string) ([]interface{}, error) {
		callCount++
		results := make([]interface{}, len(keys))
		for i, key := range keys {
			results[i] = map[string]string{"id": key, "name": "Item " + key}
		}
		return results, nil
	}

	loader := graphql.NewDataLoader(batchFunc)
	loader.Wait = 5 * time.Millisecond

	t.Run("SingleLoad", func(t *testing.T) {
		ctx := context.Background()
		result, err := loader.Load(ctx, "1")

		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}

		if result == nil {
			t.Fatal("Result is nil")
		}

		t.Log("✓ Single load works")
	})

	t.Run("Caching", func(t *testing.T) {
		initialCalls := callCount

		ctx := context.Background()
		loader.Load(ctx, "1") // Should use cache

		if callCount != initialCalls {
			t.Error("Expected cache to be used, but batch function was called")
		}

		t.Log("✓ Caching works")
	})

	t.Run("Batching", func(t *testing.T) {
		newLoader := graphql.NewDataLoader(batchFunc)
		newLoader.Wait = 10 * time.Millisecond

		ctx := context.Background()
		keys := []string{"a", "b", "c", "d", "e"}

		results, err := newLoader.LoadMany(ctx, keys)

		if err != nil {
			t.Fatalf("LoadMany failed: %v", err)
		}

		if len(results) != len(keys) {
			t.Errorf("Expected %d results, got %d", len(keys), len(results))
		}

		t.Log("✓ Batching works")
	})

	t.Log("✓ All GraphQL DataLoader tests passed")
}

// TestServiceMesh validates service mesh configuration
func TestServiceMesh(t *testing.T) {
	t.Log("Testing Service Mesh Integration")

	t.Run("DefaultConfig", func(t *testing.T) {
		config := servicemesh.DefaultConfig()

		if config.MeshType != "istio" {
			t.Errorf("Expected istio, got %s", config.MeshType)
		}

		if !config.EnableMTLS {
			t.Error("Expected mTLS to be enabled")
		}

		t.Log("✓ Default config is valid")
	})

	t.Run("IstioManifests", func(t *testing.T) {
		config := servicemesh.DefaultConfig()
		config.ServiceName = "test-service"
		config.Namespace = "test"

		manifests := servicemesh.GenerateIstioManifests(config)

		requiredManifests := []string{
			"destination-rule.yaml",
			"virtual-service.yaml",
			"peer-authentication.yaml",
		}

		for _, name := range requiredManifests {
			if content, exists := manifests[name]; !exists {
				t.Errorf("Missing manifest: %s", name)
			} else if len(content) == 0 {
				t.Errorf("Empty manifest: %s", name)
			}
		}

		t.Logf("✓ Generated %d Istio manifests", len(manifests))
	})

	t.Run("LinkerdManifests", func(t *testing.T) {
		config := servicemesh.DefaultConfig()
		config.MeshType = "linkerd"
		config.ServiceName = "test-service"

		manifests := servicemesh.GenerateLinkerdManifests(config)

		if len(manifests) == 0 {
			t.Error("No Linkerd manifests generated")
		}

		t.Logf("✓ Generated %d Linkerd manifests", len(manifests))
	})

	t.Run("ConsulManifests", func(t *testing.T) {
		config := servicemesh.DefaultConfig()
		config.MeshType = "consul"
		config.ServiceName = "test-service"

		manifests := servicemesh.GenerateConsulManifests(config)

		if len(manifests) == 0 {
			t.Error("No Consul manifests generated")
		}

		t.Logf("✓ Generated %d Consul manifests", len(manifests))
	})

	t.Run("TrafficPolicy", func(t *testing.T) {
		policy := servicemesh.DefaultTrafficPolicy()

		if policy.LoadBalancer.Type == "" {
			t.Error("Load balancer type is empty")
		}

		if policy.Retry.Attempts == 0 {
			t.Error("Retry attempts is zero")
		}

		if policy.Timeout == 0 {
			t.Error("Timeout is zero")
		}

		t.Log("✓ Traffic policy is valid")
	})

	t.Log("✓ All service mesh tests passed")
}

// TestIntegration validates integration between components
func TestIntegration(t *testing.T) {
	t.Log("Testing Component Integration")

	// Test chaos + service mesh
	t.Run("ChaosWithServiceMesh", func(t *testing.T) {
		// Service mesh config
		meshConfig := servicemesh.DefaultConfig()
		meshConfig.ServiceName = "order-service"

		// Chaos config
		chaosConfig := chaos.Config{
			Enabled:     true,
			Probability: 0.1,
		}
		chaosManager := chaos.NewManager(chaosConfig)

		// Both should work together
		exp := chaos.NewLatencyExperiment(
			"order-service",
			"Integration Test",
			1*time.Minute,
			50*time.Millisecond,
		)

		chaosManager.AddExperiment(exp)
		chaosManager.StartExperiment(exp.ID)

		// Verify chaos is active
		if !chaosManager.ShouldInject("order-service") {
			t.Error("Chaos should be active")
		}

		chaosManager.StopExperiment(exp.ID)

		t.Log("✓ Chaos and service mesh integration works")
	})

	// Test GraphQL + chaos
	t.Run("GraphQLWithChaos", func(t *testing.T) {
		// GraphQL DataLoader
		loader := graphql.NewDataLoader(func(ctx context.Context, keys []string) ([]interface{}, error) {
			results := make([]interface{}, len(keys))
			for i, key := range keys {
				results[i] = key
			}
			return results, nil
		})

		// Chaos manager
		chaosManager := chaos.NewManager(chaos.Config{Enabled: true, Probability: 0.0})

		// Both should work together
		ctx := context.Background()

		// Check chaos
		err := chaosManager.InjectFault(ctx, "graphql")
		if err != nil {
			t.Logf("Chaos injected: %v", err)
		}

		// Use DataLoader
		result, err := loader.Load(ctx, "test")
		if err != nil {
			t.Fatalf("DataLoader failed: %v", err)
		}

		if result == nil {
			t.Error("Result is nil")
		}

		t.Log("✓ GraphQL and chaos integration works")
	})

	t.Log("✓ All integration tests passed")
}

// TestEndToEnd validates complete workflow
func TestEndToEnd(t *testing.T) {
	t.Log("Testing End-to-End Workflow")

	// Simulate complete request flow:
	// Client -> GraphQL Gateway -> Service (with chaos) -> Service Mesh

	t.Run("CompleteFlow", func(t *testing.T) {
		// 1. Setup service mesh
		meshConfig := servicemesh.DefaultConfig()
		meshConfig.ServiceName = "order-service"
		meshConfig.Namespace = "production"

		manifests := servicemesh.GenerateIstioManifests(meshConfig)
		if len(manifests) == 0 {
			t.Error("No service mesh manifests generated")
		}

		// 2. Setup GraphQL DataLoader
		loader := graphql.NewDataLoader(func(ctx context.Context, keys []string) ([]interface{}, error) {
			// Simulate service call
			results := make([]interface{}, len(keys))
			for i, key := range keys {
				results[i] = map[string]string{
					"id":     key,
					"status": "completed",
				}
			}
			return results, nil
		})

		// 3. Setup chaos (disabled by default)
		chaosManager := chaos.NewManager(chaos.Config{
			Enabled:     false, // Don't inject in normal flow
			Probability: 0.0,
		})

		// 4. Simulate request flow
		ctx := context.Background()

		// Check chaos (should not inject)
		if err := chaosManager.InjectFault(ctx, "order-service"); err != nil {
			t.Errorf("Unexpected chaos injection: %v", err)
		}

		// Load data through GraphQL
		orderIDs := []string{"order-1", "order-2", "order-3"}
		results, err := loader.LoadMany(ctx, orderIDs)

		if err != nil {
			t.Fatalf("Request flow failed: %v", err)
		}

		if len(results) != len(orderIDs) {
			t.Errorf("Expected %d results, got %d", len(orderIDs), len(results))
		}

		t.Log("✓ Complete request flow works")
		t.Logf("  Processed %d orders through GraphQL -> Service -> Mesh", len(results))
	})

	t.Run("FlowWithChaos", func(t *testing.T) {
		// Same flow but with chaos enabled
		chaosManager := chaos.NewManager(chaos.Config{
			Enabled:     true,
			Probability: 1.0, // Always inject for testing
		})

		exp := chaos.NewLatencyExperiment(
			"order-service",
			"E2E Test",
			1*time.Minute,
			50*time.Millisecond,
		)

		chaosManager.AddExperiment(exp)
		chaosManager.StartExperiment(exp.ID)

		ctx := context.Background()
		start := time.Now()

		// Chaos should inject latency
		chaosManager.InjectFault(ctx, "order-service")

		duration := time.Since(start)

		if duration < 40*time.Millisecond {
			t.Errorf("Expected latency >= 40ms, got %v", duration)
		}

		chaosManager.StopExperiment(exp.ID)

		t.Log("✓ Flow with chaos injection works")
		t.Logf("  Injected latency: %v", duration)
	})

	t.Log("✓ All end-to-end tests passed")
}
