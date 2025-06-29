package cfgman

import (
	"errors"
	"os"
	"testing"
)

func TestDefaultUserInput(t *testing.T) {
	// Test that DefaultUserInput implements UserInput interface
	var _ UserInput = (*DefaultUserInput)(nil)
	var _ UserInput = &DefaultUserInput{}

	// Test NewDefaultUserInput
	input := NewDefaultUserInput()
	if _, ok := input.(UserInput); !ok {
		t.Error("NewDefaultUserInput should return UserInput interface")
	}
}

func TestDefaultUserInputMethods(t *testing.T) {
	// Since the actual methods depend on stdin, we'll just verify they exist
	// and can be called without panicking when stdin is redirected

	input := &DefaultUserInput{}

	t.Run("ReadInput", func(t *testing.T) {
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()

		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		os.Stdin = r

		go func() {
			defer w.Close()
			w.Write([]byte("test input\n"))
		}()

		result, err := input.ReadInput("Enter value: ")
		if err != nil {
			t.Errorf("ReadInput failed: %v", err)
		}
		if result != "test input" {
			t.Errorf("ReadInput = %q, want %q", result, "test input")
		}
	})

	t.Run("Confirm", func(t *testing.T) {
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()

		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		os.Stdin = r

		go func() {
			defer w.Close()
			w.Write([]byte("y\n"))
		}()

		result := input.Confirm("Continue?")
		if !result {
			t.Error("Confirm should return true for 'y' input")
		}
	})

	t.Run("ReadInputWithDefault", func(t *testing.T) {
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()

		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		os.Stdin = r

		go func() {
			defer w.Close()
			w.Write([]byte("\n")) // empty input to use default
		}()

		result, err := input.ReadInputWithDefault("Enter value", "default")
		if err != nil {
			t.Errorf("ReadInputWithDefault failed: %v", err)
		}
		if result != "default" {
			t.Errorf("ReadInputWithDefault = %q, want %q", result, "default")
		}
	})
}

// MockUserInput for testing
type MockUserInput struct {
	ReadInputFunc            func(prompt string) (string, error)
	ConfirmFunc              func(prompt string) bool
	ReadInputWithDefaultFunc func(prompt string, defaultValue string) (string, error)

	// Track calls
	ReadInputCalls            []string
	ConfirmCalls              []string
	ReadInputWithDefaultCalls []struct{ prompt, defaultValue string }
}

func (m *MockUserInput) ReadInput(prompt string) (string, error) {
	m.ReadInputCalls = append(m.ReadInputCalls, prompt)
	if m.ReadInputFunc != nil {
		return m.ReadInputFunc(prompt)
	}
	return "", nil
}

func (m *MockUserInput) Confirm(prompt string) bool {
	m.ConfirmCalls = append(m.ConfirmCalls, prompt)
	if m.ConfirmFunc != nil {
		return m.ConfirmFunc(prompt)
	}
	return false
}

func (m *MockUserInput) ReadInputWithDefault(prompt string, defaultValue string) (string, error) {
	m.ReadInputWithDefaultCalls = append(m.ReadInputWithDefaultCalls,
		struct{ prompt, defaultValue string }{prompt, defaultValue})
	if m.ReadInputWithDefaultFunc != nil {
		return m.ReadInputWithDefaultFunc(prompt, defaultValue)
	}
	return defaultValue, nil
}

func TestMockUserInput(t *testing.T) {
	// Verify MockUserInput implements UserInput interface
	var _ UserInput = (*MockUserInput)(nil)

	t.Run("ReadInput", func(t *testing.T) {
		mock := &MockUserInput{
			ReadInputFunc: func(prompt string) (string, error) {
				if prompt == "Name: " {
					return "test name", nil
				}
				return "", errors.New("unexpected prompt")
			},
		}

		result, err := mock.ReadInput("Name: ")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result != "test name" {
			t.Errorf("Result = %q, want %q", result, "test name")
		}

		if len(mock.ReadInputCalls) != 1 || mock.ReadInputCalls[0] != "Name: " {
			t.Errorf("ReadInputCalls = %v, want [\"Name: \"]", mock.ReadInputCalls)
		}
	})

	t.Run("Confirm", func(t *testing.T) {
		mock := &MockUserInput{
			ConfirmFunc: func(prompt string) bool {
				return prompt == "Continue?"
			},
		}

		if !mock.Confirm("Continue?") {
			t.Error("Expected true for 'Continue?' prompt")
		}
		if mock.Confirm("Stop?") {
			t.Error("Expected false for 'Stop?' prompt")
		}

		if len(mock.ConfirmCalls) != 2 {
			t.Errorf("Expected 2 confirm calls, got %d", len(mock.ConfirmCalls))
		}
	})

	t.Run("ReadInputWithDefault", func(t *testing.T) {
		mock := &MockUserInput{
			ReadInputWithDefaultFunc: func(prompt string, defaultValue string) (string, error) {
				if prompt == "custom" {
					return "custom value", nil
				}
				return defaultValue, nil
			},
		}

		result, err := mock.ReadInputWithDefault("custom", "default")
		if err != nil || result != "custom value" {
			t.Errorf("Expected custom value, got %q, err: %v", result, err)
		}

		result, err = mock.ReadInputWithDefault("other", "fallback")
		if err != nil || result != "fallback" {
			t.Errorf("Expected fallback, got %q, err: %v", result, err)
		}

		if len(mock.ReadInputWithDefaultCalls) != 2 {
			t.Errorf("Expected 2 calls, got %d", len(mock.ReadInputWithDefaultCalls))
		}
	})
}
