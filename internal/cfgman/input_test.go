package cfgman

import (
	"os"
	"strings"
	"testing"
)

// TestReadUserInput tests the ReadUserInput function
func TestReadUserInput(t *testing.T) {
	// Save original stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	tests := []struct {
		name        string
		input       string
		wantOutput  string
		wantErr     bool
		errContains string
	}{
		{
			name:       "normal input",
			input:      "test input\n",
			wantOutput: "test input",
			wantErr:    false,
		},
		{
			name:       "input with spaces",
			input:      "  test input with spaces  \n",
			wantOutput: "test input with spaces",
			wantErr:    false,
		},
		{
			name:       "empty input",
			input:      "\n",
			wantOutput: "",
			wantErr:    false,
		},
		{
			name:        "EOF",
			input:       "", // Empty input simulates EOF
			wantOutput:  "",
			wantErr:     true,
			errContains: "EOF received",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a pipe and set it as stdin
			r, w, _ := os.Pipe()
			os.Stdin = r

			// Write test input
			go func() {
				defer w.Close()
				w.Write([]byte(tt.input))
			}()

			// Call the function
			output, err := ReadUserInput("Test prompt: ")

			// Check error
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			// Check output
			if output != tt.wantOutput {
				t.Errorf("Expected output %q, got %q", tt.wantOutput, output)
			}
		})
	}
}

// TestConfirmPrompt tests the ConfirmPrompt function
func TestConfirmPrompt(t *testing.T) {
	// Save original values
	oldStdin := os.Stdin
	oldTestEnv := os.Getenv("CFGMAN_TEST")
	defer func() {
		os.Stdin = oldStdin
		os.Setenv("CFGMAN_TEST", oldTestEnv)
	}()

	// First test with CFGMAN_TEST=1
	os.Setenv("CFGMAN_TEST", "1")

	result := ConfirmPrompt("Test prompt")
	if result != false {
		t.Errorf("Expected false in test mode, got %v", result)
	}

	result = ConfirmPromptWithTestDefault("Test prompt", true)
	if result != true {
		t.Errorf("Expected true with test default true, got %v", result)
	}

	// Test with actual user input
	os.Setenv("CFGMAN_TEST", "")

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"yes lowercase", "yes\n", true},
		{"yes uppercase", "YES\n", true},
		{"y lowercase", "y\n", true},
		{"y uppercase", "Y\n", true},
		{"no", "no\n", false},
		{"n", "n\n", false},
		{"empty", "\n", false},
		{"invalid", "maybe\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a pipe and set it as stdin
			r, w, _ := os.Pipe()
			os.Stdin = r

			// Write test input
			go func() {
				defer w.Close()
				w.Write([]byte(tt.input))
			}()

			// Call the function
			result := ConfirmPrompt("Test prompt")

			// Check result
			if result != tt.want {
				t.Errorf("For input %q, expected %v, got %v", tt.input, tt.want, result)
			}
		})
	}
}

// TestReadUserInputWithDefault tests the ReadUserInputWithDefault function
func TestReadUserInputWithDefault(t *testing.T) {
	// Save original stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	tests := []struct {
		name         string
		input        string
		defaultValue string
		wantOutput   string
		wantErr      bool
	}{
		{
			name:         "use provided value",
			input:        "custom value\n",
			defaultValue: "default",
			wantOutput:   "custom value",
			wantErr:      false,
		},
		{
			name:         "use default on empty input",
			input:        "\n",
			defaultValue: "default value",
			wantOutput:   "default value",
			wantErr:      false,
		},
		{
			name:         "empty input with no default",
			input:        "\n",
			defaultValue: "",
			wantOutput:   "",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a pipe and set it as stdin
			r, w, _ := os.Pipe()
			os.Stdin = r

			// Write test input
			go func() {
				defer w.Close()
				w.Write([]byte(tt.input))
			}()

			// Call the function
			output, err := ReadUserInputWithDefault("Test prompt", tt.defaultValue)

			// Check error
			if tt.wantErr && err == nil {
				t.Errorf("Expected error but got none")
			} else if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check output
			if output != tt.wantOutput {
				t.Errorf("Expected output %q, got %q", tt.wantOutput, output)
			}
		})
	}
}
