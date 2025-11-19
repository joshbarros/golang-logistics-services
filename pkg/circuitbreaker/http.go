package circuitbreaker

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// HTTPClient wraps http.Client with circuit breaker protection
type HTTPClient struct {
	client         *http.Client
	circuitBreaker *CircuitBreaker
}

// NewHTTPClient creates a new HTTP client with circuit breaker
func NewHTTPClient(name string, config Config, timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
		circuitBreaker: NewCircuitBreaker(name, config),
	}
}

// Do executes an HTTP request with circuit breaker protection
func (c *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	cbErr := c.circuitBreaker.ExecuteWithContext(req.Context(), func(ctx context.Context) error {
		resp, err = c.client.Do(req)
		if err != nil {
			return err
		}

		// Consider 5xx responses as failures
		if resp.StatusCode >= 500 {
			return fmt.Errorf("server error: %d", resp.StatusCode)
		}

		return nil
	})

	if cbErr != nil {
		if cbErr == ErrCircuitOpen {
			return nil, fmt.Errorf("circuit breaker open for %s: %w", req.URL.Host, cbErr)
		}
		return resp, cbErr
	}

	return resp, err
}

// Get performs a GET request with circuit breaker protection
func (c *HTTPClient) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Post performs a POST request with circuit breaker protection
func (c *HTTPClient) Post(url, contentType string, body interface{}) (*http.Response, error) {
	// Implementation similar to http.Client.Post
	// This is a simplified version - full implementation would handle body serialization
	return nil, fmt.Errorf("not implemented - use Do() for POST requests")
}

// State returns the current circuit breaker state
func (c *HTTPClient) State() State {
	return c.circuitBreaker.State()
}

// Counts returns the current circuit breaker counts
func (c *HTTPClient) Counts() Counts {
	return c.circuitBreaker.Counts()
}

// Example usage:
//
// func main() {
//     // Create HTTP client with circuit breaker
//     client := circuitbreaker.NewHTTPClient(
//         "external-api",
//         circuitbreaker.Config{
//             MaxRequests: 3,
//             Interval:    60 * time.Second,
//             Timeout:     30 * time.Second,
//             ReadyToTrip: func(counts circuitbreaker.Counts) bool {
//                 failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
//                 return counts.Requests >= 10 && failureRatio >= 0.5
//             },
//             OnStateChange: func(name string, from, to circuitbreaker.State) {
//                 log.Printf("Circuit breaker %s changed from %s to %s", name, from, to)
//             },
//         },
//         10*time.Second,
//     )
//
//     // Use the client
//     resp, err := client.Get("https://api.example.com/endpoint")
//     if err != nil {
//         if errors.Is(err, circuitbreaker.ErrCircuitOpen) {
//             // Handle circuit open - maybe use cached data
//             log.Println("Circuit is open, using fallback")
//         } else {
//             log.Printf("Request failed: %v", err)
//         }
//         return
//     }
//     defer resp.Body.Close()
// }
