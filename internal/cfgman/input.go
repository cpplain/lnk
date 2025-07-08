package cfgman

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ConfirmAction prompts the user for confirmation before proceeding with an action.
// Returns true if the user confirms (y/yes), false otherwise.
// If stdout is not a terminal, returns false (safe default for scripts).
func ConfirmAction(prompt string) (bool, error) {
	// Don't prompt if not in a terminal
	if !isTerminal() {
		return false, nil
	}

	// Display the prompt
	fmt.Fprintf(os.Stdout, "%s", prompt)

	// Read user input
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read input: %w", err)
	}

	// Trim whitespace and convert to lowercase
	response = strings.TrimSpace(strings.ToLower(response))

	// Check for affirmative responses
	switch response {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}
