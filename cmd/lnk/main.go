// Package main provides the command-line interface for lnk,
// an opinionated symlink manager for dotfiles and more.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/cpplain/lnk/internal/lnk"
)

// Version variables set via ldflags during build
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

// parseIgnorePatterns parses a comma-separated string of ignore patterns
func parseIgnorePatterns(patterns string) []string {
	if patterns == "" {
		return nil
	}

	result := strings.Split(patterns, ",")
	for i := range result {
		result[i] = strings.TrimSpace(result[i])
	}
	return result
}

// parseFlagValue parses a flag that might be in --flag=value or --flag value format
// Returns the flag name, value, and whether a value was found
func parseFlagValue(arg string, args []string, index int) (flag string, value string, hasValue bool, consumed int) {
	// Check for --flag=value format
	if idx := strings.Index(arg, "="); idx > 0 {
		return arg[:idx], arg[idx+1:], true, 0
	}

	// Check for --flag value format
	if index+1 < len(args) && !strings.HasPrefix(args[index+1], "-") {
		return arg, args[index+1], true, 1
	}

	return arg, "", false, 0
}

// levenshteinDistance calculates the minimum edit distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create a 2D slice for dynamic programming
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
	}

	// Initialize first row and column
	for i := 0; i <= len(s1); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	// Fill the matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// suggestCommand finds the closest matching command
func suggestCommand(input string) string {
	commands := []string{"adopt", "create", "orphan", "prune", "remove", "status", "version"}

	bestMatch := ""
	bestDistance := len(input) + 1

	for _, cmd := range commands {
		dist := levenshteinDistance(input, cmd)
		// Only suggest if the distance is reasonable (less than half the input length)
		if dist < bestDistance && dist <= len(input)/2+1 {
			bestMatch = cmd
			bestDistance = dist
		}
	}

	return bestMatch
}

// min returns the minimum of three integers
func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func printConfigHelp() {
	fmt.Printf("%s lnk help config\n", lnk.Bold("Usage:"))
	fmt.Println("\nConfiguration discovery")
	fmt.Println()
	lnk.PrintHelpSection("Configuration Discovery:")
	fmt.Println("  Configuration is loaded from the first available source:")
	fmt.Println("    1. --config flag")
	fmt.Println("    2. $XDG_CONFIG_HOME/lnk/config.json")
	fmt.Println("    3. ~/.config/lnk/config.json")
	fmt.Println("    4. ~/.lnk.json")
	fmt.Printf("    5. %s in current directory\n", lnk.ConfigFileName)
	fmt.Println("    6. Built-in defaults")
	fmt.Println()
	lnk.PrintHelpSection("Configuration Format:")
	fmt.Println("  Configuration files use JSON format with LinkMapping structure:")
	fmt.Println("  {")
	fmt.Println("    \"mappings\": [")
	fmt.Println("      {")
	fmt.Println("        \"source\": \"~/dotfiles/home\",")
	fmt.Println("        \"target\": \"~/\"")
	fmt.Println("      }")
	fmt.Println("    ],")
	fmt.Println("    \"ignore\": [\".git\", \"*.swp\"]")
	fmt.Println("  }")
}

func main() {
	// Parse global flags first
	var globalVerbose, globalQuiet, globalNoColor, globalVersion, globalYes bool
	var globalConfig, globalIgnore, globalOutput string
	remainingArgs := []string{}

	// Manual parsing to extract global flags before command
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Parse potential flag with value
		flag, value, hasValue, consumed := parseFlagValue(arg, args, i)

		switch flag {
		case "--verbose", "-v":
			globalVerbose = true
		case "--quiet", "-q":
			globalQuiet = true
		case "--output":
			if hasValue {
				globalOutput = value
				i += consumed
			}
		case "--no-color":
			globalNoColor = true
		case "--version":
			globalVersion = true
		case "--yes", "-y":
			globalYes = true
		case "--config":
			if hasValue {
				globalConfig = value
				i += consumed
			}
		case "--ignore":
			if hasValue {
				globalIgnore = value
				i += consumed
			}
		case "-h", "--help":
			// Let it pass through to be handled later
			remainingArgs = append(remainingArgs, arg)
		default:
			remainingArgs = append(remainingArgs, arg)
		}
	}

	// Set color preference first
	if globalNoColor {
		lnk.SetNoColor(true)
	}

	// Handle --version after processing color settings
	if globalVersion {
		lnk.PrintInfo("%s %s", lnk.Bold("lnk version"), lnk.Green(version))
		return
	}

	// Set verbosity level based on flags
	if globalQuiet && globalVerbose {
		lnk.PrintErrorWithHint(lnk.WithHint(
			fmt.Errorf("cannot use --quiet and --verbose together"),
			"Use either --quiet or --verbose, not both"))
		os.Exit(lnk.ExitUsage)
	}
	if globalQuiet {
		lnk.SetVerbosity(lnk.VerbosityQuiet)
	} else if globalVerbose {
		lnk.SetVerbosity(lnk.VerbosityVerbose)
	}

	// Set output format
	switch globalOutput {
	case "json":
		lnk.SetOutputFormat(lnk.FormatJSON)
		// JSON output implies quiet mode for non-data output
		if !globalVerbose {
			lnk.SetVerbosity(lnk.VerbosityQuiet)
		}
	case "text", "":
		// Default is already text/human format
	default:
		lnk.PrintErrorWithHint(lnk.WithHint(
			fmt.Errorf("invalid output format: %s", globalOutput),
			"Valid formats are: text, json"))
		os.Exit(lnk.ExitUsage)
	}

	if len(remainingArgs) < 1 {
		printUsage()
		os.Exit(lnk.ExitUsage)
	}

	command := remainingArgs[0]

	// Handle global help
	if command == "-h" || command == "--help" || command == "help" {
		if len(remainingArgs) > 1 {
			printCommandHelp(remainingArgs[1])
		} else {
			printUsage()
		}
		return
	}

	// Create global config options from parsed flags
	globalOptions := &lnk.ConfigOptions{
		ConfigPath:     globalConfig,
		IgnorePatterns: parseIgnorePatterns(globalIgnore),
	}

	// Route to command handler with remaining args
	commandArgs := remainingArgs[1:]
	switch command {
	case "status":
		handleStatus(commandArgs, globalOptions)
	case "adopt":
		handleAdopt(commandArgs, globalOptions)
	case "orphan":
		handleOrphan(commandArgs, globalOptions, globalYes)
	case "create":
		handleCreate(commandArgs, globalOptions)
	case "remove":
		handleRemove(commandArgs, globalOptions, globalYes)
	case "prune":
		handlePrune(commandArgs, globalOptions, globalYes)
	case "version":
		handleVersion(commandArgs)
	default:
		suggestion := suggestCommand(command)
		if suggestion != "" {
			lnk.PrintErrorWithHint(lnk.WithHint(
				fmt.Errorf("unknown command: %s", command),
				fmt.Sprintf("Did you mean '%s'? Run 'lnk --help' to see available commands", suggestion)))
		} else {
			lnk.PrintErrorWithHint(lnk.WithHint(
				fmt.Errorf("unknown command: %s", command),
				"Run 'lnk --help' to see available commands"))
		}
		os.Exit(lnk.ExitUsage)
	}
}

func handleStatus(args []string, globalOptions *lnk.ConfigOptions) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Printf("%s lnk status [options]\n", lnk.Bold("Usage:"))
		fmt.Println("\nShow status of all managed symlinks")
		fmt.Println()
		lnk.PrintHelpSection("Options:")
		// Collect all options including command-specific flags and config options
		options := [][]string{}
		fs.VisitAll(func(f *flag.Flag) {
			usage := f.Usage
			if f.DefValue != "" && f.DefValue != "false" {
				usage += fmt.Sprintf(" (default: %s)", f.DefValue)
			}
			options = append(options, []string{"--" + f.Name, usage})
		})
		// Add config options
		options = append(options,
			[]string{"--config PATH", "Path to configuration file"},
			[]string{"--ignore LIST", "Ignore patterns (comma-separated)"},
		)
		if len(options) == 0 {
			fmt.Println("  (none)")
		} else {
			lnk.PrintHelpItems(options)
		}
		fmt.Println()
		lnk.PrintHelpSection("Examples:")
		lnk.PrintHelpItems([][]string{
			{"lnk status", ""},
			{"lnk status --output json", ""},
		})
		fmt.Println()
		lnk.PrintHelpSection("See also:")
		fmt.Println("  create, prune")
	}
	fs.Parse(args)

	config, configSource, err := lnk.LoadConfigWithOptions(globalOptions)
	if err != nil {
		lnk.PrintErrorWithHint(err)
		os.Exit(lnk.ExitError)
	}

	// Show config source in verbose mode
	lnk.PrintVerbose("Using configuration from: %s", configSource)

	if err := lnk.Status(config); err != nil {
		lnk.PrintErrorWithHint(err)
		os.Exit(lnk.ExitError)
	}
}

func handleAdopt(args []string, globalOptions *lnk.ConfigOptions) {
	fs := flag.NewFlagSet("adopt", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Preview changes without making them")
	path := fs.String("path", "", "The file or directory to adopt")
	sourceDir := fs.String("source-dir", "", "The source directory (absolute path, e.g., ~/dotfiles/home)")

	fs.Usage = func() {
		fmt.Printf("%s lnk adopt [options]\n", lnk.Bold("Usage:"))
		fmt.Println("\nAdopt a file or directory into the repository")
		fmt.Println()
		lnk.PrintHelpSection("Options:")
		// Collect all options including command-specific flags
		options := [][]string{}
		fs.VisitAll(func(f *flag.Flag) {
			usage := f.Usage
			if f.DefValue != "" && f.DefValue != "false" {
				usage += fmt.Sprintf(" (default: %s)", f.DefValue)
			}
			options = append(options, []string{"--" + f.Name, usage})
		})
		// Note: adopt doesn't need config options since it has its own --source-dir
		if len(options) == 0 {
			fmt.Println("  (none)")
		} else {
			lnk.PrintHelpItems(options)
		}
		fmt.Println()
		lnk.PrintHelpSection("Examples:")
		lnk.PrintHelpItems([][]string{
			{"lnk adopt --path ~/.gitconfig --source-dir ~/dotfiles/home", ""},
			{"lnk adopt --path ~/.ssh/config --source-dir ~/dotfiles/private/home", ""},
			{"lnk adopt --path ~/.bashrc --source-dir ~/dotfiles/home", ""},
		})
		fmt.Println()
		lnk.PrintHelpSection("See also:")
		fmt.Println("  orphan, create, status")
	}

	fs.Parse(args)

	if *path == "" || *sourceDir == "" {
		lnk.PrintErrorWithHint(lnk.WithHint(
			fmt.Errorf("both --path and --source-dir are required"),
			"Run 'lnk adopt --help' for usage examples"))
		os.Exit(lnk.ExitUsage)
	}

	config, configSource, err := lnk.LoadConfigWithOptions(globalOptions)
	if err != nil {
		lnk.PrintErrorWithHint(err)
		os.Exit(lnk.ExitError)
	}

	// Show config source in verbose mode
	lnk.PrintVerbose("Using configuration from: %s", configSource)

	if err := lnk.Adopt(*path, config, *sourceDir, *dryRun); err != nil {
		lnk.PrintErrorWithHint(err)
		os.Exit(lnk.ExitError)
	}
}

func handleOrphan(args []string, globalOptions *lnk.ConfigOptions, globalYes bool) {
	fs := flag.NewFlagSet("orphan", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Preview changes without making them")
	path := fs.String("path", "", "The file or directory to orphan")

	fs.Usage = func() {
		fmt.Printf("%s lnk orphan [options]\n", lnk.Bold("Usage:"))
		fmt.Println("\nRemove a file or directory from repository management")
		fmt.Println("For directories, recursively orphans all managed symlinks within")
		fmt.Println()
		lnk.PrintHelpSection("Options:")
		// Collect all options including command-specific flags and config options
		options := [][]string{}
		fs.VisitAll(func(f *flag.Flag) {
			usage := f.Usage
			if f.DefValue != "" && f.DefValue != "false" {
				usage += fmt.Sprintf(" (default: %s)", f.DefValue)
			}
			options = append(options, []string{"--" + f.Name, usage})
		})
		// Add config options
		options = append(options,
			[]string{"--config PATH", "Path to configuration file"},
			[]string{"--ignore LIST", "Ignore patterns (comma-separated)"},
		)
		if len(options) == 0 {
			fmt.Println("  (none)")
		} else {
			lnk.PrintHelpItems(options)
		}
		fmt.Println()
		lnk.PrintHelpSection("Examples:")
		lnk.PrintHelpItems([][]string{
			{"lnk orphan --path ~/.gitconfig", ""},
			{"lnk orphan --path ~/.config/nvim", ""},
			{"lnk orphan --path ~/.bashrc", ""},
		})
		fmt.Println()
		lnk.PrintHelpSection("See also:")
		fmt.Println("  adopt, status")
	}

	fs.Parse(args)

	if *path == "" {
		lnk.PrintErrorWithHint(lnk.WithHint(
			fmt.Errorf("--path is required"),
			"Run 'lnk orphan --help' for usage examples"))
		os.Exit(lnk.ExitUsage)
	}

	config, configSource, err := lnk.LoadConfigWithOptions(globalOptions)
	if err != nil {
		lnk.PrintErrorWithHint(err)
		os.Exit(lnk.ExitError)
	}

	// Show config source in verbose mode
	lnk.PrintVerbose("Using configuration from: %s", configSource)

	if err := lnk.Orphan(*path, config, *dryRun, globalYes); err != nil {
		lnk.PrintErrorWithHint(err)
		os.Exit(lnk.ExitError)
	}
}

func handleCreate(args []string, globalOptions *lnk.ConfigOptions) {
	fs := flag.NewFlagSet("create", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Preview changes without making them")

	fs.Usage = func() {
		fmt.Printf("%s lnk create [options]\n", lnk.Bold("Usage:"))
		fmt.Println("\nCreate symlinks from repository to target directories")
		fmt.Println()
		lnk.PrintHelpSection("Options:")
		// Collect all options including command-specific flags and config options
		options := [][]string{}
		fs.VisitAll(func(f *flag.Flag) {
			usage := f.Usage
			if f.DefValue != "" && f.DefValue != "false" {
				usage += fmt.Sprintf(" (default: %s)", f.DefValue)
			}
			options = append(options, []string{"--" + f.Name, usage})
		})
		// Add config options
		options = append(options,
			[]string{"--config PATH", "Path to configuration file"},
			[]string{"--ignore LIST", "Ignore patterns (comma-separated)"},
		)
		if len(options) == 0 {
			fmt.Println("  (none)")
		} else {
			lnk.PrintHelpItems(options)
		}
		fmt.Println()
		lnk.PrintHelpSection("Examples:")
		lnk.PrintHelpItems([][]string{
			{"lnk create", ""},
			{"lnk create --dry-run", ""},
		})
		fmt.Println()
		lnk.PrintHelpSection("See also:")
		fmt.Println("  remove, status, adopt")
	}

	fs.Parse(args)

	config, configSource, err := lnk.LoadConfigWithOptions(globalOptions)
	if err != nil {
		lnk.PrintErrorWithHint(err)
		os.Exit(lnk.ExitError)
	}

	// Show config source in verbose mode
	lnk.PrintVerbose("Using configuration from: %s", configSource)

	if err := lnk.CreateLinks(config, *dryRun); err != nil {
		lnk.PrintErrorWithHint(err)
		os.Exit(lnk.ExitError)
	}
}

func handleRemove(args []string, globalOptions *lnk.ConfigOptions, globalYes bool) {
	fs := flag.NewFlagSet("remove", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Preview changes without making them")

	fs.Usage = func() {
		fmt.Printf("%s lnk remove [options]\n", lnk.Bold("Usage:"))
		fmt.Println("\nRemove all managed symlinks")
		fmt.Println()
		lnk.PrintHelpSection("Options:")
		// Collect all options including command-specific flags and config options
		options := [][]string{}
		fs.VisitAll(func(f *flag.Flag) {
			usage := f.Usage
			if f.DefValue != "" && f.DefValue != "false" {
				usage += fmt.Sprintf(" (default: %s)", f.DefValue)
			}
			options = append(options, []string{"--" + f.Name, usage})
		})
		// Add config options
		options = append(options,
			[]string{"--config PATH", "Path to configuration file"},
			[]string{"--ignore LIST", "Ignore patterns (comma-separated)"},
		)
		if len(options) == 0 {
			fmt.Println("  (none)")
		} else {
			lnk.PrintHelpItems(options)
		}
		fmt.Println()
		lnk.PrintHelpSection("Examples:")
		lnk.PrintHelpItems([][]string{
			{"lnk remove", ""},
			{"lnk remove --dry-run", ""},
		})
		fmt.Println()
		lnk.PrintHelpSection("See also:")
		fmt.Println("  create, prune, orphan")
	}

	fs.Parse(args)

	config, configSource, err := lnk.LoadConfigWithOptions(globalOptions)
	if err != nil {
		lnk.PrintErrorWithHint(err)
		os.Exit(lnk.ExitError)
	}

	// Show config source in verbose mode
	lnk.PrintVerbose("Using configuration from: %s", configSource)

	if err := lnk.RemoveLinks(config, *dryRun, globalYes); err != nil {
		lnk.PrintErrorWithHint(err)
		os.Exit(lnk.ExitError)
	}
}

func handlePrune(args []string, globalOptions *lnk.ConfigOptions, globalYes bool) {
	fs := flag.NewFlagSet("prune", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Preview changes without making them")

	fs.Usage = func() {
		fmt.Printf("%s lnk prune [options]\n", lnk.Bold("Usage:"))
		fmt.Println("\nRemove broken symlinks")
		fmt.Println()
		lnk.PrintHelpSection("Options:")
		// Collect all options including command-specific flags and config options
		options := [][]string{}
		fs.VisitAll(func(f *flag.Flag) {
			usage := f.Usage
			if f.DefValue != "" && f.DefValue != "false" {
				usage += fmt.Sprintf(" (default: %s)", f.DefValue)
			}
			options = append(options, []string{"--" + f.Name, usage})
		})
		// Add config options
		options = append(options,
			[]string{"--config PATH", "Path to configuration file"},
			[]string{"--ignore LIST", "Ignore patterns (comma-separated)"},
		)
		if len(options) == 0 {
			fmt.Println("  (none)")
		} else {
			lnk.PrintHelpItems(options)
		}
		fmt.Println()
		lnk.PrintHelpSection("Examples:")
		lnk.PrintHelpItems([][]string{
			{"lnk prune", ""},
			{"lnk prune --dry-run", ""},
		})
		fmt.Println()
		lnk.PrintHelpSection("See also:")
		fmt.Println("  remove, status")
	}

	fs.Parse(args)

	config, configSource, err := lnk.LoadConfigWithOptions(globalOptions)
	if err != nil {
		lnk.PrintErrorWithHint(err)
		os.Exit(lnk.ExitError)
	}

	// Show config source in verbose mode
	lnk.PrintVerbose("Using configuration from: %s", configSource)

	if err := lnk.PruneLinks(config, *dryRun, globalYes); err != nil {
		lnk.PrintErrorWithHint(err)
		os.Exit(lnk.ExitError)
	}
}

func handleVersion(args []string) {
	fs := flag.NewFlagSet("version", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Printf("%s lnk version [options]\n", lnk.Bold("Usage:"))
		fmt.Println("\nShow version information")
		fmt.Println()
		lnk.PrintHelpSection("Options:")
		// Collect all options including command-specific flags
		options := [][]string{}
		fs.VisitAll(func(f *flag.Flag) {
			usage := f.Usage
			if f.DefValue != "" && f.DefValue != "false" {
				usage += fmt.Sprintf(" (default: %s)", f.DefValue)
			}
			options = append(options, []string{"--" + f.Name, usage})
		})
		if len(options) == 0 {
			fmt.Println("  (none)")
		} else {
			lnk.PrintHelpItems(options)
		}
		fmt.Println()
		lnk.PrintHelpSection("Examples:")
		lnk.PrintHelpItems([][]string{
			{"lnk version", ""},
			{"lnk --version", ""},
		})
	}
	fs.Parse(args)

	lnk.PrintInfo("%s %s", lnk.Bold("lnk version"), lnk.Green(version))
	lnk.PrintInfo("  commit: %s", lnk.Cyan(commit))
	lnk.PrintInfo("  built:  %s", lnk.Cyan(date))
}

func printUsage() {
	fmt.Printf("%s lnk [options] <command> [command-options]\n", lnk.Bold("Usage:"))
	fmt.Println()
	fmt.Println("An opinionated symlink manager for dotfiles and more")
	fmt.Println()

	lnk.PrintHelpSection("Commands:")
	lnk.PrintHelpItems([][]string{
		{"adopt", "Adopt file/directory into repository"},
		{"create", "Create symlinks from repo to target dirs"},
		{"orphan", "Remove file/directory from repo management"},
		{"prune", "Remove broken symlinks"},
		{"remove", "Remove all managed symlinks"},
		{"status", "Show status of all managed symlinks"},
		{"version", "Show version information"},
	})
	fmt.Println()

	lnk.PrintHelpSection("Options:")
	lnk.PrintHelpItems([][]string{
		{"-h, --help", "Show this help message"},
		{"    --no-color", "Disable colored output"},
		{"    --output FORMAT", "Output format: text (default), json"},
		{"-q, --quiet", "Suppress all non-error output"},
		{"-v, --verbose", "Enable verbose output"},
		{"    --version", "Show version information"},
		{"-y, --yes", "Assume yes to all prompts"},
	})
	fmt.Println()

	fmt.Printf("Use '%s' for more information about a command\n", lnk.Bold("lnk <command> --help"))
}

func printCommandHelp(command string) {
	// Create empty options for help display
	emptyOptions := &lnk.ConfigOptions{}

	switch command {
	case "status":
		handleStatus([]string{"-h"}, emptyOptions)
	case "adopt":
		handleAdopt([]string{"-h"}, emptyOptions)
	case "orphan":
		handleOrphan([]string{"-h"}, emptyOptions, false)
	case "create":
		handleCreate([]string{"-h"}, emptyOptions)
	case "remove":
		handleRemove([]string{"-h"}, emptyOptions, false)
	case "prune":
		handlePrune([]string{"-h"}, emptyOptions, false)
	case "version":
		handleVersion([]string{"-h"})
	case "config":
		printConfigHelp()
	default:
		suggestion := suggestCommand(command)
		if suggestion != "" {
			lnk.PrintErrorWithHint(lnk.WithHint(
				fmt.Errorf("unknown command: %s", command),
				fmt.Sprintf("Did you mean 'lnk help %s'?", suggestion)))
		} else {
			lnk.PrintErrorWithHint(lnk.WithHint(
				fmt.Errorf("unknown command: %s", command),
				"Run 'lnk --help' to see available commands"))
		}
	}
}
