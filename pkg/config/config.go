package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// GetEnv retrieves an environment variable or returns default value
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvInt retrieves an integer environment variable or returns default
func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetEnvBool retrieves a boolean environment variable or returns default
func GetEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// GetEnvDuration retrieves a duration environment variable or returns default
func GetEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// LoadDatabaseConfig loads database configuration from environment
func LoadDatabaseConfig(servicePrefix string) DatabaseConfig {
	return DatabaseConfig{
		Host:     GetEnv("DB_HOST", "localhost"),
		Port:     GetEnv("DB_PORT", "5432"),
		User:     GetEnv("DB_USER", "postgres"),
		Password: GetEnv("DB_PASSWORD", "postgres"),
		DBName:   GetEnv("DB_NAME", servicePrefix+"_db"),
		SSLMode:  GetEnv("DB_SSLMODE", "disable"),
	}
}

// LoadServerConfig loads HTTP server configuration from environment
func LoadServerConfig() ServerConfig {
	return ServerConfig{
		Port:            GetEnv("HTTP_PORT", "8080"),
		ReadTimeout:     GetEnvDuration("HTTP_READ_TIMEOUT", 15*time.Second),
		WriteTimeout:    GetEnvDuration("HTTP_WRITE_TIMEOUT", 15*time.Second),
		ShutdownTimeout: GetEnvDuration("HTTP_SHUTDOWN_TIMEOUT", 30*time.Second),
	}
}

// DSN generates a PostgreSQL connection string
func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		c.Host, c.User, c.Password, c.DBName, c.Port, c.SSLMode)
}

// Validate validates database configuration
func (c DatabaseConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("DB_HOST is required")
	}
	if c.User == "" {
		return fmt.Errorf("DB_USER is required")
	}
	if c.Password == "" {
		return fmt.Errorf("DB_PASSWORD is required")
	}
	if c.DBName == "" {
		return fmt.Errorf("DB_NAME is required")
	}
	return nil
}

// RequireEnv requires an environment variable to be set
func RequireEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return value, nil
}

// GetServiceName returns the service name from environment or default
func GetServiceName(defaultName string) string {
	return GetEnv("SERVICE_NAME", defaultName)
}

// GetServiceVersion returns the service version from environment
func GetServiceVersion() string {
	return GetEnv("SERVICE_VERSION", "dev")
}

// GetEnvironment returns the deployment environment
func GetEnvironment() string {
	return GetEnv("ENVIRONMENT", "development")
}

// IsProduction checks if running in production
func IsProduction() bool {
	return GetEnvironment() == "production"
}

// IsDevelopment checks if running in development
func IsDevelopment() bool {
	return GetEnvironment() == "development"
}
