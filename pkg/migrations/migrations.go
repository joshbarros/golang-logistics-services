package migrations

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// MigrationConfig holds migration configuration
type MigrationConfig struct {
	DB            *sql.DB
	MigrationsFS  embed.FS
	MigrationsDir string
	DatabaseName  string
}

// Runner handles database migrations
type Runner struct {
	migrate *migrate.Migrate
}

// NewRunner creates a new migration runner
func NewRunner(config MigrationConfig) (*Runner, error) {
	// Create source from embedded FS
	sourceDriver, err := iofs.New(config.MigrationsFS, config.MigrationsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create migration source: %w", err)
	}

	// Create database driver
	dbDriver, err := postgres.WithInstance(config.DB, &postgres.Config{
		DatabaseName: config.DatabaseName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create database driver: %w", err)
	}

	// Create migrate instance
	m, err := migrate.NewWithInstance("iofs", sourceDriver, config.DatabaseName, dbDriver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return &Runner{migrate: m}, nil
}

// Up applies all pending migrations
func (r *Runner) Up() error {
	if err := r.migrate.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil // No migrations to apply is not an error
		}
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	return nil
}

// Down rolls back one migration
func (r *Runner) Down() error {
	if err := r.migrate.Down(); err != nil {
		return fmt.Errorf("failed to rollback migration: %w", err)
	}
	return nil
}

// Steps runs a specific number of migrations
func (r *Runner) Steps(n int) error {
	if err := r.migrate.Steps(n); err != nil {
		return fmt.Errorf("failed to run %d migration steps: %w", n, err)
	}
	return nil
}

// Force sets the migration version without running migrations
func (r *Runner) Force(version int) error {
	if err := r.migrate.Force(version); err != nil {
		return fmt.Errorf("failed to force migration version: %w", err)
	}
	return nil
}

// Version returns the current migration version
func (r *Runner) Version() (uint, bool, error) {
	version, dirty, err := r.migrate.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			return 0, false, nil // No migrations applied yet
		}
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}
	return version, dirty, nil
}

// Close closes the migration runner
func (r *Runner) Close() error {
	srcErr, dbErr := r.migrate.Close()
	if srcErr != nil {
		return srcErr
	}
	return dbErr
}

// MigrationInfo holds information about migrations
type MigrationInfo struct {
	Version uint
	Dirty   bool
}

// GetMigrationInfo returns current migration information
func (r *Runner) GetMigrationInfo() (*MigrationInfo, error) {
	version, dirty, err := r.Version()
	if err != nil {
		return nil, err
	}

	return &MigrationInfo{
		Version: version,
		Dirty:   dirty,
	}, nil
}
