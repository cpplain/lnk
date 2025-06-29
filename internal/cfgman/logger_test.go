package cfgman

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestDefaultLogger(t *testing.T) {
	// Save original stdout and stderr
	originalStdout := os.Stdout
	originalStderr := os.Stderr
	originalDebugEnv := os.Getenv("CFGMAN_DEBUG")

	defer func() {
		os.Stdout = originalStdout
		os.Stderr = originalStderr
		os.Setenv("CFGMAN_DEBUG", originalDebugEnv)
	}()

	t.Run("NewDefaultLogger", func(t *testing.T) {
		// Test without debug
		os.Unsetenv("CFGMAN_DEBUG")
		logger := NewDefaultLogger()
		if dl, ok := logger.(*DefaultLogger); !ok || dl.DebugEnabled {
			t.Error("Debug should be disabled by default")
		}

		// Test with debug
		os.Setenv("CFGMAN_DEBUG", "1")
		logger = NewDefaultLogger()
		if dl, ok := logger.(*DefaultLogger); !ok || !dl.DebugEnabled {
			t.Error("Debug should be enabled when CFGMAN_DEBUG is set")
		}
	})

	t.Run("Info logging", func(t *testing.T) {
		logger := &DefaultLogger{DebugEnabled: false}

		stdout := captureOutput(t, func() {
			logger.Info("test info %s", "message")
		}, &os.Stdout)

		expected := "test info message\n"
		if stdout != expected {
			t.Errorf("Info output = %q, want %q", stdout, expected)
		}
	})

	t.Run("Debug logging disabled", func(t *testing.T) {
		logger := &DefaultLogger{DebugEnabled: false}

		stderr := captureOutput(t, func() {
			logger.Debug("debug message")
		}, &os.Stderr)

		if stderr != "" {
			t.Errorf("Debug should not output when disabled, got %q", stderr)
		}
	})

	t.Run("Debug logging enabled", func(t *testing.T) {
		logger := &DefaultLogger{DebugEnabled: true}

		stderr := captureOutput(t, func() {
			logger.Debug("debug %d", 123)
		}, &os.Stderr)

		expected := "[DEBUG] debug 123\n"
		if stderr != expected {
			t.Errorf("Debug output = %q, want %q", stderr, expected)
		}
	})

	t.Run("Warn logging", func(t *testing.T) {
		logger := &DefaultLogger{DebugEnabled: false}

		// Temporarily disable color for predictable output
		oldNoColor := os.Getenv("NO_COLOR")
		os.Setenv("NO_COLOR", "1")
		defer os.Setenv("NO_COLOR", oldNoColor)

		stderr := captureOutput(t, func() {
			logger.Warn("warning %s", "test")
		}, &os.Stderr)

		if !strings.Contains(stderr, "Warning: warning test") {
			t.Errorf("Warn output doesn't contain expected message: %q", stderr)
		}
	})

	t.Run("Error logging", func(t *testing.T) {
		logger := &DefaultLogger{DebugEnabled: false}

		// Temporarily disable color for predictable output
		oldNoColor := os.Getenv("NO_COLOR")
		os.Setenv("NO_COLOR", "1")
		defer os.Setenv("NO_COLOR", oldNoColor)

		stderr := captureOutput(t, func() {
			logger.Error("error %v", "occurred")
		}, &os.Stderr)

		if !strings.Contains(stderr, "Error: error occurred") {
			t.Errorf("Error output doesn't contain expected message: %q", stderr)
		}
	})
}

func TestSetLogger(t *testing.T) {
	// Save original logger
	originalLogger := log
	defer func() { log = originalLogger }()

	// Create a mock logger
	mock := &mockLogger{}
	SetLogger(mock)

	// Verify it was set
	if log != mock {
		t.Error("SetLogger didn't set the global logger")
	}

	// Test that global log uses the mock
	log.Info("test")
	if !mock.infoCalled {
		t.Error("Global logger didn't use the mock")
	}
}

func TestLoggerInterface(t *testing.T) {
	// Ensure DefaultLogger implements Logger interface
	var _ Logger = (*DefaultLogger)(nil)
	var _ Logger = &DefaultLogger{}

	// Test that NewDefaultLogger returns Logger interface
	logger := NewDefaultLogger()
	if _, ok := logger.(Logger); !ok {
		t.Error("NewDefaultLogger should return Logger interface")
	}
}

// Helper function to capture output
func captureOutput(t *testing.T, fn func(), target **os.File) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	original := *target
	*target = w

	outChan := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outChan <- buf.String()
	}()

	fn()

	w.Close()
	*target = original

	return <-outChan
}

// Mock logger for testing
type mockLogger struct {
	debugCalled bool
	infoCalled  bool
	warnCalled  bool
	errorCalled bool
	lastMessage string
}

func (m *mockLogger) Debug(format string, args ...interface{}) {
	m.debugCalled = true
	m.lastMessage = format
}

func (m *mockLogger) Info(format string, args ...interface{}) {
	m.infoCalled = true
	m.lastMessage = format
}

func (m *mockLogger) Warn(format string, args ...interface{}) {
	m.warnCalled = true
	m.lastMessage = format
}

func (m *mockLogger) Error(format string, args ...interface{}) {
	m.errorCalled = true
	m.lastMessage = format
}
