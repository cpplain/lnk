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
)

// PrintSkip prints a skip message with a neutral icon
func PrintSkip(format string, args ...interface{}) {
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
	fmt.Printf(format+"\n", args...)
}

// PrintDetail prints an indented detail message (for sub-items)
func PrintDetail(format string, args ...interface{}) {
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

// PrintCommandHeader prints a command header with standard spacing
// This ensures all commands have consistent header formatting
func PrintCommandHeader(text string) {
	if ShouldSimplifyOutput() {
		return
	}
	fmt.Println(Bold(text))
	fmt.Println()
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

// PrintWarningWithHint prints a warning message with an optional hint extracted from the error.
// Always writes to stderr. Not gated by verbosity.
func PrintWarningWithHint(err error) {
	// stub — implementation pending
}

// PrintNextStep prints a standard next step hint.
// sourceDir is contracted via ContractPath for display.
func PrintNextStep(command, sourceDir, description string) {
	PrintInfo("Next: Run 'lnk %s' to %s", command, description)
}

// PrintDryRunSummary prints the standard dry-run mode message
func PrintDryRunSummary() {
	PrintInfo("No changes made in dry-run mode")
}
