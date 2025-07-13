package lnk

// Output Standards
//
// All commands should follow these output patterns for consistency:
//
// 1. Command Header:
//    PrintCommandHeader("Command Name")
//
// 2. Empty Results:
//    PrintEmptyResult("items to process")
//
// 3. Individual Operations:
//    PrintSuccess("Action: %s", path)
//    PrintError("Failed: %s", err)
//
// 4. Summary Section (when applicable):
//    PrintSummary("Successfully processed %d item(s)", count)
//
// 5. Next Steps (when applicable):
//    PrintNextStep("command", "description of what it does")
//
// 6. Dry Run Mode:
//    - Show what would happen without making changes
//    - Prefix operations with DryRunPrefix
//    - End with: PrintDryRunSummary()
//
// Example flow:
//    PrintCommandHeader("Creating Symlinks")
//    // ... operations ...
//    PrintSummary("Created %d symlink(s) successfully", count)
//    PrintNextStep("status", "verify links")

import (
	"fmt"
	"os"
	"text/tabwriter"
)

// Debug prints debug messages to stderr when LNK_DEBUG is set or in verbose mode.
// This follows the pattern used by many Go CLI tools for debug output.
func Debug(format string, args ...interface{}) {
	if os.Getenv("LNK_DEBUG") != "" || IsVerbose() {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// PrintHeader prints a bold header for command output
func PrintHeader(text string) {
	if IsQuiet() {
		return
	}
	fmt.Println(Bold(text))
}

// PrintSkip prints a skip message with a neutral icon
func PrintSkip(format string, args ...interface{}) {
	if IsQuiet() {
		return
	}
	message := fmt.Sprintf(format, args...)
	if ShouldSimplifyOutput() {
		// For piped output, use simple text marker
		fmt.Printf("skip %s\n", message)
	} else {
		fmt.Printf("%s %s\n", Yellow("○"), message)
	}
}

// PrintWarning prints a warning message to stderr with the warning icon
func PrintWarning(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	if ShouldSimplifyOutput() {
		// For piped output, use simple text marker
		fmt.Fprintf(os.Stderr, "warning: %s\n", message)
	} else {
		fmt.Fprintf(os.Stderr, "%s %s\n", Yellow(WarningIcon), message)
	}
}

// PrintSuccess prints a success message with the success icon
func PrintSuccess(format string, args ...interface{}) {
	if IsQuiet() {
		return
	}
	message := fmt.Sprintf(format, args...)
	if ShouldSimplifyOutput() {
		// For piped output, use simple text marker
		fmt.Printf("success %s\n", message)
	} else {
		fmt.Printf("%s %s\n", Green(SuccessIcon), message)
	}
}

// PrintDryRun prints a dry-run message with the dry-run prefix
func PrintDryRun(format string, args ...interface{}) {
	if IsQuiet() {
		return
	}
	message := fmt.Sprintf(format, args...)
	if ShouldSimplifyOutput() {
		// For piped output, use simple text marker
		fmt.Printf("dry-run: %s\n", message)
	} else {
		fmt.Printf("%s %s\n", Yellow(DryRunPrefix), message)
	}
}

// PrintError prints an error message to stderr with the error icon
func PrintError(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	if ShouldSimplifyOutput() {
		// For piped output, use simple text marker
		fmt.Fprintf(os.Stderr, "error: %s\n", message)
	} else {
		fmt.Fprintf(os.Stderr, "%s Error: %s\n", Red(FailureIcon), message)
	}
}

// PrintErrorWithHint prints an error message with an optional hint
func PrintErrorWithHint(err error) {
	if ShouldSimplifyOutput() {
		// For piped output, use simple format
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		if hint := GetErrorHint(err); hint != "" {
			fmt.Fprintf(os.Stderr, "hint: %s\n", hint)
		}
	} else {
		// First print the error message
		fmt.Fprintf(os.Stderr, "%s Error: %v\n", Red(FailureIcon), err)

		// Check if there's a hint
		if hint := GetErrorHint(err); hint != "" {
			fmt.Fprintf(os.Stderr, "  %s %s\n", Cyan("Try:"), hint)
		}
	}
}

// PrintInfo prints an informational message without any prefix
func PrintInfo(format string, args ...interface{}) {
	if IsQuiet() {
		return
	}
	fmt.Printf(format+"\n", args...)
}

// PrintDetail prints an indented detail message (for sub-items)
func PrintDetail(format string, args ...interface{}) {
	if IsQuiet() {
		return
	}
	message := fmt.Sprintf(format, args...)
	fmt.Printf("  %s\n", message)
}

// PrintVerbose prints a message only when in verbose mode
func PrintVerbose(format string, args ...interface{}) {
	if !IsVerbose() {
		return
	}
	message := fmt.Sprintf(format, args...)
	fmt.Printf("[VERBOSE] %s\n", message)
}

// PrintHelpSection prints a section header for help text
func PrintHelpSection(title string) {
	fmt.Println(Bold(title))
}

// PrintHelpItem prints an aligned help item using tabwriter
// This ensures consistent spacing across all help sections
func PrintHelpItem(name, description string) {
	// Using a single shared tabwriter would be more efficient,
	// but for simplicity we create one per call
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "  %s\t%s\n", name, description)
	w.Flush()
}

// PrintHelpItems prints multiple aligned help items at once
// This is more efficient than calling PrintHelpItem multiple times
func PrintHelpItems(items [][]string) {
	if len(items) == 0 {
		return
	}

	// Find the longest item in the first column for proper padding
	maxLen := 0
	for _, item := range items {
		if len(item) >= 1 && len(item[0]) > maxLen {
			maxLen = len(item[0])
		}
	}

	// Print with consistent spacing (no extra padding)
	for _, item := range items {
		if len(item) >= 2 {
			fmt.Printf("  %-*s  %s\n", maxLen, item[0], item[1])
		}
	}
}

// PrintCommandHeader prints a command header with standard spacing
// This ensures all commands have consistent header formatting
func PrintCommandHeader(text string) {
	PrintHeader(text)
	fmt.Println() // Standard newline after header
}

// PrintSummary prints a summary with standard spacing
// This ensures all summaries have consistent formatting
func PrintSummary(format string, args ...interface{}) {
	fmt.Println() // Standard newline before summary
	PrintSuccess(format, args...)
}

// PrintEmptyResult prints a standard "No X found" message
func PrintEmptyResult(itemType string) {
	PrintInfo("No %s found.", itemType)
}

// PrintNextStep prints a standard next step hint
func PrintNextStep(command, description string) {
	PrintInfo("Next: Run 'lnk %s' to %s", command, description)
}

// PrintDryRunSummary prints the standard dry-run mode message
func PrintDryRunSummary() {
	PrintInfo("No changes made in dry-run mode")
}
