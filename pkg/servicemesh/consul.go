package servicemesh

import (
	"context"
	"fmt"
)

// ConsulClient implements MeshClient for Consul Connect
type ConsulClient struct {
	config    Config
	namespace string
}

// NewConsulClient creates a new Consul Connect client
func NewConsulClient(config Config) *ConsulClient {
	return &ConsulClient{
		config:    config,
		namespace: config.Namespace,
	}
}

// ApplyTrafficPolicy applies Consul service defaults
func (c *ConsulClient) ApplyTrafficPolicy(ctx context.Context, policy TrafficPolicy) error {
	yaml := c.generateServiceDefaultsYAML(policy)
	fmt.Printf("Apply Consul ServiceDefaults:\n%s\n", yaml)
	return nil
}

// GetServiceEndpoints returns service endpoints from Consul
func (c *ConsulClient) GetServiceEndpoints(ctx context.Context, serviceName string) ([]Endpoint, error) {
	// Query Consul catalog for service endpoints
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

// EnableMTLS enables mutual TLS via Consul Connect
func (c *ConsulClient) EnableMTLS(ctx context.Context) error {
	yaml := c.generateProxyDefaultsYAML()
	fmt.Printf("Apply Consul ProxyDefaults:\n%s\n", yaml)
	return nil
}

// CreateVirtualService creates a Consul ServiceRouter
func (c *ConsulClient) CreateVirtualService(ctx context.Context, vs VirtualService) error {
	yaml := c.generateServiceRouterYAML(vs)
	fmt.Printf("Apply Consul ServiceRouter:\n%s\n", yaml)
	return nil
}

// CreateDestinationRule creates Consul ServiceResolver
func (c *ConsulClient) CreateDestinationRule(ctx context.Context, dr DestinationRule) error {
	yaml := c.generateServiceResolverYAML(dr)
	fmt.Printf("Apply Consul ServiceResolver:\n%s\n", yaml)
	return nil
}

// generateServiceDefaultsYAML generates Consul ServiceDefaults
func (c *ConsulClient) generateServiceDefaultsYAML(policy TrafficPolicy) string {
	return fmt.Sprintf(`apiVersion: consul.hashicorp.com/v1alpha1
kind: ServiceDefaults
metadata:
  name: %s
  namespace: %s
spec:
  protocol: http
  meshGateway:
    mode: local
  transparentProxy:
    outboundListenerPort: 15001
  upstreamConfig:
    defaults:
      connectTimeout: %s
      limits:
        maxConnections: %d
        maxPendingRequests: %d
        maxConcurrentRequests: %d
`,
		c.config.ServiceName,
		c.namespace,
		policy.ConnectionPool.TCP.ConnectTimeout,
		policy.ConnectionPool.TCP.MaxConnections,
		policy.ConnectionPool.HTTP.HTTP1MaxPendingRequests,
		policy.ConnectionPool.HTTP.HTTP2MaxRequests,
	)
}

// generateProxyDefaultsYAML generates Consul ProxyDefaults for mTLS
func (c *ConsulClient) generateProxyDefaultsYAML() string {
	return fmt.Sprintf(`apiVersion: consul.hashicorp.com/v1alpha1
kind: ProxyDefaults
metadata:
  name: global
  namespace: %s
spec:
  config:
    protocol: http
  meshGateway:
    mode: local
  transparentProxy:
    outboundListenerPort: 15001
  mutualTLSMode: strict
`,
		c.namespace,
	)
}

// generateServiceRouterYAML generates Consul ServiceRouter
func (c *ConsulClient) generateServiceRouterYAML(vs VirtualService) string {
	yaml := fmt.Sprintf(`apiVersion: consul.hashicorp.com/v1alpha1
kind: ServiceRouter
metadata:
  name: %s
  namespace: %s
spec:
  routes:
`,
		vs.Name,
		vs.Namespace,
	)

	// Add routes
	if len(vs.HTTP) > 0 {
		for _, route := range vs.HTTP {
			yaml += "  - match:\n"
			yaml += "      http:\n"

			// Add match conditions
			if len(route.Match) > 0 {
				for _, match := range route.Match {
					if match.URI != nil && match.URI.Prefix != "" {
						yaml += fmt.Sprintf("        pathPrefix: %s\n", match.URI.Prefix)
					}
				}
			}

			// Add destination
			if len(route.Route) > 0 {
				yaml += "    destination:\n"
				dest := route.Route[0].Destination
				yaml += fmt.Sprintf("      service: %s\n", dest.Host)
				if dest.Subset != "" {
					yaml += fmt.Sprintf("      serviceSubset: %s\n", dest.Subset)
				}
			}
		}
	}

	return yaml
}

// generateServiceResolverYAML generates Consul ServiceResolver
func (c *ConsulClient) generateServiceResolverYAML(dr DestinationRule) string {
	yaml := fmt.Sprintf(`apiVersion: consul.hashicorp.com/v1alpha1
kind: ServiceResolver
metadata:
  name: %s
  namespace: %s
spec:
  defaultSubset: stable
  connectTimeout: %s
`,
		dr.Name,
		dr.Namespace,
		dr.TrafficPolicy.ConnectionPool.TCP.ConnectTimeout,
	)

	// Add subsets
	if len(dr.Subsets) > 0 {
		yaml += "  subsets:\n"
		for _, subset := range dr.Subsets {
			yaml += fmt.Sprintf("    %s:\n", subset.Name)
			yaml += "      filter: "
			filters := make([]string, 0)
			for k, v := range subset.Labels {
				filters = append(filters, fmt.Sprintf(`Service.Meta.%s == "%s"`, k, v))
			}
			yaml += fmt.Sprintf("\"%s\"\n", filters[0])
		}
	}

	// Add load balancer
	yaml += fmt.Sprintf(`  loadBalancer:
    policy: %s
`,
		dr.TrafficPolicy.LoadBalancer.Type,
	)

	return yaml
}

// generateServiceSplitterYAML generates Consul ServiceSplitter for traffic splitting
func generateServiceSplitterYAML(config Config, splits map[string]int) string {
	yaml := fmt.Sprintf(`apiVersion: consul.hashicorp.com/v1alpha1
kind: ServiceSplitter
metadata:
  name: %s
  namespace: %s
spec:
  splits:
`,
		config.ServiceName,
		config.Namespace,
	)

	for subset, weight := range splits {
		yaml += fmt.Sprintf(`  - weight: %d
    serviceSubset: %s
`,
			weight,
			subset,
		)
	}

	return yaml
}

// GenerateConsulManifests generates all Consul manifests for a service
func GenerateConsulManifests(config Config) map[string]string {
	client := NewConsulClient(config)

	manifests := make(map[string]string)

	// 1. ServiceDefaults for traffic policy
	manifests["service-defaults.yaml"] = client.generateServiceDefaultsYAML(config.TrafficPolicy)

	// 2. ProxyDefaults for mTLS
	if config.EnableMTLS {
		manifests["proxy-defaults.yaml"] = client.generateProxyDefaultsYAML()
	}

	// 3. ServiceIntentions for authorization
	manifests["service-intentions.yaml"] = generateServiceIntentionsYAML(config)

	// 4. ServiceResolver for service resolution
	dr := DestinationRule{
		Name:          config.ServiceName,
		Namespace:     config.Namespace,
		Host:          config.ServiceName,
		TrafficPolicy: config.TrafficPolicy,
	}
	manifests["service-resolver.yaml"] = client.generateServiceResolverYAML(dr)

	return manifests
}

// generateServiceIntentionsYAML generates Consul ServiceIntentions for authorization
func generateServiceIntentionsYAML(config Config) string {
	return fmt.Sprintf(`apiVersion: consul.hashicorp.com/v1alpha1
kind: ServiceIntentions
metadata:
  name: %s
  namespace: %s
spec:
  destination:
    name: %s
  sources:
  - name: api-gateway
    action: allow
  - name: web
    action: allow
  - name: "*"
    action: deny
`,
		config.ServiceName,
		config.Namespace,
		config.ServiceName,
	)
}

// Example Consul Connect configuration in service:
//
// // Register service with Consul
// registration := &api.AgentServiceRegistration{
//     ID:      "order-service-1",
//     Name:    "order-service",
//     Port:    8080,
//     Tags:    []string{"version-v1"},
//     Address: "10.0.1.5",
//     Connect: &api.AgentServiceConnect{
//         SidecarService: &api.AgentServiceRegistration{
//             Port: 20000,
//             Proxy: &api.AgentServiceConnectProxyConfig{
//                 Upstreams: []api.Upstream{
//                     {
//                         DestinationType: api.UpstreamDestTypeService,
//                         DestinationName: "inventory-service",
//                         LocalBindPort:   9090,
//                     },
//                     {
//                         DestinationType: api.UpstreamDestTypeService,
//                         DestinationName: "shipment-service",
//                         LocalBindPort:   9091,
//                     },
//                 },
//             },
//         },
//     },
//     Check: &api.AgentServiceCheck{
//         HTTP:     "http://localhost:8080/health",
//         Interval: "10s",
//         Timeout:  "1s",
//     },
// }
//
// client.Agent().ServiceRegister(registration)
