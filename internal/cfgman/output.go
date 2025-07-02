package cfgman

import (
	"fmt"
	"os"
)

// Debug prints debug messages to stderr when CFGMAN_DEBUG is set.
// This follows the pattern used by many Go CLI tools for debug output.
func Debug(format string, args ...interface{}) {
	if os.Getenv("CFGMAN_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// PrintHeader prints a bold header for command output
func PrintHeader(text string) {
	fmt.Println(Bold(text))
}

// PrintSkip prints a skip message with a neutral icon
func PrintSkip(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", Yellow("○"), message)
}

// PrintWarning prints a warning message to stderr with the warning icon
func PrintWarning(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "%s %s\n", Yellow(WarningIcon), message)
}

// PrintSuccess prints a success message with the success icon
func PrintSuccess(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", Green(SuccessIcon), message)
}

// PrintDryRun prints a dry-run message with the dry-run prefix
func PrintDryRun(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", Yellow(DryRunPrefix), message)
}

// PrintError prints an error message to stderr with the error icon
func PrintError(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "%s Error: %s\n", Red(FailureIcon), message)
}

// PrintInfo prints an informational message without any prefix
func PrintInfo(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

// PrintDetail prints an indented detail message (for sub-items)
func PrintDetail(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Printf("  %s\n", message)
}
