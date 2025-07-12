package lnk

import (
	"errors"
	"testing"
)

func TestPathError(t *testing.T) {
	tests := []struct {
		name     string
		err      *PathError
		expected string
	}{
		{
			name: "path error with wrapped error",
			err: &PathError{
				Op:   "open",
				Path: "/path/to/file",
				Err:  errors.New("permission denied"),
			},
			expected: "open /path/to/file: permission denied",
		},
		{
			name: "path error with nil wrapped error",
			err: &PathError{
				Op:   "stat",
				Path: "/some/path",
				Err:  nil,
			},
			expected: "stat /some/path: <nil>",
		},
		{
			name: "path error with standard error",
			err: &PathError{
				Op:   "remove",
				Path: "/tmp/test",
				Err:  ErrNotSymlink,
			},
			expected: "remove /tmp/test: not a symlink",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("PathError.Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPathErrorUnwrap(t *testing.T) {
	baseErr := errors.New("base error")
	pathErr := &PathError{
		Op:   "test",
		Path: "/test",
		Err:  baseErr,
	}

	unwrapped := pathErr.Unwrap()
	if unwrapped != baseErr {
		t.Errorf("PathError.Unwrap() = %v, want %v", unwrapped, baseErr)
	}

	// Test with nil error
	pathErr.Err = nil
	if unwrapped := pathErr.Unwrap(); unwrapped != nil {
		t.Errorf("PathError.Unwrap() with nil = %v, want nil", unwrapped)
	}
}

func TestLinkError(t *testing.T) {
	tests := []struct {
		name     string
		err      *LinkError
		expected string
	}{
		{
			name: "link error with source and target",
			err: &LinkError{
				Op:     "symlink",
				Source: "/source/file",
				Target: "/target/file",
				Err:    errors.New("file exists"),
			},
			expected: "symlink /source/file -> /target/file: file exists",
		},
		{
			name: "link error with source only",
			err: &LinkError{
				Op:     "readlink",
				Source: "/some/link",
				Target: "",
				Err:    errors.New("not a symlink"),
			},
			expected: "readlink /some/link: not a symlink",
		},
		{
			name: "link error with standard error",
			err: &LinkError{
				Op:     "create",
				Source: "/src",
				Target: "/dst",
				Err:    ErrAlreadyAdopted,
			},
			expected: "create /src -> /dst: file already adopted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("LinkError.Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestLinkErrorUnwrap(t *testing.T) {
	baseErr := errors.New("base error")
	linkErr := &LinkError{
		Op:     "test",
		Source: "/src",
		Target: "/dst",
		Err:    baseErr,
	}

	unwrapped := linkErr.Unwrap()
	if unwrapped != baseErr {
		t.Errorf("LinkError.Unwrap() = %v, want %v", unwrapped, baseErr)
	}
}

func TestValidationError(t *testing.T) {
	tests := []struct {
		name     string
		err      *ValidationError
		expected string
	}{
		{
			name: "validation error with value",
			err: &ValidationError{
				Field:   "source",
				Value:   "/invalid/path",
				Message: "path does not exist",
			},
			expected: "invalid source '/invalid/path': path does not exist",
		},
		{
			name: "validation error without value",
			err: &ValidationError{
				Field:   "target",
				Value:   "",
				Message: "target cannot be empty",
			},
			expected: "invalid target: target cannot be empty",
		},
		{
			name: "validation error for config field",
			err: &ValidationError{
				Field:   "LinkMappings",
				Value:   "",
				Message: "at least one mapping is required",
			},
			expected: "invalid LinkMappings: at least one mapping is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("ValidationError.Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestErrorHelpers(t *testing.T) {
	t.Run("NewPathError", func(t *testing.T) {
		err := errors.New("test error")
		pathErr := NewPathError("test-op", "/test/path", err)

		pe, ok := pathErr.(*PathError)
		if !ok {
			t.Fatal("NewPathError should return *PathError")
		}

		if pe.Op != "test-op" {
			t.Errorf("Op = %q, want %q", pe.Op, "test-op")
		}
		if pe.Path != "/test/path" {
			t.Errorf("Path = %q, want %q", pe.Path, "/test/path")
		}
		if pe.Err != err {
			t.Errorf("Err = %v, want %v", pe.Err, err)
		}
	})

	t.Run("NewLinkError", func(t *testing.T) {
		err := errors.New("test error")
		linkErr := NewLinkError("link-op", "/src", "/dst", err)

		le, ok := linkErr.(*LinkError)
		if !ok {
			t.Fatal("NewLinkError should return *LinkError")
		}

		if le.Op != "link-op" {
			t.Errorf("Op = %q, want %q", le.Op, "link-op")
		}
		if le.Source != "/src" {
			t.Errorf("Source = %q, want %q", le.Source, "/src")
		}
		if le.Target != "/dst" {
			t.Errorf("Target = %q, want %q", le.Target, "/dst")
		}
		if le.Err != err {
			t.Errorf("Err = %v, want %v", le.Err, err)
		}
	})

	t.Run("NewValidationError", func(t *testing.T) {
		valErr := NewValidationError("field", "value", "message")

		ve, ok := valErr.(*ValidationError)
		if !ok {
			t.Fatal("NewValidationError should return *ValidationError")
		}

		if ve.Field != "field" {
			t.Errorf("Field = %q, want %q", ve.Field, "field")
		}
		if ve.Value != "value" {
			t.Errorf("Value = %q, want %q", ve.Value, "value")
		}
		if ve.Message != "message" {
			t.Errorf("Message = %q, want %q", ve.Message, "message")
		}
	})
}

func TestStandardErrors(t *testing.T) {
	// Test that standard errors have expected messages
	tests := []struct {
		err      error
		expected string
	}{
		{ErrConfigNotFound, "configuration file not found"},
		{ErrInvalidConfig, "invalid configuration"},
		{ErrNoLinkMappings, "no link mappings defined"},
		{ErrNotSymlink, "not a symlink"},
		{ErrAlreadyAdopted, "file already adopted"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error message = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestErrorWrapping(t *testing.T) {
	// Test error wrapping with errors.Is
	baseErr := ErrNotSymlink
	pathErr := NewPathError("check", "/path", baseErr)

	if !errors.Is(pathErr, ErrNotSymlink) {
		t.Error("errors.Is should find wrapped ErrNotSymlink")
	}

	// Test with custom error
	customErr := errors.New("custom")
	linkErr := NewLinkError("link", "/a", "/b", customErr)

	if !errors.Is(linkErr, customErr) {
		t.Error("errors.Is should find wrapped custom error")
	}
}
