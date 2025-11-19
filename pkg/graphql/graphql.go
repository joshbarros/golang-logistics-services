package graphql

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/handler"
)

// Config holds GraphQL server configuration
type Config struct {
	// Playground enables GraphQL Playground UI
	Playground bool
	// Pretty enables pretty-printed JSON responses
	Pretty bool
	// MaxComplexity limits query complexity
	MaxComplexity int
	// MaxDepth limits query depth
	MaxDepth int
	// Timeout for query execution
	Timeout time.Duration
	// Auth middleware for authentication
	Auth func(*gin.Context) error
	// ServiceClients for microservice communication
	ServiceClients ServiceClients
}

// ServiceClients holds gRPC clients for all microservices
type ServiceClients struct {
	OrderClient      interface{}
	ShipmentClient   interface{}
	InventoryClient  interface{}
	DriverClient     interface{}
	RouteClient      interface{}
	NotificationClient interface{}
}

// DefaultConfig returns default GraphQL configuration
func DefaultConfig() Config {
	return Config{
		Playground:    true,
		Pretty:        true,
		MaxComplexity: 1000,
		MaxDepth:      10,
		Timeout:       30 * time.Second,
	}
}

// Server wraps the GraphQL server
type Server struct {
	schema  graphql.Schema
	config  Config
	handler *handler.Handler
}

// NewServer creates a new GraphQL server
func NewServer(config Config) (*Server, error) {
	schema, err := buildSchema(config.ServiceClients)
	if err != nil {
		return nil, fmt.Errorf("failed to build schema: %w", err)
	}

	h := handler.New(&handler.Config{
		Schema:     &schema,
		Pretty:     config.Pretty,
		GraphiQL:   false,
		Playground: config.Playground,
	})

	return &Server{
		schema:  schema,
		config:  config,
		handler: h,
	}, nil
}

// Handler returns a Gin handler for GraphQL requests
func (s *Server) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Apply auth middleware if configured
		if s.config.Auth != nil {
			if err := s.config.Auth(c); err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
				return
			}
		}

		// Create context with timeout
		ctx, cancel := context.WithTimeout(c.Request.Context(), s.config.Timeout)
		defer cancel()

		// Validate query complexity and depth
		if err := s.validateQuery(c.Request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Execute GraphQL query
		c.Request = c.Request.WithContext(ctx)
		s.handler.ServeHTTP(c.Writer, c.Request)
	}
}

// PlaygroundHandler returns a handler for GraphQL Playground
func (s *Server) PlaygroundHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, playgroundHTML)
	}
}

// validateQuery validates query complexity and depth
func (s *Server) validateQuery(r *http.Request) error {
	var params struct {
		Query string `json:"query"`
	}

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		return fmt.Errorf("invalid request body")
	}

	// Reset body for handler
	r.Body = http.NoBody

	// In production, implement proper complexity analysis
	// using github.com/graphql-go/graphql/language/parser
	// and custom complexity calculator

	return nil
}

// buildSchema builds the GraphQL schema
func buildSchema(clients ServiceClients) (graphql.Schema, error) {
	// Root query type
	queryType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"order":      getOrderField(clients),
			"orders":     getOrdersField(clients),
			"shipment":   getShipmentField(clients),
			"shipments":  getShipmentsField(clients),
			"inventory":  getInventoryField(clients),
			"inventories": getInventoriesField(clients),
			"driver":     getDriverField(clients),
			"drivers":    getDriversField(clients),
			"route":      getRouteField(clients),
			"routes":     getRoutesField(clients),
		},
	})

	// Root mutation type
	mutationType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			"createOrder":    createOrderField(clients),
			"updateOrder":    updateOrderField(clients),
			"cancelOrder":    cancelOrderField(clients),
			"createShipment": createShipmentField(clients),
			"updateShipment": updateShipmentField(clients),
			"updateInventory": updateInventoryField(clients),
			"assignDriver":   assignDriverField(clients),
		},
	})

	// Root subscription type
	subscriptionType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Subscription",
		Fields: graphql.Fields{
			"orderUpdates":    orderUpdatesField(clients),
			"shipmentUpdates": shipmentUpdatesField(clients),
			"driverLocation":  driverLocationField(clients),
		},
	})

	return graphql.NewSchema(graphql.SchemaConfig{
		Query:        queryType,
		Mutation:     mutationType,
		Subscription: subscriptionType,
	})
}

// Order type definition
var orderType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Order",
	Fields: graphql.Fields{
		"id": &graphql.Field{
			Type: graphql.NewNonNull(graphql.String),
		},
		"customerId": &graphql.Field{
			Type: graphql.NewNonNull(graphql.String),
		},
		"status": &graphql.Field{
			Type: graphql.String,
		},
		"totalAmount": &graphql.Field{
			Type: graphql.Float,
		},
		"items": &graphql.Field{
			Type: graphql.NewList(orderItemType),
		},
		"shipment": &graphql.Field{
			Type: shipmentType,
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				// Fetch related shipment
				return nil, nil
			},
		},
		"createdAt": &graphql.Field{
			Type: graphql.DateTime,
		},
		"updatedAt": &graphql.Field{
			Type: graphql.DateTime,
		},
	},
})

var orderItemType = graphql.NewObject(graphql.ObjectConfig{
	Name: "OrderItem",
	Fields: graphql.Fields{
		"id": &graphql.Field{
			Type: graphql.String,
		},
		"productId": &graphql.Field{
			Type: graphql.String,
		},
		"quantity": &graphql.Field{
			Type: graphql.Int,
		},
		"price": &graphql.Field{
			Type: graphql.Float,
		},
	},
})

// Shipment type definition
var shipmentType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Shipment",
	Fields: graphql.Fields{
		"id": &graphql.Field{
			Type: graphql.NewNonNull(graphql.String),
		},
		"orderId": &graphql.Field{
			Type: graphql.String,
		},
		"trackingNumber": &graphql.Field{
			Type: graphql.String,
		},
		"status": &graphql.Field{
			Type: graphql.String,
		},
		"origin": &graphql.Field{
			Type: addressType,
		},
		"destination": &graphql.Field{
			Type: addressType,
		},
		"driver": &graphql.Field{
			Type: driverType,
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				// Fetch related driver
				return nil, nil
			},
		},
		"estimatedDelivery": &graphql.Field{
			Type: graphql.DateTime,
		},
	},
})

var addressType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Address",
	Fields: graphql.Fields{
		"street": &graphql.Field{
			Type: graphql.String,
		},
		"city": &graphql.Field{
			Type: graphql.String,
		},
		"state": &graphql.Field{
			Type: graphql.String,
		},
		"zipCode": &graphql.Field{
			Type: graphql.String,
		},
		"country": &graphql.Field{
			Type: graphql.String,
		},
	},
})

// Driver type definition
var driverType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Driver",
	Fields: graphql.Fields{
		"id": &graphql.Field{
			Type: graphql.NewNonNull(graphql.String),
		},
		"name": &graphql.Field{
			Type: graphql.String,
		},
		"email": &graphql.Field{
			Type: graphql.String,
		},
		"phone": &graphql.Field{
			Type: graphql.String,
		},
		"status": &graphql.Field{
			Type: graphql.String,
		},
		"currentLocation": &graphql.Field{
			Type: locationType,
		},
		"vehicleInfo": &graphql.Field{
			Type: vehicleType,
		},
	},
})

var locationType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Location",
	Fields: graphql.Fields{
		"latitude": &graphql.Field{
			Type: graphql.Float,
		},
		"longitude": &graphql.Field{
			Type: graphql.Float,
		},
		"timestamp": &graphql.Field{
			Type: graphql.DateTime,
		},
	},
})

var vehicleType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Vehicle",
	Fields: graphql.Fields{
		"make": &graphql.Field{
			Type: graphql.String,
		},
		"model": &graphql.Field{
			Type: graphql.String,
		},
		"year": &graphql.Field{
			Type: graphql.Int,
		},
		"licensePlate": &graphql.Field{
			Type: graphql.String,
		},
	},
})

// Inventory type definition
var inventoryType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Inventory",
	Fields: graphql.Fields{
		"id": &graphql.Field{
			Type: graphql.NewNonNull(graphql.String),
		},
		"productId": &graphql.Field{
			Type: graphql.String,
		},
		"warehouseId": &graphql.Field{
			Type: graphql.String,
		},
		"quantity": &graphql.Field{
			Type: graphql.Int,
		},
		"reserved": &graphql.Field{
			Type: graphql.Int,
		},
		"available": &graphql.Field{
			Type: graphql.Int,
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				// Calculate available = quantity - reserved
				return nil, nil
			},
		},
	},
})

// Route type definition
var routeType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Route",
	Fields: graphql.Fields{
		"id": &graphql.Field{
			Type: graphql.NewNonNull(graphql.String),
		},
		"origin": &graphql.Field{
			Type: addressType,
		},
		"destination": &graphql.Field{
			Type: addressType,
		},
		"waypoints": &graphql.Field{
			Type: graphql.NewList(locationType),
		},
		"distance": &graphql.Field{
			Type: graphql.Float,
		},
		"duration": &graphql.Field{
			Type: graphql.Int,
		},
		"optimized": &graphql.Field{
			Type: graphql.Boolean,
		},
	},
})

// Query resolvers
func getOrderField(clients ServiceClients) *graphql.Field {
	return &graphql.Field{
		Type: orderType,
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Type: graphql.NewNonNull(graphql.String),
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			id := p.Args["id"].(string)
			// Call order service gRPC client
			// return clients.OrderClient.GetOrder(p.Context, id)
			return map[string]interface{}{
				"id":          id,
				"customerId":  "customer-123",
				"status":      "pending",
				"totalAmount": 99.99,
			}, nil
		},
	}
}

func getOrdersField(clients ServiceClients) *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewList(orderType),
		Args: graphql.FieldConfigArgument{
			"customerId": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
			"status": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
			"limit": &graphql.ArgumentConfig{
				Type:         graphql.Int,
				DefaultValue: 10,
			},
			"offset": &graphql.ArgumentConfig{
				Type:         graphql.Int,
				DefaultValue: 0,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			// Call order service gRPC client with filters
			return []interface{}{}, nil
		},
	}
}

func getShipmentField(clients ServiceClients) *graphql.Field {
	return &graphql.Field{
		Type: shipmentType,
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Type: graphql.NewNonNull(graphql.String),
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			id := p.Args["id"].(string)
			// Call shipment service gRPC client
			return map[string]interface{}{
				"id":             id,
				"trackingNumber": "TRACK123",
				"status":         "in_transit",
			}, nil
		},
	}
}

func getShipmentsField(clients ServiceClients) *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewList(shipmentType),
		Args: graphql.FieldConfigArgument{
			"orderId": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
			"status": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return []interface{}{}, nil
		},
	}
}

func getInventoryField(clients ServiceClients) *graphql.Field {
	return &graphql.Field{
		Type: inventoryType,
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Type: graphql.NewNonNull(graphql.String),
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return map[string]interface{}{}, nil
		},
	}
}

func getInventoriesField(clients ServiceClients) *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewList(inventoryType),
		Args: graphql.FieldConfigArgument{
			"warehouseId": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return []interface{}{}, nil
		},
	}
}

func getDriverField(clients ServiceClients) *graphql.Field {
	return &graphql.Field{
		Type: driverType,
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Type: graphql.NewNonNull(graphql.String),
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return map[string]interface{}{}, nil
		},
	}
}

func getDriversField(clients ServiceClients) *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewList(driverType),
		Args: graphql.FieldConfigArgument{
			"status": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return []interface{}{}, nil
		},
	}
}

func getRouteField(clients ServiceClients) *graphql.Field {
	return &graphql.Field{
		Type: routeType,
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Type: graphql.NewNonNull(graphql.String),
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return map[string]interface{}{}, nil
		},
	}
}

func getRoutesField(clients ServiceClients) *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewList(routeType),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return []interface{}{}, nil
		},
	}
}

// Mutation resolvers
func createOrderField(clients ServiceClients) *graphql.Field {
	return &graphql.Field{
		Type: orderType,
		Args: graphql.FieldConfigArgument{
			"customerId": &graphql.ArgumentConfig{
				Type: graphql.NewNonNull(graphql.String),
			},
			"items": &graphql.ArgumentConfig{
				Type: graphql.NewNonNull(graphql.NewList(graphql.String)),
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			// Call order service gRPC client to create order
			return map[string]interface{}{}, nil
		},
	}
}

func updateOrderField(clients ServiceClients) *graphql.Field {
	return &graphql.Field{
		Type: orderType,
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Type: graphql.NewNonNull(graphql.String),
			},
			"status": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return map[string]interface{}{}, nil
		},
	}
}

func cancelOrderField(clients ServiceClients) *graphql.Field {
	return &graphql.Field{
		Type: graphql.Boolean,
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Type: graphql.NewNonNull(graphql.String),
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return true, nil
		},
	}
}

func createShipmentField(clients ServiceClients) *graphql.Field {
	return &graphql.Field{
		Type: shipmentType,
		Args: graphql.FieldConfigArgument{
			"orderId": &graphql.ArgumentConfig{
				Type: graphql.NewNonNull(graphql.String),
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return map[string]interface{}{}, nil
		},
	}
}

func updateShipmentField(clients ServiceClients) *graphql.Field {
	return &graphql.Field{
		Type: shipmentType,
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Type: graphql.NewNonNull(graphql.String),
			},
			"status": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return map[string]interface{}{}, nil
		},
	}
}

func updateInventoryField(clients ServiceClients) *graphql.Field {
	return &graphql.Field{
		Type: inventoryType,
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Type: graphql.NewNonNull(graphql.String),
			},
			"quantity": &graphql.ArgumentConfig{
				Type: graphql.Int,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return map[string]interface{}{}, nil
		},
	}
}

func assignDriverField(clients ServiceClients) *graphql.Field {
	return &graphql.Field{
		Type: shipmentType,
		Args: graphql.FieldConfigArgument{
			"shipmentId": &graphql.ArgumentConfig{
				Type: graphql.NewNonNull(graphql.String),
			},
			"driverId": &graphql.ArgumentConfig{
				Type: graphql.NewNonNull(graphql.String),
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return map[string]interface{}{}, nil
		},
	}
}

// Subscription resolvers
func orderUpdatesField(clients ServiceClients) *graphql.Field {
	return &graphql.Field{
		Type: orderType,
		Args: graphql.FieldConfigArgument{
			"orderId": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			// Subscribe to order updates via WebSocket/SSE
			return nil, nil
		},
	}
}

func shipmentUpdatesField(clients ServiceClients) *graphql.Field {
	return &graphql.Field{
		Type: shipmentType,
		Args: graphql.FieldConfigArgument{
			"shipmentId": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return nil, nil
		},
	}
}

func driverLocationField(clients ServiceClients) *graphql.Field {
	return &graphql.Field{
		Type: locationType,
		Args: graphql.FieldConfigArgument{
			"driverId": &graphql.ArgumentConfig{
				Type: graphql.NewNonNull(graphql.String),
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return nil, nil
		},
	}
}

const playgroundHTML = `<!DOCTYPE html>
<html>
<head>
  <meta charset=utf-8/>
  <meta name="viewport" content="user-scalable=no, initial-scale=1.0, minimum-scale=1.0, maximum-scale=1.0, minimal-ui">
  <title>GraphQL Playground</title>
  <link rel="stylesheet" href="//cdn.jsdelivr.net/npm/graphql-playground-react/build/static/css/index.css" />
  <link rel="shortcut icon" href="//cdn.jsdelivr.net/npm/graphql-playground-react/build/favicon.png" />
  <script src="//cdn.jsdelivr.net/npm/graphql-playground-react/build/static/js/middleware.js"></script>
</head>
<body>
  <div id="root">
    <style>
      body {
        background-color: rgb(23, 42, 58);
        font-family: Open Sans, sans-serif;
        height: 90vh;
      }
      #root {
        height: 100%;
        width: 100%;
        display: flex;
        align-items: center;
        justify-content: center;
      }
      .loading {
        font-size: 32px;
        font-weight: 200;
        color: rgba(255, 255, 255, .6);
        margin-left: 20px;
      }
      img {
        width: 78px;
        height: 78px;
      }
      .title {
        font-weight: 400;
      }
    </style>
    <img src='//cdn.jsdelivr.net/npm/graphql-playground-react/build/logo.png' alt=''>
    <div class="loading"> Loading
      <span class="title">GraphQL Playground</span>
    </div>
  </div>
  <script>window.addEventListener('load', function (event) {
      GraphQLPlayground.init(document.getElementById('root'), {
        endpoint: '/graphql'
      })
    })</script>
</body>
</html>`

// Example usage:
//
// // Create GraphQL server
// gqlServer, err := graphql.NewServer(graphql.Config{
//     Playground: true,
//     Pretty:     true,
//     MaxComplexity: 1000,
//     MaxDepth:      10,
//     Timeout:       30 * time.Second,
//     ServiceClients: graphql.ServiceClients{
//         OrderClient:    orderClient,
//         ShipmentClient: shipmentClient,
//     },
//     Auth: func(c *gin.Context) error {
//         // Validate JWT token
//         return nil
//     },
// })
//
// // Setup routes
// router := gin.Default()
// router.POST("/graphql", gqlServer.Handler())
// router.GET("/playground", gqlServer.PlaygroundHandler())
//
// // Example query:
// // {
// //   order(id: "order-123") {
// //     id
// //     customerId
// //     status
// //     totalAmount
// //     shipment {
// //       trackingNumber
// //       status
// //       driver {
// //         name
// //         currentLocation {
// //           latitude
// //           longitude
// //         }
// //       }
// //     }
// //   }
// // }
//
// // Example mutation:
// // mutation {
// //   createOrder(customerId: "customer-123", items: ["item1", "item2"]) {
// //     id
// //     status
// //   }
// // }
