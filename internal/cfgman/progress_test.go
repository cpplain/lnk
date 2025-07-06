package cfgman

import (
	"testing"
	"time"
)

func TestProgressIndicator(t *testing.T) {
	// Save original verbosity
	originalVerbosity := GetVerbosity()
	defer SetVerbosity(originalVerbosity)

	// Set to normal verbosity
	SetVerbosity(VerbosityNormal)

	t.Run("ShowProgress with quick operation", func(t *testing.T) {
		called := false
		err := ShowProgress("Quick operation", func() error {
			called = true
			return nil
		})
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !called {
			t.Error("Function was not called")
		}
	})

	t.Run("ShowProgress with slow operation", func(t *testing.T) {
		called := false
		err := ShowProgress("Slow operation", func() error {
			time.Sleep(1100 * time.Millisecond)
			called = true
			return nil
		})
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !called {
			t.Error("Function was not called")
		}
	})

	t.Run("Progress updates", func(t *testing.T) {
		progress := NewProgressIndicator("Test operation")
		progress.SetTotal(100)

		// Simulate updates
		for i := 0; i <= 100; i += 10 {
			progress.Update(i)
			time.Sleep(10 * time.Millisecond)
		}
	})
}
