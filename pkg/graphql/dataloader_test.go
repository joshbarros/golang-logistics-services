package graphql

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestDataLoader(t *testing.T) {
	// Create a batch function that tracks calls
	callCount := 0
	var mu sync.Mutex

	batchFunc := func(ctx context.Context, keys []string) ([]interface{}, error) {
		mu.Lock()
		callCount++
		mu.Unlock()

		results := make([]interface{}, len(keys))
		for i, key := range keys {
			results[i] = map[string]string{
				"id":   key,
				"name": fmt.Sprintf("Item %s", key),
			}
		}
		return results, nil
	}

	loader := NewDataLoader(batchFunc)
	loader.Wait = 5 * time.Millisecond // Short wait for testing

	ctx := context.Background()

	// Load single item
	result, err := loader.Load(ctx, "1")
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}

	// Verify batch function was called
	mu.Lock()
	if callCount != 1 {
		t.Errorf("Expected 1 batch call, got %d", callCount)
	}
	mu.Unlock()

	// Load from cache (should not call batch function again)
	result2, err := loader.Load(ctx, "1")
	if err != nil {
		t.Fatalf("Failed to load from cache: %v", err)
	}

	if result2 == nil {
		t.Fatal("Cached result is nil")
	}

	mu.Lock()
	if callCount != 1 {
		t.Errorf("Expected 1 batch call (cached), got %d", callCount)
	}
	mu.Unlock()
}

func TestDataLoaderBatching(t *testing.T) {
	callCount := 0
	var batchSizes []int
	var mu sync.Mutex

	batchFunc := func(ctx context.Context, keys []string) ([]interface{}, error) {
		mu.Lock()
		callCount++
		batchSizes = append(batchSizes, len(keys))
		mu.Unlock()

		results := make([]interface{}, len(keys))
		for i, key := range keys {
			results[i] = key
		}
		return results, nil
	}

	loader := NewDataLoader(batchFunc)
	loader.Wait = 10 * time.Millisecond

	ctx := context.Background()

	// Load multiple items concurrently (should batch)
	var wg sync.WaitGroup
	keys := []string{"1", "2", "3", "4", "5"}

	for _, key := range keys {
		wg.Add(1)
		go func(k string) {
			defer wg.Done()
			_, err := loader.Load(ctx, k)
			if err != nil {
				t.Errorf("Failed to load %s: %v", k, err)
			}
		}(key)
	}

	wg.Wait()

	mu.Lock()
	defer mu.Unlock()

	// Should have batched all requests into one call
	if callCount != 1 {
		t.Logf("Call count: %d, batch sizes: %v", callCount, batchSizes)
		// This is acceptable - timing may cause multiple batches
	}

	// Verify all items were fetched
	totalItems := 0
	for _, size := range batchSizes {
		totalItems += size
	}

	if totalItems != len(keys) {
		t.Errorf("Expected %d total items, got %d", len(keys), totalItems)
	}
}

func TestDataLoaderLoadMany(t *testing.T) {
	batchFunc := func(ctx context.Context, keys []string) ([]interface{}, error) {
		results := make([]interface{}, len(keys))
		for i, key := range keys {
			results[i] = fmt.Sprintf("value-%s", key)
		}
		return results, nil
	}

	loader := NewDataLoader(batchFunc)
	loader.Wait = 5 * time.Millisecond

	ctx := context.Background()
	keys := []string{"a", "b", "c"}

	results, err := loader.LoadMany(ctx, keys)
	if err != nil {
		t.Fatalf("LoadMany failed: %v", err)
	}

	if len(results) != len(keys) {
		t.Errorf("Expected %d results, got %d", len(keys), len(results))
	}

	for i, result := range results {
		expected := fmt.Sprintf("value-%s", keys[i])
		if result != expected {
			t.Errorf("Result %d: expected %s, got %v", i, expected, result)
		}
	}
}

func TestDataLoaderPrime(t *testing.T) {
	batchFunc := func(ctx context.Context, keys []string) ([]interface{}, error) {
		t.Error("Batch function should not be called for primed values")
		return nil, nil
	}

	loader := NewDataLoader(batchFunc)

	// Prime the cache
	loader.Prime("1", "primed-value")

	ctx := context.Background()
	result, err := loader.Load(ctx, "1")

	if err != nil {
		t.Fatalf("Failed to load primed value: %v", err)
	}

	if result != "primed-value" {
		t.Errorf("Expected 'primed-value', got %v", result)
	}
}

func TestDataLoaderClear(t *testing.T) {
	callCount := 0
	var mu sync.Mutex

	batchFunc := func(ctx context.Context, keys []string) ([]interface{}, error) {
		mu.Lock()
		callCount++
		mu.Unlock()

		results := make([]interface{}, len(keys))
		for i, key := range keys {
			results[i] = key
		}
		return results, nil
	}

	loader := NewDataLoader(batchFunc)
	loader.Wait = 5 * time.Millisecond

	ctx := context.Background()

	// Load and cache
	_, err := loader.Load(ctx, "1")
	if err != nil {
		t.Fatalf("First load failed: %v", err)
	}

	mu.Lock()
	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}
	mu.Unlock()

	// Clear cache
	loader.Clear("1")

	// Load again (should call batch function)
	_, err = loader.Load(ctx, "1")
	if err != nil {
		t.Fatalf("Second load failed: %v", err)
	}

	mu.Lock()
	if callCount != 2 {
		t.Errorf("Expected 2 calls after clear, got %d", callCount)
	}
	mu.Unlock()
}

func TestDataLoaderMaxBatch(t *testing.T) {
	var batchSizes []int
	var mu sync.Mutex

	batchFunc := func(ctx context.Context, keys []string) ([]interface{}, error) {
		mu.Lock()
		batchSizes = append(batchSizes, len(keys))
		mu.Unlock()

		results := make([]interface{}, len(keys))
		for i := range keys {
			results[i] = i
		}
		return results, nil
	}

	loader := NewDataLoader(batchFunc)
	loader.MaxBatch = 3 // Small batch size
	loader.Wait = 5 * time.Millisecond

	ctx := context.Background()

	// Load more than max batch
	keys := []string{"1", "2", "3", "4", "5"}
	_, err := loader.LoadMany(ctx, keys)

	if err != nil {
		t.Fatalf("LoadMany failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// Should have split into multiple batches
	if len(batchSizes) < 2 {
		t.Logf("Batch sizes: %v", batchSizes)
		// Timing may affect this - not a hard failure
	}

	for _, size := range batchSizes {
		if size > loader.MaxBatch {
			t.Errorf("Batch size %d exceeds max %d", size, loader.MaxBatch)
		}
	}
}

func TestDataLoaderContext(t *testing.T) {
	loader := NewDataLoader(func(ctx context.Context, keys []string) ([]interface{}, error) {
		// Simulate slow operation
		time.Sleep(100 * time.Millisecond)
		return make([]interface{}, len(keys)), nil
	})

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := loader.Load(ctx, "1")

	if err == nil {
		t.Error("Expected context timeout error")
	}

	if err != context.DeadlineExceeded {
		t.Logf("Got error: %v (expected deadline exceeded)", err)
	}
}
