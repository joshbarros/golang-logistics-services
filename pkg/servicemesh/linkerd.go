package servicemesh

import (
	"context"
	"fmt"
)

// LinkerdClient implements MeshClient for Linkerd
type LinkerdClient struct {
	config    Config
	namespace string
}

// NewLinkerdClient creates a new Linkerd client
func NewLinkerdClient(config Config) *LinkerdClient {
	return &LinkerdClient{
		config:    config,
		namespace: config.Namespace,
	}
}

// ApplyTrafficPolicy applies Linkerd traffic policy
func (c *LinkerdClient) ApplyTrafficPolicy(ctx context.Context, policy TrafficPolicy) error {
	yaml := c.generateTrafficSplitYAML(policy)
	fmt.Printf("Apply Linkerd TrafficSplit:\n%s\n", yaml)
	return nil
}

// GetServiceEndpoints returns service endpoints from Linkerd
func (c *LinkerdClient) GetServiceEndpoints(ctx context.Context, serviceName string) ([]Endpoint, error) {
	return []Endpoint{
		{
			Address: "10.0.1.1",
			Port:    8080,
			Weight:  100,
			Labels: map[string]string{
				"version": "v1",
			},
		},
	}, nil
}

// EnableMTLS enables mutual TLS for the service
func (c *LinkerdClient) EnableMTLS(ctx context.Context) error {
	// Linkerd enables mTLS by default for all meshed services
	fmt.Printf("Linkerd mTLS is enabled by default for meshed services\n")
	return nil
}

// CreateVirtualService creates a Linkerd TrafficSplit
func (c *LinkerdClient) CreateVirtualService(ctx context.Context, vs VirtualService) error {
	yaml := c.generateTrafficSplitFromVirtualService(vs)
	fmt.Printf("Apply Linkerd TrafficSplit:\n%s\n", yaml)
	return nil
}

// CreateDestinationRule creates Linkerd ServerPolicy
func (c *LinkerdClient) CreateDestinationRule(ctx context.Context, dr DestinationRule) error {
	yaml := c.generateServerPolicyYAML(dr)
	fmt.Printf("Apply Linkerd ServerPolicy:\n%s\n", yaml)
	return nil
}

// generateTrafficSplitYAML generates Linkerd TrafficSplit YAML
func (c *LinkerdClient) generateTrafficSplitYAML(policy TrafficPolicy) string {
	return fmt.Sprintf(`apiVersion: split.smi-spec.io/v1alpha2
kind: TrafficSplit
metadata:
  name: %s
  namespace: %s
spec:
  service: %s
  backends:
  - service: %s-v1
    weight: 100
`,
		c.config.ServiceName,
		c.namespace,
		c.config.ServiceName,
		c.config.ServiceName,
	)
}

// generateTrafficSplitFromVirtualService generates TrafficSplit from VirtualService
func (c *LinkerdClient) generateTrafficSplitFromVirtualService(vs VirtualService) string {
	yaml := fmt.Sprintf(`apiVersion: split.smi-spec.io/v1alpha2
kind: TrafficSplit
metadata:
  name: %s
  namespace: %s
spec:
  service: %s
  backends:
`,
		vs.Name,
		vs.Namespace,
		vs.Name,
	)

	// Add backends from HTTP routes
	if len(vs.HTTP) > 0 && len(vs.HTTP[0].Route) > 0 {
		for _, route := range vs.HTTP[0].Route {
			yaml += fmt.Sprintf(`  - service: %s-%s
    weight: %d
`,
				route.Destination.Host,
				route.Destination.Subset,
				route.Weight,
			)
		}
	}

	return yaml
}

// generateServerPolicyYAML generates Linkerd ServerPolicy YAML
func (c *LinkerdClient) generateServerPolicyYAML(dr DestinationRule) string {
	return fmt.Sprintf(`apiVersion: policy.linkerd.io/v1beta1
kind: ServerPolicy
metadata:
  name: %s
  namespace: %s
spec:
  targetRef:
    group: ""
    kind: Service
    name: %s
  server:
    selector:
      matchLabels:
        app: %s
`,
		dr.Name,
		dr.Namespace,
		dr.Host,
		dr.Host,
	)
}

// GenerateLinkerdManifests generates all Linkerd manifests for a service
func GenerateLinkerdManifests(config Config) map[string]string {
	client := NewLinkerdClient(config)

	manifests := make(map[string]string)

	// 1. TrafficSplit for traffic management
	manifests["traffic-split.yaml"] = client.generateTrafficSplitYAML(config.TrafficPolicy)

	// 2. Service Profile for retries and timeouts
	manifests["service-profile.yaml"] = generateServiceProfileYAML(config)

	// 3. ServerPolicy for authorization
	manifests["server-policy.yaml"] = generateServerPolicyForService(config)

	return manifests
}

// generateServiceProfileYAML generates Linkerd ServiceProfile
func generateServiceProfileYAML(config Config) string {
	return fmt.Sprintf(`apiVersion: linkerd.io/v1alpha2
kind: ServiceProfile
metadata:
  name: %s.%s.svc.cluster.local
  namespace: %s
spec:
  routes:
  - name: health
    condition:
      method: GET
      pathRegex: /health
    isRetryable: false
  - name: api
    condition:
      method: POST
      pathRegex: /api/.*
    isRetryable: true
    timeout: %s
    retryBudget:
      retryRatio: 0.2
      minRetriesPerSecond: 10
      ttl: 10s
`,
		config.ServiceName,
		config.Namespace,
		config.Namespace,
		config.TrafficPolicy.Timeout,
	)
}

// generateServerPolicyForService generates ServerPolicy for service
func generateServerPolicyForService(config Config) string {
	return fmt.Sprintf(`apiVersion: policy.linkerd.io/v1beta1
kind: ServerPolicy
metadata:
  name: %s
  namespace: %s
spec:
  targetRef:
    group: ""
    kind: Service
    name: %s
  server:
    selector:
      matchLabels:
        app: %s
  authorizations:
  - name: allow-same-namespace
    client:
      meshTLS:
        serviceAccounts:
        - name: "*"
`,
		config.ServiceName,
		config.Namespace,
		config.ServiceName,
		config.ServiceName,
	)
}

// Example Linkerd annotations for Kubernetes deployment:
//
// apiVersion: apps/v1
// kind: Deployment
// metadata:
//   name: order-service
//   annotations:
//     # Linkerd proxy configuration
//     linkerd.io/inject: enabled
//     config.linkerd.io/proxy-cpu-request: "100m"
//     config.linkerd.io/proxy-memory-request: "128Mi"
//     config.linkerd.io/proxy-cpu-limit: "1000m"
//     config.linkerd.io/proxy-memory-limit: "512Mi"
// spec:
//   template:
//     metadata:
//       annotations:
//         # Skip outbound ports (database, external APIs)
//         config.linkerd.io/skip-outbound-ports: "5432,3306"
//         # Skip inbound ports (admin interface)
//         config.linkerd.io/skip-inbound-ports: "9090"
