package cfgman

import (
	"fmt"
	"os"
)

// Logger defines the interface for logging operations
type Logger interface {
	// Debug logs a debug message
	Debug(format string, args ...interface{})
	// Info logs an informational message
	Info(format string, args ...interface{})
	// Warn logs a warning message
	Warn(format string, args ...interface{})
	// Error logs an error message
	Error(format string, args ...interface{})
}

// DefaultLogger provides the default implementation of Logger
type DefaultLogger struct {
	// DebugEnabled controls whether debug messages are printed
	DebugEnabled bool
}

// NewDefaultLogger creates a new DefaultLogger
func NewDefaultLogger() Logger {
	return &DefaultLogger{
		DebugEnabled: os.Getenv("CFGMAN_DEBUG") != "",
	}
}

// Debug logs a debug message
func (d *DefaultLogger) Debug(format string, args ...interface{}) {
	if d.DebugEnabled {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// Info logs an informational message
func (d *DefaultLogger) Info(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

// Warn logs a warning message
func (d *DefaultLogger) Warn(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, Yellow("Warning: "+format)+"\n", args...)
}

// Error logs an error message
func (d *DefaultLogger) Error(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, Red("Error: "+format)+"\n", args...)
}

// Global logger instance
var log Logger = NewDefaultLogger()

// SetLogger sets the global logger
func SetLogger(l Logger) {
	log = l
}
