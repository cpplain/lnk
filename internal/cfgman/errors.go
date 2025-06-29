package cfgman

import (
	"errors"
	"fmt"
)

// Common errors
var (
	// ErrConfigNotFound indicates that the configuration file does not exist
	ErrConfigNotFound = errors.New("configuration file not found")

	// ErrInvalidConfig indicates that the configuration file is malformed
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrNoLinkMappings indicates that no link mappings are defined
	ErrNoLinkMappings = errors.New("no link mappings defined")

	// ErrNotSymlink indicates that the path is not a symlink
	ErrNotSymlink = errors.New("not a symlink")

	// ErrAlreadyAdopted indicates that a file is already adopted
	ErrAlreadyAdopted = errors.New("file already adopted")

	// ErrOperationCancelled indicates that the user cancelled the operation
	ErrOperationCancelled = errors.New("operation cancelled")
)

// PathError represents an error related to a specific path
type PathError struct {
	Op   string // Operation being performed
	Path string // Path that caused the error
	Err  error  // Underlying error
}

func (e *PathError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("%s %s: <nil>", e.Op, e.Path)
	}
	return fmt.Sprintf("%s %s: %v", e.Op, e.Path, e.Err)
}

func (e *PathError) Unwrap() error {
	return e.Err
}

// LinkError represents an error related to symlink operations
type LinkError struct {
	Op     string // Operation being performed
	Source string // Source path
	Target string // Target path
	Err    error  // Underlying error
}

func (e *LinkError) Error() string {
	if e.Target == "" {
		return fmt.Sprintf("%s %s: %v", e.Op, e.Source, e.Err)
	}
	return fmt.Sprintf("%s %s -> %s: %v", e.Op, e.Source, e.Target, e.Err)
}

func (e *LinkError) Unwrap() error {
	return e.Err
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string // Field that failed validation
	Value   string // Invalid value
	Message string // Error message
}

func (e *ValidationError) Error() string {
	if e.Value != "" {
		return fmt.Sprintf("invalid %s '%s': %s", e.Field, e.Value, e.Message)
	}
	return fmt.Sprintf("invalid %s: %s", e.Field, e.Message)
}

// Helper functions for creating errors

// NewPathError creates a new PathError
func NewPathError(op, path string, err error) error {
	return &PathError{Op: op, Path: path, Err: err}
}

// NewLinkError creates a new LinkError
func NewLinkError(op, source, target string, err error) error {
	return &LinkError{Op: op, Source: source, Target: target, Err: err}
}

// NewValidationError creates a new ValidationError
func NewValidationError(field, value, message string) error {
	return &ValidationError{Field: field, Value: value, Message: message}
}
