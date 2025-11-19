// Package shutdown provides graceful shutdown capabilities for services
package shutdown

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

// Manager handles graceful shutdown of service components
type Manager struct {
	logger          *logrus.Logger
	shutdownTimeout time.Duration
	callbacks       []ShutdownCallback
	mu              sync.Mutex
	signalChan      chan os.Signal
}

// ShutdownCallback is a function to be called during shutdown
type ShutdownCallback struct {
	Name     string
	Priority int // Lower number = higher priority (runs first)
	Fn       func(ctx context.Context) error
}

// Config for shutdown manager
type Config struct {
	Logger          *logrus.Logger
	ShutdownTimeout time.Duration // Default: 30 seconds
	Signals         []os.Signal   // Default: SIGINT, SIGTERM
}

// NewManager creates a new shutdown manager
func NewManager(config Config) *Manager {
	if config.Logger == nil {
		config.Logger = logrus.New()
	}

	if config.ShutdownTimeout == 0 {
		config.ShutdownTimeout = 30 * time.Second
	}

	if len(config.Signals) == 0 {
		config.Signals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, config.Signals...)

	return &Manager{
		logger:          config.Logger,
		shutdownTimeout: config.ShutdownTimeout,
		callbacks:       make([]ShutdownCallback, 0),
		signalChan:      signalChan,
	}
}

// Register adds a shutdown callback
func (m *Manager) Register(name string, priority int, fn func(ctx context.Context) error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callbacks = append(m.callbacks, ShutdownCallback{
		Name:     name,
		Priority: priority,
		Fn:       fn,
	})

	m.logger.WithFields(logrus.Fields{
		"component": name,
		"priority":  priority,
	}).Debug("Registered shutdown callback")
}

// Wait blocks until a shutdown signal is received, then executes all callbacks
func (m *Manager) Wait() error {
	// Wait for shutdown signal
	sig := <-m.signalChan
	m.logger.WithField("signal", sig.String()).Info("Received shutdown signal")

	return m.Shutdown()
}

// Shutdown executes all registered callbacks with timeout
func (m *Manager) Shutdown() error {
	m.logger.Info("Starting graceful shutdown...")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), m.shutdownTimeout)
	defer cancel()

	// Sort callbacks by priority
	m.mu.Lock()
	callbacks := make([]ShutdownCallback, len(m.callbacks))
	copy(callbacks, m.callbacks)
	m.mu.Unlock()

	// Simple sort by priority (lower number = higher priority)
	for i := 0; i < len(callbacks); i++ {
		for j := i + 1; j < len(callbacks); j++ {
			if callbacks[j].Priority < callbacks[i].Priority {
				callbacks[i], callbacks[j] = callbacks[j], callbacks[i]
			}
		}
	}

	// Execute callbacks in order
	errors := make([]error, 0)
	for _, cb := range callbacks {
		logger := m.logger.WithField("component", cb.Name)
		logger.Info("Shutting down component...")

		start := time.Now()
		if err := cb.Fn(ctx); err != nil {
			logger.WithError(err).Error("Error during shutdown")
			errors = append(errors, fmt.Errorf("%s: %w", cb.Name, err))
		} else {
			duration := time.Since(start)
			logger.WithField("duration", duration).Info("Component shut down successfully")
		}

		// Check if context is done
		if ctx.Err() != nil {
			m.logger.WithError(ctx.Err()).Warn("Shutdown timeout exceeded")
			break
		}
	}

	if len(errors) > 0 {
		m.logger.WithField("error_count", len(errors)).Error("Shutdown completed with errors")
		return fmt.Errorf("shutdown errors: %v", errors)
	}

	m.logger.Info("Graceful shutdown completed successfully")
	return nil
}

// RegisterHTTPServer registers an HTTP server for graceful shutdown
func (m *Manager) RegisterHTTPServer(name string, server interface {
	Shutdown(ctx context.Context) error
}) {
	m.Register(name, 10, func(ctx context.Context) error {
		m.logger.WithField("server", name).Info("Stopping HTTP server...")
		return server.Shutdown(ctx)
	})
}

// RegisterGRPCServer registers a gRPC server for graceful shutdown
func (m *Manager) RegisterGRPCServer(name string, server interface {
	GracefulStop()
}) {
	m.Register(name, 10, func(ctx context.Context) error {
		m.logger.WithField("server", name).Info("Stopping gRPC server...")

		// GracefulStop in goroutine since it blocks
		stopped := make(chan struct{})
		go func() {
			server.GracefulStop()
			close(stopped)
		}()

		// Wait for stop or timeout
		select {
		case <-stopped:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
}

// RegisterDatabase registers a database connection for graceful shutdown
func (m *Manager) RegisterDatabase(name string, db interface {
	Close() error
}) {
	m.Register(name, 20, func(ctx context.Context) error {
		m.logger.WithField("database", name).Info("Closing database connection...")
		return db.Close()
	})
}

// RegisterMessageBroker registers a message broker for graceful shutdown
func (m *Manager) RegisterMessageBroker(name string, broker interface {
	Close() error
}) {
	m.Register(name, 15, func(ctx context.Context) error {
		m.logger.WithField("broker", name).Info("Closing message broker connection...")
		return broker.Close()
	})
}

// RegisterCache registers a cache connection for graceful shutdown
func (m *Manager) RegisterCache(name string, cache interface {
	Close() error
}) {
	m.Register(name, 25, func(ctx context.Context) error {
		m.logger.WithField("cache", name).Info("Closing cache connection...")
		return cache.Close()
	})
}

// RegisterWorkerPool registers a worker pool for graceful shutdown
func (m *Manager) RegisterWorkerPool(name string, pool interface {
	Stop()
}) {
	m.Register(name, 5, func(ctx context.Context) error {
		m.logger.WithField("pool", name).Info("Stopping worker pool...")
		pool.Stop()
		return nil
	})
}

// RegisterCustom registers a custom shutdown function
func (m *Manager) RegisterCustom(name string, priority int, fn func(ctx context.Context) error) {
	m.Register(name, priority, fn)
}

// Stop sends a shutdown signal programmatically
func (m *Manager) Stop() {
	m.signalChan <- syscall.SIGTERM
}

// Helper function to create a basic shutdown manager with default config
func New(logger *logrus.Logger) *Manager {
	return NewManager(Config{
		Logger:          logger,
		ShutdownTimeout: 30 * time.Second,
	})
}

// Helper function for services that need to wait for background tasks
func WaitForTasks(ctx context.Context, tasks ...func(context.Context) error) error {
	errChan := make(chan error, len(tasks))

	for _, task := range tasks {
		go func(t func(context.Context) error) {
			errChan <- t(ctx)
		}(task)
	}

	// Collect all errors
	errors := make([]error, 0)
	for i := 0; i < len(tasks); i++ {
		if err := <-errChan; err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("task errors: %v", errors)
	}

	return nil
}
