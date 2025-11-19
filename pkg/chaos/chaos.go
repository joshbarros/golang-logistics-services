package chaos

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// Chaos provides chaos engineering functionality for resilience testing

var (
	// ErrChaosInjected is returned when chaos is injected
	ErrChaosInjected = errors.New("chaos injected")
)

// Config holds chaos engineering configuration
type Config struct {
	// Enabled enables chaos injection
	Enabled bool
	// Probability is the chance (0.0-1.0) that chaos will be injected
	Probability float64
	// Experiments list of active experiments
	Experiments []Experiment
}

// Experiment defines a chaos experiment
type Experiment struct {
	// ID uniquely identifies the experiment
	ID string
	// Name is human-readable name
	Name string
	// Description explains the experiment
	Description string
	// Target specifies what to target (service, endpoint, database)
	Target Target
	// Fault defines the fault to inject
	Fault Fault
	// Schedule when to run the experiment
	Schedule Schedule
	// Active indicates if experiment is running
	Active bool
	// Metrics collected during experiment
	Metrics ExperimentMetrics
}

// Target specifies the chaos target
type Target struct {
	// Type: service, endpoint, database, cache, queue
	Type string
	// Name of the target
	Name string
	// Selector for filtering (e.g., "version=v2")
	Selector map[string]string
}

// Fault defines the type of fault to inject
type Fault struct {
	// Type: latency, error, abort, resource_exhaustion, network_partition
	Type string
	// Latency configuration
	Latency *LatencyFault
	// Error configuration
	Error *ErrorFault
	// Abort configuration
	Abort *AbortFault
	// Resource configuration
	Resource *ResourceFault
	// Network configuration
	Network *NetworkFault
}

// LatencyFault injects latency
type LatencyFault struct {
	// Duration of latency
	Duration time.Duration
	// Jitter adds randomness (0-100%)
	Jitter int
}

// ErrorFault injects errors
type ErrorFault struct {
	// StatusCode for HTTP errors
	StatusCode int
	// Message for error
	Message string
}

// AbortFault aborts connections
type AbortFault struct {
	// Percentage of connections to abort
	Percentage float64
}

// ResourceFault exhausts resources
type ResourceFault struct {
	// CPUPercent to consume (0-100)
	CPUPercent int
	// MemoryMB to consume
	MemoryMB int
	// DiskIOPercent to consume
	DiskIOPercent int
}

// NetworkFault simulates network issues
type NetworkFault struct {
	// PacketLoss percentage (0-100)
	PacketLoss int
	// Bandwidth limit in Kbps
	Bandwidth int
	// Partition simulates network partition
	Partition bool
}

// Schedule defines when to run experiment
type Schedule struct {
	// StartTime when experiment starts
	StartTime time.Time
	// EndTime when experiment ends
	EndTime time.Time
	// Duration how long to run (alternative to EndTime)
	Duration time.Duration
	// DaysOfWeek when to run (1=Monday, 7=Sunday)
	DaysOfWeek []int
	// TimeOfDay when to run (e.g., "09:00-17:00")
	TimeOfDay string
}

// ExperimentMetrics tracks experiment results
type ExperimentMetrics struct {
	// StartTime when experiment started
	StartTime time.Time
	// EndTime when experiment ended
	EndTime time.Time
	// TotalRequests during experiment
	TotalRequests int64
	// FailedRequests during experiment
	FailedRequests int64
	// AverageLatency during experiment
	AverageLatency time.Duration
	// ErrorRate percentage
	ErrorRate float64
}

// Manager manages chaos experiments
type Manager struct {
	config      Config
	experiments map[string]*Experiment
	mutex       sync.RWMutex
	active      bool
}

// NewManager creates a new chaos manager
func NewManager(config Config) *Manager {
	return &Manager{
		config:      config,
		experiments: make(map[string]*Experiment),
		active:      config.Enabled,
	}
}

// AddExperiment adds a chaos experiment
func (m *Manager) AddExperiment(exp Experiment) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if exp.ID == "" {
		exp.ID = generateID()
	}

	m.experiments[exp.ID] = &exp
}

// RemoveExperiment removes an experiment
func (m *Manager) RemoveExperiment(id string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.experiments, id)
}

// StartExperiment starts an experiment
func (m *Manager) StartExperiment(id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	exp, exists := m.experiments[id]
	if !exists {
		return errors.New("experiment not found")
	}

	exp.Active = true
	exp.Metrics.StartTime = time.Now()

	fmt.Printf("Started chaos experiment: %s (%s)\n", exp.Name, exp.ID)
	return nil
}

// StopExperiment stops an experiment
func (m *Manager) StopExperiment(id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	exp, exists := m.experiments[id]
	if !exists {
		return errors.New("experiment not found")
	}

	exp.Active = false
	exp.Metrics.EndTime = time.Now()

	fmt.Printf("Stopped chaos experiment: %s (%s)\n", exp.Name, exp.ID)
	return nil
}

// GetExperiment returns an experiment by ID
func (m *Manager) GetExperiment(id string) (*Experiment, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	exp, exists := m.experiments[id]
	if !exists {
		return nil, errors.New("experiment not found")
	}

	return exp, nil
}

// ListExperiments returns all experiments
func (m *Manager) ListExperiments() []Experiment {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	experiments := make([]Experiment, 0, len(m.experiments))
	for _, exp := range m.experiments {
		experiments = append(experiments, *exp)
	}

	return experiments
}

// ShouldInject determines if chaos should be injected
func (m *Manager) ShouldInject(target string) bool {
	if !m.active {
		return false
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Check if any active experiments target this
	for _, exp := range m.experiments {
		if !exp.Active {
			continue
		}

		if exp.Target.Name == target || exp.Target.Name == "*" {
			// Check probability
			if rand.Float64() < m.config.Probability {
				return true
			}
		}
	}

	return false
}

// InjectFault injects a fault for the given target
func (m *Manager) InjectFault(ctx context.Context, target string) error {
	if !m.ShouldInject(target) {
		return nil
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Find active experiment for target
	for _, exp := range m.experiments {
		if !exp.Active {
			continue
		}

		if exp.Target.Name == target || exp.Target.Name == "*" {
			return m.executeFault(ctx, exp.Fault)
		}
	}

	return nil
}

// executeFault executes the fault
func (m *Manager) executeFault(ctx context.Context, fault Fault) error {
	switch fault.Type {
	case "latency":
		if fault.Latency != nil {
			return m.injectLatency(ctx, *fault.Latency)
		}
	case "error":
		if fault.Error != nil {
			return m.injectError(*fault.Error)
		}
	case "abort":
		if fault.Abort != nil {
			return m.injectAbort(*fault.Abort)
		}
	case "resource":
		if fault.Resource != nil {
			return m.injectResourceExhaustion(*fault.Resource)
		}
	}

	return nil
}

// injectLatency injects latency
func (m *Manager) injectLatency(ctx context.Context, fault LatencyFault) error {
	duration := fault.Duration

	// Add jitter
	if fault.Jitter > 0 {
		jitterAmount := time.Duration(rand.Intn(fault.Jitter)) * time.Millisecond
		duration += jitterAmount
	}

	select {
	case <-time.After(duration):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// injectError injects an error
func (m *Manager) injectError(fault ErrorFault) error {
	return fmt.Errorf("%w: %s (status: %d)", ErrChaosInjected, fault.Message, fault.StatusCode)
}

// injectAbort aborts the connection
func (m *Manager) injectAbort(fault AbortFault) error {
	if rand.Float64() < fault.Percentage {
		return ErrChaosInjected
	}
	return nil
}

// injectResourceExhaustion exhausts resources
func (m *Manager) injectResourceExhaustion(fault ResourceFault) error {
	// In production, actually consume resources
	// For now, simulate by introducing delay
	time.Sleep(100 * time.Millisecond)
	return nil
}

// HTTPMiddleware returns middleware that injects chaos
func (m *Manager) HTTPMiddleware(target string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if chaos should be injected
			if err := m.InjectFault(r.Context(), target); err != nil {
				if errors.Is(err, ErrChaosInjected) {
					// Find the experiment to get status code
					m.mutex.RLock()
					for _, exp := range m.experiments {
						if exp.Active && (exp.Target.Name == target || exp.Target.Name == "*") {
							if exp.Fault.Error != nil {
								w.WriteHeader(exp.Fault.Error.StatusCode)
								w.Write([]byte(exp.Fault.Error.Message))
								m.mutex.RUnlock()
								return
							}
						}
					}
					m.mutex.RUnlock()

					// Default error
					http.Error(w, "Chaos injected", http.StatusServiceUnavailable)
					return
				}
			}

			// Continue to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// generateID generates a unique experiment ID
func generateID() string {
	return fmt.Sprintf("exp_%d", time.Now().UnixNano())
}

// Common experiment templates

// NewLatencyExperiment creates a latency injection experiment
func NewLatencyExperiment(target, name string, duration, latency time.Duration) Experiment {
	return Experiment{
		ID:          generateID(),
		Name:        name,
		Description: fmt.Sprintf("Inject %s latency to %s", latency, target),
		Target: Target{
			Type: "service",
			Name: target,
		},
		Fault: Fault{
			Type: "latency",
			Latency: &LatencyFault{
				Duration: latency,
				Jitter:   20, // 20% jitter
			},
		},
		Schedule: Schedule{
			StartTime: time.Now(),
			Duration:  duration,
		},
	}
}

// NewErrorExperiment creates an error injection experiment
func NewErrorExperiment(target, name string, duration time.Duration, statusCode int, message string) Experiment {
	return Experiment{
		ID:          generateID(),
		Name:        name,
		Description: fmt.Sprintf("Inject HTTP %d errors to %s", statusCode, target),
		Target: Target{
			Type: "service",
			Name: target,
		},
		Fault: Fault{
			Type: "error",
			Error: &ErrorFault{
				StatusCode: statusCode,
				Message:    message,
			},
		},
		Schedule: Schedule{
			StartTime: time.Now(),
			Duration:  duration,
		},
	}
}

// NewAbortExperiment creates a connection abort experiment
func NewAbortExperiment(target, name string, duration time.Duration, percentage float64) Experiment {
	return Experiment{
		ID:          generateID(),
		Name:        name,
		Description: fmt.Sprintf("Abort %.0f%% of connections to %s", percentage*100, target),
		Target: Target{
			Type: "service",
			Name: target,
		},
		Fault: Fault{
			Type: "abort",
			Abort: &AbortFault{
				Percentage: percentage,
			},
		},
		Schedule: Schedule{
			StartTime: time.Now(),
			Duration:  duration,
		},
	}
}

// Example usage:
//
// // Create chaos manager
// config := chaos.Config{
//     Enabled:     true,
//     Probability: 0.1, // 10% of requests
// }
// manager := chaos.NewManager(config)
//
// // Add latency experiment
// exp := chaos.NewLatencyExperiment(
//     "order-service",
//     "Test service degradation",
//     5 * time.Minute,
//     2 * time.Second,
// )
// manager.AddExperiment(exp)
// manager.StartExperiment(exp.ID)
//
// // Add error experiment
// errorExp := chaos.NewErrorExperiment(
//     "inventory-service",
//     "Test error handling",
//     5 * time.Minute,
//     503,
//     "Service temporarily unavailable",
// )
// manager.AddExperiment(errorExp)
// manager.StartExperiment(errorExp.ID)
//
// // Use in HTTP server
// http.Handle("/api/orders",
//     manager.HTTPMiddleware("order-service")(ordersHandler),
// )
//
// // Check if chaos should be injected
// if err := manager.InjectFault(ctx, "order-service"); err != nil {
//     log.Printf("Chaos injected: %v", err)
//     return err
// }
//
// // Stop experiment after duration
// time.AfterFunc(5*time.Minute, func() {
//     manager.StopExperiment(exp.ID)
// })
