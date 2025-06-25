package cfgman

import (
	"io"
	"os"
	"strings"
	"testing"
)

// CaptureStdin temporarily replaces stdin with the provided input
func CaptureStdin(t *testing.T, input string) func() {
	t.Helper()

	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	os.Stdin = r

	go func() {
		defer w.Close()
		io.WriteString(w, input)
	}()

	return func() {
		os.Stdin = oldStdin
		r.Close()
	}
}

// CaptureOutput captures stdout during function execution
func CaptureOutput(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	os.Stdout = w

	outChan := make(chan string)
	go func() {
		out, _ := io.ReadAll(r)
		outChan <- string(out)
	}()

	fn()

	w.Close()
	os.Stdout = oldStdout

	return <-outChan
}

// ContainsOutput checks if the output contains all expected strings
func ContainsOutput(t *testing.T, output string, expected ...string) {
	t.Helper()

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("Output missing expected string: %q\nFull output:\n%s", exp, output)
		}
	}
}

// NotContainsOutput checks if the output does not contain any of the strings
func NotContainsOutput(t *testing.T, output string, notExpected ...string) {
	t.Helper()

	for _, notExp := range notExpected {
		if strings.Contains(output, notExp) {
			t.Errorf("Output contains unexpected string: %q\nFull output:\n%s", notExp, output)
		}
	}
}
