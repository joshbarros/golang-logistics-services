package servicemesh

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.MeshType != "istio" {
		t.Errorf("Expected default mesh type 'istio', got '%s'", config.MeshType)
	}

	if config.Namespace != "default" {
		t.Errorf("Expected default namespace 'default', got '%s'", config.Namespace)
	}

	if !config.EnableMTLS {
		t.Error("Expected mTLS to be enabled by default")
	}

	if !config.EnableTracing {
		t.Error("Expected tracing to be enabled by default")
	}

	if !config.EnableMetrics {
		t.Error("Expected metrics to be enabled by default")
	}
}

func TestDefaultTrafficPolicy(t *testing.T) {
	policy := DefaultTrafficPolicy()

	if policy.LoadBalancer.Type != "ROUND_ROBIN" {
		t.Errorf("Expected ROUND_ROBIN load balancer, got '%s'", policy.LoadBalancer.Type)
	}

	if policy.ConnectionPool.HTTP.HTTP1MaxPendingRequests != 100 {
		t.Errorf("Expected 100 max pending requests, got %d",
			policy.ConnectionPool.HTTP.HTTP1MaxPendingRequests)
	}

	if policy.OutlierDetection.ConsecutiveErrors != 5 {
		t.Errorf("Expected 5 consecutive errors, got %d",
			policy.OutlierDetection.ConsecutiveErrors)
	}

	if policy.Retry.Attempts != 3 {
		t.Errorf("Expected 3 retry attempts, got %d", policy.Retry.Attempts)
	}

	if policy.Timeout != 30*time.Second {
		t.Errorf("Expected 30s timeout, got %v", policy.Timeout)
	}
}

func TestNewManager(t *testing.T) {
	tests := []struct {
		name     string
		meshType string
		wantErr  bool
	}{
		{"Istio", "istio", false},
		{"Linkerd", "linkerd", false},
		{"Consul", "consul", false},
		{"Invalid", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.MeshType = tt.meshType
			config.ServiceName = "test-service"

			manager, err := NewManager(config)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error for invalid mesh type")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if manager == nil {
				t.Error("Manager is nil")
			}
		})
	}
}

func TestGenerateIstioManifests(t *testing.T) {
	config := DefaultConfig()
	config.ServiceName = "order-service"
	config.ServiceVersion = "v1.0.0"
	config.Namespace = "production"
	config.EnableMTLS = true

	manifests := GenerateIstioManifests(config)

	// Check required manifests are generated
	requiredManifests := []string{
		"destination-rule.yaml",
		"peer-authentication.yaml",
		"virtual-service.yaml",
		"service-entry.yaml",
		"gateway.yaml",
	}

	for _, name := range requiredManifests {
		if content, exists := manifests[name]; !exists {
			t.Errorf("Missing manifest: %s", name)
		} else if content == "" {
			t.Errorf("Empty manifest: %s", name)
		}
	}

	// Verify destination rule contains service name
	if dr, ok := manifests["destination-rule.yaml"]; ok {
		if !contains(dr, "order-service") {
			t.Error("Destination rule doesn't contain service name")
		}
		if !contains(dr, "production") {
			t.Error("Destination rule doesn't contain namespace")
		}
	}

	// Verify mTLS configuration
	if pa, ok := manifests["peer-authentication.yaml"]; ok {
		if !contains(pa, "STRICT") {
			t.Error("PeerAuthentication doesn't have STRICT mTLS mode")
		}
	}
}

func TestGenerateLinkerdManifests(t *testing.T) {
	config := DefaultConfig()
	config.MeshType = "linkerd"
	config.ServiceName = "shipment-service"
	config.Namespace = "production"

	manifests := GenerateLinkerdManifests(config)

	requiredManifests := []string{
		"traffic-split.yaml",
		"service-profile.yaml",
		"server-policy.yaml",
	}

	for _, name := range requiredManifests {
		if content, exists := manifests[name]; !exists {
			t.Errorf("Missing Linkerd manifest: %s", name)
		} else if content == "" {
			t.Errorf("Empty Linkerd manifest: %s", name)
		}
	}
}

func TestGenerateConsulManifests(t *testing.T) {
	config := DefaultConfig()
	config.MeshType = "consul"
	config.ServiceName = "inventory-service"
	config.Namespace = "production"
	config.EnableMTLS = true

	manifests := GenerateConsulManifests(config)

	requiredManifests := []string{
		"service-defaults.yaml",
		"proxy-defaults.yaml",
		"service-intentions.yaml",
		"service-resolver.yaml",
	}

	for _, name := range requiredManifests {
		if content, exists := manifests[name]; !exists {
			t.Errorf("Missing Consul manifest: %s", name)
		} else if content == "" {
			t.Errorf("Empty Consul manifest: %s", name)
		}
	}
}

func TestVirtualService(t *testing.T) {
	vs := VirtualService{
		Name:      "test-service",
		Namespace: "default",
		Hosts:     []string{"test-service"},
		HTTP: []HTTPRoute{
			{
				Name: "route1",
				Route: []HTTPRouteDestination{
					{
						Destination: Destination{
							Host:   "test-service",
							Subset: "v1",
						},
						Weight: 90,
					},
					{
						Destination: Destination{
							Host:   "test-service",
							Subset: "v2",
						},
						Weight: 10,
					},
				},
			},
		},
	}

	if vs.Name != "test-service" {
		t.Errorf("Expected name 'test-service', got '%s'", vs.Name)
	}

	if len(vs.HTTP) != 1 {
		t.Errorf("Expected 1 HTTP route, got %d", len(vs.HTTP))
	}

	if len(vs.HTTP[0].Route) != 2 {
		t.Errorf("Expected 2 destinations, got %d", len(vs.HTTP[0].Route))
	}

	totalWeight := 0
	for _, dest := range vs.HTTP[0].Route {
		totalWeight += dest.Weight
	}

	if totalWeight != 100 {
		t.Errorf("Expected total weight 100, got %d", totalWeight)
	}
}

func TestDestinationRule(t *testing.T) {
	dr := DestinationRule{
		Name:      "test-service",
		Namespace: "default",
		Host:      "test-service",
		Subsets: []Subset{
			{
				Name: "v1",
				Labels: map[string]string{
					"version": "1.0.0",
				},
			},
			{
				Name: "v2",
				Labels: map[string]string{
					"version": "2.0.0",
				},
			},
		},
	}

	if len(dr.Subsets) != 2 {
		t.Errorf("Expected 2 subsets, got %d", len(dr.Subsets))
	}

	if dr.Subsets[0].Name != "v1" {
		t.Errorf("Expected subset name 'v1', got '%s'", dr.Subsets[0].Name)
	}

	if dr.Subsets[0].Labels["version"] != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", dr.Subsets[0].Labels["version"])
	}
}

func TestHTTPFaultInjection(t *testing.T) {
	fault := HTTPFaultInjection{
		Delay: &Delay{
			Percentage: 10.0,
			FixedDelay: 5 * time.Second,
		},
		Abort: &Abort{
			Percentage: 5.0,
			HTTPStatus: 503,
		},
	}

	if fault.Delay.Percentage != 10.0 {
		t.Errorf("Expected delay percentage 10.0, got %.1f", fault.Delay.Percentage)
	}

	if fault.Delay.FixedDelay != 5*time.Second {
		t.Errorf("Expected delay 5s, got %v", fault.Delay.FixedDelay)
	}

	if fault.Abort.HTTPStatus != 503 {
		t.Errorf("Expected abort status 503, got %d", fault.Abort.HTTPStatus)
	}
}

func TestLoadBalancerConfig(t *testing.T) {
	tests := []struct {
		name string
		lb   LoadBalancerConfig
	}{
		{
			name: "RoundRobin",
			lb: LoadBalancerConfig{
				Type: "ROUND_ROBIN",
			},
		},
		{
			name: "LeastRequest",
			lb: LoadBalancerConfig{
				Type: "LEAST_REQUEST",
			},
		},
		{
			name: "ConsistentHash",
			lb: LoadBalancerConfig{
				Type: "CONSISTENT_HASH",
				ConsistentHash: &ConsistentHashConfig{
					HTTPHeaderName: "user-id",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.lb.Type == "" {
				t.Error("Load balancer type is empty")
			}

			if tt.lb.Type == "CONSISTENT_HASH" {
				if tt.lb.ConsistentHash == nil {
					t.Error("ConsistentHash config is nil")
				}
				if tt.lb.ConsistentHash.HTTPHeaderName == "" {
					t.Error("HTTP header name is empty")
				}
			}
		})
	}
}

func TestRetryConfig(t *testing.T) {
	retry := RetryConfig{
		Attempts:      3,
		PerTryTimeout: 2 * time.Second,
		RetryOn:       []string{"5xx", "reset", "connect-failure"},
	}

	if retry.Attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", retry.Attempts)
	}

	if retry.PerTryTimeout != 2*time.Second {
		t.Errorf("Expected 2s timeout, got %v", retry.PerTryTimeout)
	}

	if len(retry.RetryOn) != 3 {
		t.Errorf("Expected 3 retry conditions, got %d", len(retry.RetryOn))
	}
}

func TestOutlierDetectionConfig(t *testing.T) {
	od := OutlierDetectionConfig{
		ConsecutiveErrors:  5,
		Interval:           30 * time.Second,
		BaseEjectionTime:   30 * time.Second,
		MaxEjectionPercent: 50,
		MinHealthPercent:   50,
	}

	if od.ConsecutiveErrors != 5 {
		t.Errorf("Expected 5 consecutive errors, got %d", od.ConsecutiveErrors)
	}

	if od.MaxEjectionPercent > 100 {
		t.Error("Max ejection percent exceeds 100")
	}

	if od.MinHealthPercent < 0 || od.MinHealthPercent > 100 {
		t.Error("Min health percent out of range")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
