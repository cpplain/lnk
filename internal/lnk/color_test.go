package lnk

import (
	"os"
	"sync"
	"testing"
)

func TestColorOutput(t *testing.T) {
	tests := []struct {
		name     string
		envVar   string
		expected bool
	}{
		{
			name:     "NO_COLOR not set",
			envVar:   "",
			expected: true, // Assuming tests run in a terminal
		},
		{
			name:     "NO_COLOR set to 1",
			envVar:   "1",
			expected: false,
		},
		{
			name:     "NO_COLOR set to true",
			envVar:   "true",
			expected: false,
		},
		{
			name:     "NO_COLOR set to any non-empty value",
			envVar:   "any value",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original NO_COLOR value
			original := os.Getenv("NO_COLOR")
			defer os.Setenv("NO_COLOR", original)

			// Set test environment
			os.Setenv("NO_COLOR", tt.envVar)

			// Reset the color detection for this test
			colorEnabledOnce = sync.Once{}

			// Test color functions
			testString := "test"

			// When NO_COLOR is set, functions should return plain text
			if tt.envVar != "" {
				if Red(testString) != testString {
					t.Errorf("Red() should return plain text when NO_COLOR is set")
				}
				if Green(testString) != testString {
					t.Errorf("Green() should return plain text when NO_COLOR is set")
				}
				if Yellow(testString) != testString {
					t.Errorf("Yellow() should return plain text when NO_COLOR is set")
				}
				if Blue(testString) != testString {
					t.Errorf("Blue() should return plain text when NO_COLOR is set")
				}
				if Cyan(testString) != testString {
					t.Errorf("Cyan() should return plain text when NO_COLOR is set")
				}
				if Bold(testString) != testString {
					t.Errorf("Bold() should return plain text when NO_COLOR is set")
				}
			}
		})
	}
}

func TestColorConstants(t *testing.T) {
	// Verify that color constants are properly defined
	if ColorReset != "\033[0m" {
		t.Errorf("ColorReset has incorrect value")
	}
	if ColorRed != "\033[0;31m" {
		t.Errorf("ColorRed has incorrect value")
	}
	if ColorGreen != "\033[0;32m" {
		t.Errorf("ColorGreen has incorrect value")
	}
	if ColorYellow != "\033[0;33m" {
		t.Errorf("ColorYellow has incorrect value")
	}
	if ColorBlue != "\033[0;34m" {
		t.Errorf("ColorBlue has incorrect value")
	}
	if ColorCyan != "\033[0;36m" {
		t.Errorf("ColorCyan has incorrect value")
	}
	if ColorBold != "\033[1m" {
		t.Errorf("ColorBold has incorrect value")
	}
}

func TestSetNoColor(t *testing.T) {
	// Save original state
	origNoColor := os.Getenv("NO_COLOR")
	defer func() {
		if origNoColor == "" {
			os.Unsetenv("NO_COLOR")
		} else {
			os.Setenv("NO_COLOR", origNoColor)
		}
		// Reset global state
		SetNoColor(false)
		colorEnabledOnce = sync.Once{}
	}()

	tests := []struct {
		name       string
		noColorEnv string
		flagValue  bool
		wantColor  bool
	}{
		{
			name:       "no flag, no env",
			noColorEnv: "",
			flagValue:  false,
			wantColor:  true, // Assuming terminal
		},
		{
			name:       "flag set",
			noColorEnv: "",
			flagValue:  true,
			wantColor:  false,
		},
		{
			name:       "env set",
			noColorEnv: "1",
			flagValue:  false,
			wantColor:  false,
		},
		{
			name:       "both flag and env set",
			noColorEnv: "1",
			flagValue:  true,
			wantColor:  false,
		},
		{
			name:       "flag takes precedence over missing env",
			noColorEnv: "",
			flagValue:  true,
			wantColor:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset state
			colorEnabledOnce = sync.Once{}

			// Set environment
			if tt.noColorEnv == "" {
				os.Unsetenv("NO_COLOR")
			} else {
				os.Setenv("NO_COLOR", tt.noColorEnv)
			}

			// Set flag
			SetNoColor(tt.flagValue)

			// Test color functions
			if tt.wantColor {
				// If we want color, output should be different
				red := Red("test")
				if red == "test" && isTerminal() {
					t.Errorf("Expected colored output, got plain text")
				}
			} else {
				// If we don't want color, output should be plain
				red := Red("test")
				if red != "test" {
					t.Errorf("Expected plain text, got colored output: %q", red)
				}
			}
		})
	}
}
