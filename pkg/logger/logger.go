package logger

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// LogLevel represents the severity of a log message
type LogLevel string

const (
	DEBUG LogLevel = "DEBUG"
	INFO  LogLevel = "INFO"
	WARN  LogLevel = "WARN"
	ERROR LogLevel = "ERROR"
	FATAL LogLevel = "FATAL"
)

// Logger provides structured logging capabilities
type Logger struct {
	serviceName string
	output      io.Writer
	level       LogLevel
}

// Fields represents structured log fields
type Fields map[string]interface{}

// New creates a new logger instance
func New(serviceName string) *Logger {
	return &Logger{
		serviceName: serviceName,
		output:      os.Stdout,
		level:       INFO,
	}
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// SetOutput sets the output writer
func (l *Logger) SetOutput(w io.Writer) {
	l.output = w
}

// Debug logs a debug message
func (l *Logger) Debug(message string, fields Fields) {
	if l.shouldLog(DEBUG) {
		l.log(DEBUG, message, fields)
	}
}

// Info logs an info message
func (l *Logger) Info(message string, fields Fields) {
	if l.shouldLog(INFO) {
		l.log(INFO, message, fields)
	}
}

// Warn logs a warning message
func (l *Logger) Warn(message string, fields Fields) {
	if l.shouldLog(WARN) {
		l.log(WARN, message, fields)
	}
}

// Error logs an error message
func (l *Logger) Error(message string, fields Fields) {
	if l.shouldLog(ERROR) {
		l.log(ERROR, message, fields)
	}
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(message string, fields Fields) {
	l.log(FATAL, message, fields)
	os.Exit(1)
}

// WithFields returns a new logger with additional fields
func (l *Logger) WithFields(fields Fields) *Logger {
	// In production, this would create a child logger with merged fields
	return l
}

// shouldLog determines if a message should be logged based on level
func (l *Logger) shouldLog(level LogLevel) bool {
	levels := map[LogLevel]int{
		DEBUG: 0,
		INFO:  1,
		WARN:  2,
		ERROR: 3,
		FATAL: 4,
	}
	return levels[level] >= levels[l.level]
}

// log writes a structured log message
func (l *Logger) log(level LogLevel, message string, fields Fields) {
	timestamp := time.Now().Format(time.RFC3339)

	// Build log line
	logLine := fmt.Sprintf("[%s] %s service=%s message=\"%s\"",
		timestamp, level, l.serviceName, message)

	// Add fields
	if fields != nil {
		for k, v := range fields {
			logLine += fmt.Sprintf(" %s=%v", k, v)
		}
	}

	fmt.Fprintln(l.output, logLine)
}

// GinMiddleware returns a Gin middleware for request logging
func (l *Logger) GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// Get request ID if available
		requestID := c.GetString("request_id")
		if requestID == "" {
			requestID = "unknown"
		}

		l.Info("Request started", Fields{
			"method":     method,
			"path":       path,
			"request_id": requestID,
			"client_ip":  c.ClientIP(),
		})

		// Process request
		c.Next()

		// Log completion
		duration := time.Since(start)
		statusCode := c.Writer.Status()

		logLevel := INFO
		if statusCode >= 500 {
			logLevel = ERROR
		} else if statusCode >= 400 {
			logLevel = WARN
		}

		fields := Fields{
			"method":      method,
			"path":        path,
			"request_id":  requestID,
			"status":      statusCode,
			"duration_ms": duration.Milliseconds(),
			"client_ip":   c.ClientIP(),
		}

		if len(c.Errors) > 0 {
			fields["errors"] = c.Errors.String()
		}

		switch logLevel {
		case ERROR:
			l.Error("Request completed", fields)
		case WARN:
			l.Warn("Request completed", fields)
		default:
			l.Info("Request completed", fields)
		}
	}
}

// FromEnv returns the log level from environment variable
func FromEnv() LogLevel {
	level := os.Getenv("LOG_LEVEL")
	switch level {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
	default:
		return INFO
	}
}
