package graphql

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DataLoader solves the N+1 query problem by batching and caching requests
type DataLoader struct {
	// BatchFunc is the function that fetches a batch of items
	BatchFunc func(ctx context.Context, keys []string) ([]interface{}, error)
	// Wait is the time to wait before executing a batch
	Wait time.Duration
	// MaxBatch is the maximum number of keys per batch
	MaxBatch int
	// Cache stores loaded items
	Cache map[string]interface{}
	// mutex protects the cache
	mutex sync.RWMutex
	// batch holds pending keys
	batch []string
	// batchMutex protects the batch
	batchMutex sync.Mutex
	// results channels for pending loads
	results map[string]chan result
}

type result struct {
	data interface{}
	err  error
}

// NewDataLoader creates a new data loader
func NewDataLoader(batchFunc func(context.Context, []string) ([]interface{}, error)) *DataLoader {
	return &DataLoader{
		BatchFunc: batchFunc,
		Wait:      1 * time.Millisecond,
		MaxBatch:  100,
		Cache:     make(map[string]interface{}),
		batch:     make([]string, 0),
		results:   make(map[string]chan result),
	}
}

// Load loads a single item by key
func (dl *DataLoader) Load(ctx context.Context, key string) (interface{}, error) {
	// Check cache first
	dl.mutex.RLock()
	if cached, ok := dl.Cache[key]; ok {
		dl.mutex.RUnlock()
		return cached, nil
	}
	dl.mutex.RUnlock()

	// Add to batch
	dl.batchMutex.Lock()

	// Create result channel for this key
	if _, exists := dl.results[key]; !exists {
		dl.results[key] = make(chan result, 1)
	}
	resultChan := dl.results[key]

	// Add key to batch
	dl.batch = append(dl.batch, key)
	batchSize := len(dl.batch)

	// If batch is full, execute immediately
	if batchSize >= dl.MaxBatch {
		go dl.executeBatch(ctx)
	} else if batchSize == 1 {
		// Start timer for first item in batch
		go func() {
			time.Sleep(dl.Wait)
			dl.executeBatch(ctx)
		}()
	}

	dl.batchMutex.Unlock()

	// Wait for result
	select {
	case res := <-resultChan:
		return res.data, res.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// LoadMany loads multiple items by keys
func (dl *DataLoader) LoadMany(ctx context.Context, keys []string) ([]interface{}, error) {
	results := make([]interface{}, len(keys))
	errors := make([]error, len(keys))
	var wg sync.WaitGroup

	for i, key := range keys {
		wg.Add(1)
		go func(idx int, k string) {
			defer wg.Done()
			data, err := dl.Load(ctx, k)
			results[idx] = data
			errors[idx] = err
		}(i, key)
	}

	wg.Wait()

	// Check if any errors occurred
	for _, err := range errors {
		if err != nil {
			return results, err
		}
	}

	return results, nil
}

// Prime adds an item to the cache
func (dl *DataLoader) Prime(key string, value interface{}) {
	dl.mutex.Lock()
	defer dl.mutex.Unlock()
	dl.Cache[key] = value
}

// Clear clears the cache
func (dl *DataLoader) Clear(key string) {
	dl.mutex.Lock()
	defer dl.mutex.Unlock()
	delete(dl.Cache, key)
}

// ClearAll clears all cached items
func (dl *DataLoader) ClearAll() {
	dl.mutex.Lock()
	defer dl.mutex.Unlock()
	dl.Cache = make(map[string]interface{})
}

// executeBatch executes the current batch
func (dl *DataLoader) executeBatch(ctx context.Context) {
	dl.batchMutex.Lock()

	if len(dl.batch) == 0 {
		dl.batchMutex.Unlock()
		return
	}

	// Get current batch and reset
	keys := make([]string, len(dl.batch))
	copy(keys, dl.batch)
	dl.batch = make([]string, 0)

	dl.batchMutex.Unlock()

	// Execute batch function
	results, err := dl.BatchFunc(ctx, keys)

	if err != nil {
		// Send error to all waiting goroutines
		for _, key := range keys {
			dl.batchMutex.Lock()
			if ch, exists := dl.results[key]; exists {
				ch <- result{err: err}
				delete(dl.results, key)
			}
			dl.batchMutex.Unlock()
		}
		return
	}

	// Cache and send results
	for i, key := range keys {
		var data interface{}
		if i < len(results) {
			data = results[i]

			// Cache the result
			dl.mutex.Lock()
			dl.Cache[key] = data
			dl.mutex.Unlock()
		}

		// Send result to waiting goroutine
		dl.batchMutex.Lock()
		if ch, exists := dl.results[key]; exists {
			ch <- result{data: data, err: nil}
			delete(dl.results, key)
		}
		dl.batchMutex.Unlock()
	}
}

// DataLoaderContext holds data loaders for all entity types
type DataLoaderContext struct {
	OrderLoader      *DataLoader
	ShipmentLoader   *DataLoader
	InventoryLoader  *DataLoader
	DriverLoader     *DataLoader
	RouteLoader      *DataLoader
}

type contextKey string

const dataLoaderKey contextKey = "dataloader"

// NewDataLoaderContext creates a new data loader context
func NewDataLoaderContext(clients ServiceClients) *DataLoaderContext {
	return &DataLoaderContext{
		OrderLoader: NewDataLoader(func(ctx context.Context, ids []string) ([]interface{}, error) {
			// Batch fetch orders from order service
			// return clients.OrderClient.GetOrdersByIDs(ctx, ids)
			results := make([]interface{}, len(ids))
			for i, id := range ids {
				results[i] = map[string]interface{}{
					"id":         id,
					"customerId": "customer-123",
					"status":     "pending",
				}
			}
			return results, nil
		}),

		ShipmentLoader: NewDataLoader(func(ctx context.Context, ids []string) ([]interface{}, error) {
			// Batch fetch shipments
			results := make([]interface{}, len(ids))
			for i, id := range ids {
				results[i] = map[string]interface{}{
					"id":             id,
					"trackingNumber": fmt.Sprintf("TRACK-%s", id),
					"status":         "in_transit",
				}
			}
			return results, nil
		}),

		InventoryLoader: NewDataLoader(func(ctx context.Context, ids []string) ([]interface{}, error) {
			// Batch fetch inventory items
			results := make([]interface{}, len(ids))
			for i, id := range ids {
				results[i] = map[string]interface{}{
					"id":        id,
					"productId": fmt.Sprintf("product-%s", id),
					"quantity":  100,
				}
			}
			return results, nil
		}),

		DriverLoader: NewDataLoader(func(ctx context.Context, ids []string) ([]interface{}, error) {
			// Batch fetch drivers
			results := make([]interface{}, len(ids))
			for i, id := range ids {
				results[i] = map[string]interface{}{
					"id":     id,
					"name":   fmt.Sprintf("Driver %s", id),
					"status": "available",
				}
			}
			return results, nil
		}),

		RouteLoader: NewDataLoader(func(ctx context.Context, ids []string) ([]interface{}, error) {
			// Batch fetch routes
			results := make([]interface{}, len(ids))
			for i, id := range ids {
				results[i] = map[string]interface{}{
					"id":       id,
					"distance": 10.5,
					"duration": 30,
				}
			}
			return results, nil
		}),
	}
}

// WithDataLoader adds data loader context to the request context
func WithDataLoader(ctx context.Context, loaders *DataLoaderContext) context.Context {
	return context.WithValue(ctx, dataLoaderKey, loaders)
}

// GetDataLoader retrieves data loader context from the request context
func GetDataLoader(ctx context.Context) *DataLoaderContext {
	loaders, ok := ctx.Value(dataLoaderKey).(*DataLoaderContext)
	if !ok {
		return nil
	}
	return loaders
}

// DataLoaderMiddleware creates middleware that adds data loaders to context
func DataLoaderMiddleware(clients ServiceClients) func(context.Context) context.Context {
	return func(ctx context.Context) context.Context {
		loaders := NewDataLoaderContext(clients)
		return WithDataLoader(ctx, loaders)
	}
}

// Example usage with resolvers:
//
// func getOrderField(clients ServiceClients) *graphql.Field {
//     return &graphql.Field{
//         Type: orderType,
//         Args: graphql.FieldConfigArgument{
//             "id": &graphql.ArgumentConfig{
//                 Type: graphql.NewNonNull(graphql.String),
//             },
//         },
//         Resolve: func(p graphql.ResolveParams) (interface{}, error) {
//             id := p.Args["id"].(string)
//
//             // Use data loader to batch requests
//             loaders := GetDataLoader(p.Context)
//             if loaders != nil {
//                 return loaders.OrderLoader.Load(p.Context, id)
//             }
//
//             // Fallback to direct call
//             return clients.OrderClient.GetOrder(p.Context, id)
//         },
//     }
// }
//
// // In the shipment resolver, fetch related order efficiently:
// "order": &graphql.Field{
//     Type: orderType,
//     Resolve: func(p graphql.ResolveParams) (interface{}, error) {
//         shipment := p.Source.(map[string]interface{})
//         orderID := shipment["orderId"].(string)
//
//         // This will be batched with other order requests
//         loaders := GetDataLoader(p.Context)
//         return loaders.OrderLoader.Load(p.Context, orderID)
//     },
// }
//
// Benefits:
// - Solves N+1 query problem
// - Batches multiple requests into single RPC call
// - Caches results within single request
// - Reduces latency and load on backend services
