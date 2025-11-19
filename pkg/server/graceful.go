package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// GracefulShutdownConfig defines configuration for graceful shutdown
type GracefulShutdownConfig struct {
	// Timeout for shutdown operations
	Timeout time.Duration
	// Signals to listen for (default: SIGINT, SIGTERM)
	Signals []os.Signal
}

// DefaultShutdownConfig returns default graceful shutdown configuration
func DefaultShutdownConfig() GracefulShutdownConfig {
	return GracefulShutdownConfig{
		Timeout: 30 * time.Second,
		Signals: []os.Signal{syscall.SIGINT, syscall.SIGTERM},
	}
}

// RunWithGracefulShutdown runs an HTTP server with graceful shutdown
func RunWithGracefulShutdown(srv *http.Server, config GracefulShutdownConfig, onShutdown func()) error {
	// Create a channel to listen for interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, config.Signals...)

	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("Server starting on %s", srv.Addr)
		serverErrors <- srv.ListenAndServe()
	}()

	// Wait for interrupt signal or server error
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-quit:
		log.Printf("Received shutdown signal: %v", sig)

		// Create a deadline for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
		defer cancel()

		// Shutdown the server
		log.Println("Shutting down server gracefully...")

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Error during server shutdown: %v", err)
			return fmt.Errorf("server shutdown error: %w", err)
		}

		// Call custom shutdown handler
		if onShutdown != nil {
			log.Println("Running cleanup tasks...")
			onShutdown()
		}

		log.Println("Server stopped gracefully")
	}

	return nil
}

// ShutdownHook represents a function to be called during shutdown
type ShutdownHook func() error

// ShutdownManager manages multiple shutdown hooks
type ShutdownManager struct {
	hooks []ShutdownHook
}

// NewShutdownManager creates a new shutdown manager
func NewShutdownManager() *ShutdownManager {
	return &ShutdownManager{
		hooks: make([]ShutdownHook, 0),
	}
}

// AddHook adds a shutdown hook
func (sm *ShutdownManager) AddHook(hook ShutdownHook) {
	sm.hooks = append(sm.hooks, hook)
}

// Shutdown executes all registered hooks
func (sm *ShutdownManager) Shutdown() {
	log.Printf("Executing %d shutdown hooks...", len(sm.hooks))

	for i, hook := range sm.hooks {
		if err := hook(); err != nil {
			log.Printf("Shutdown hook %d failed: %v", i, err)
		}
	}

	log.Println("All shutdown hooks completed")
}
