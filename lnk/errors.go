package lnk

import (
	"errors"
	"fmt"
)

// Common errors
var (
	// ErrNotSymlink indicates that the path is not a symlink
	ErrNotSymlink = errors.New("not a symlink")

	// ErrAlreadyAdopted indicates that a file is already adopted
	ErrAlreadyAdopted = errors.New("file already adopted")
)

// PathError represents an error related to a specific path
type PathError struct {
	Op   string // Operation being performed
	Path string // Path that caused the error
	Err  error  // Underlying error
	Hint string // Optional hint for resolving the error
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
	Hint   string // Optional hint for resolving the error
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
	Hint    string // Optional hint for resolving the error
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

// NewPathErrorWithHint creates a new PathError with a hint
func NewPathErrorWithHint(op, path string, err error, hint string) error {
	return &PathError{Op: op, Path: path, Err: err, Hint: hint}
}

// NewLinkErrorWithHint creates a new LinkError with a hint
func NewLinkErrorWithHint(op, source, target string, err error, hint string) error {
	return &LinkError{Op: op, Source: source, Target: target, Err: err, Hint: hint}
}

// HintedError wraps an error with an actionable hint
type HintedError struct {
	Err  error
	Hint string
}

func (e *HintedError) Error() string {
	return e.Err.Error()
}

func (e *HintedError) Unwrap() error {
	return e.Err
}

// WithHint wraps an error with an actionable hint for the user
func WithHint(err error, hint string) error {
	if err == nil {
		return nil
	}
	return &HintedError{Err: err, Hint: hint}
}

// GetHint returns the hint for HintedError
func (e *HintedError) GetHint() string {
	return e.Hint
}

// NewValidationErrorWithHint creates a new ValidationError with a hint
func NewValidationErrorWithHint(field, value, message, hint string) error {
	return &ValidationError{Field: field, Value: value, Message: message, Hint: hint}
}

// HintableError is an interface for errors that can provide hints
type HintableError interface {
	error
	GetHint() string
}

// GetHint returns the hint for PathError
func (e *PathError) GetHint() string {
	return e.Hint
}

// GetHint returns the hint for LinkError
func (e *LinkError) GetHint() string {
	return e.Hint
}

// GetHint returns the hint for ValidationError
func (e *ValidationError) GetHint() string {
	return e.Hint
}

// GetErrorHint extracts a hint from an error if it implements HintableError
func GetErrorHint(err error) string {
	var hintableErr HintableError
	if errors.As(err, &hintableErr) {
		return hintableErr.GetHint()
	}
	return ""
}
