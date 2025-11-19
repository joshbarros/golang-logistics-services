package retry

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"
)

var (
	// ErrMaxAttemptsReached is returned when max retry attempts are exhausted
	ErrMaxAttemptsReached = errors.New("max retry attempts reached")
)

// Config holds retry configuration
type Config struct {
	// MaxAttempts is the maximum number of retry attempts
	MaxAttempts int
	// InitialDelay is the delay before the first retry
	InitialDelay time.Duration
	// MaxDelay is the maximum delay between retries
	MaxDelay time.Duration
	// Multiplier for exponential backoff (typically 2.0)
	Multiplier float64
	// Jitter adds randomness to delay to prevent thundering herd
	Jitter bool
	// RetryableErrors is a list of errors that should trigger a retry
	RetryableErrors []error
	// OnRetry is called before each retry attempt
	OnRetry func(attempt int, err error, delay time.Duration)
}

// DefaultConfig returns a default retry configuration
func DefaultConfig() Config {
	return Config{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
	}
}

// Do executes the function with retry logic
func Do(ctx context.Context, config Config, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		// Execute function
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryable(err, config.RetryableErrors) {
			return err
		}

		// Don't retry on last attempt
		if attempt == config.MaxAttempts-1 {
			break
		}

		// Calculate delay
		delay := calculateDelay(attempt, config)

		// Call retry callback
		if config.OnRetry != nil {
			config.OnRetry(attempt+1, err, delay)
		}

		// Wait before retry
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return fmt.Errorf("%w: %v", ErrMaxAttemptsReached, lastErr)
}

// DoWithValue executes the function with retry logic and returns a value
func DoWithValue[T any](ctx context.Context, config Config, fn func() (T, error)) (T, error) {
	var result T
	var lastErr error

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		// Execute function
		res, err := fn()
		if err == nil {
			return res, nil
		}

		result = res
		lastErr = err

		// Check if error is retryable
		if !isRetryable(err, config.RetryableErrors) {
			return result, err
		}

		// Don't retry on last attempt
		if attempt == config.MaxAttempts-1 {
			break
		}

		// Calculate delay
		delay := calculateDelay(attempt, config)

		// Call retry callback
		if config.OnRetry != nil {
			config.OnRetry(attempt+1, err, delay)
		}

		// Wait before retry
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(delay):
		}
	}

	return result, fmt.Errorf("%w: %v", ErrMaxAttemptsReached, lastErr)
}

// calculateDelay calculates the delay before next retry using exponential backoff
func calculateDelay(attempt int, config Config) time.Duration {
	// Exponential backoff: initialDelay * multiplier^attempt
	delay := float64(config.InitialDelay) * math.Pow(config.Multiplier, float64(attempt))

	// Cap at max delay
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}

	// Add jitter
	if config.Jitter {
		jitter := rand.Float64() * delay * 0.1 // 10% jitter
		delay = delay + jitter
	}

	return time.Duration(delay)
}

// isRetryable checks if an error should trigger a retry
func isRetryable(err error, retryableErrors []error) bool {
	if len(retryableErrors) == 0 {
		// No specific errors specified, retry all errors
		return true
	}

	for _, retryable := range retryableErrors {
		if errors.Is(err, retryable) {
			return true
		}
	}

	return false
}

// Retrier provides a reusable retry mechanism
type Retrier struct {
	config Config
}

// NewRetrier creates a new Retrier
func NewRetrier(config Config) *Retrier {
	return &Retrier{config: config}
}

// Do executes the function with retry logic
func (r *Retrier) Do(ctx context.Context, fn func() error) error {
	return Do(ctx, r.config, fn)
}

// DoWithValue executes the function with retry logic and returns a value
func (r *Retrier) DoWithValue[T any](ctx context.Context, fn func() (T, error)) (T, error) {
	return DoWithValue(ctx, r.config, fn)
}

// Common retryable errors
var (
	ErrTemporaryFailure = errors.New("temporary failure")
	ErrTimeout          = errors.New("timeout")
	ErrConnectionFailed = errors.New("connection failed")
)

// Example usage:
//
// func main() {
//     ctx := context.Background()
//
//     // Simple retry
//     err := retry.Do(ctx, retry.Config{
//         MaxAttempts:  5,
//         InitialDelay: 100 * time.Millisecond,
//         MaxDelay:     5 * time.Second,
//         Multiplier:   2.0,
//         Jitter:       true,
//         OnRetry: func(attempt int, err error, delay time.Duration) {
//             log.Printf("Retry attempt %d after error: %v (waiting %v)", attempt, err, delay)
//         },
//     }, func() error {
//         // Your operation here
//         return apiClient.Call()
//     })
//
//     // Retry with value
//     result, err := retry.DoWithValue(ctx, retry.DefaultConfig(), func() (*Order, error) {
//         return orderRepo.GetByID("order-123")
//     })
//
//     // Reusable retrier
//     retrier := retry.NewRetrier(retry.Config{
//         MaxAttempts:     3,
//         InitialDelay:    50 * time.Millisecond,
//         MaxDelay:        1 * time.Second,
//         Multiplier:      2.0,
//         Jitter:          true,
//         RetryableErrors: []error{retry.ErrTemporaryFailure, retry.ErrTimeout},
//     })
//
//     err = retrier.Do(ctx, func() error {
//         return database.Query()
//     })
// }
