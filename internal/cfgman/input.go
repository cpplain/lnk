package cfgman

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// ReadUserInput reads a line of input from the user, handling EOF and interrupts gracefully
func ReadUserInput(prompt string) (string, error) {
	reader := bufio.NewReader(os.Stdin)

	if prompt != "" {
		fmt.Print(prompt)
	}

	input, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			// Handle EOF gracefully (e.g., Ctrl+D)
			fmt.Println() // Add newline for clean output
			return "", fmt.Errorf("EOF received")
		}
		return "", fmt.Errorf("error reading input: %w", err)
	}

	return strings.TrimSpace(input), nil
}

// ConfirmPrompt asks the user for a yes/no confirmation
func ConfirmPrompt(prompt string) bool {
	fullPrompt := fmt.Sprintf("%s [y/N]: ", prompt)
	response, err := ReadUserInput(fullPrompt)
	if err != nil {
		return false
	}

	response = strings.ToLower(response)
	return response == "y" || response == "yes"
}

// ReadUserInputWithDefault reads input with a default value
func ReadUserInputWithDefault(prompt string, defaultValue string) (string, error) {
	fullPrompt := prompt
	if defaultValue != "" {
		fullPrompt = fmt.Sprintf("%s [%s]: ", prompt, defaultValue)
	}

	input, err := ReadUserInput(fullPrompt)
	if err != nil {
		return "", err
	}

	if input == "" && defaultValue != "" {
		return defaultValue, nil
	}

	return input, nil
}
