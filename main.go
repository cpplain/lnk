// Package main provides the command-line interface for lnk,
// an opinionated symlink manager for dotfiles and more.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/cpplain/lnk/lnk"
)

// Version variables set via ldflags during build
var (
	version = "dev"
)

// validCommands lists all recognized subcommands.
var validCommands = []string{"create", "remove", "status", "prune", "adopt", "orphan"}

func main() {
	args := os.Args[1:]

	// Extract global flags that must be handled before command dispatch
	var noColor bool
	for _, arg := range args {
		if arg == "--no-color" {
			noColor = true
		}
	}
	if noColor {
		lnk.SetNoColor(true)
	}

	// Handle --version anywhere in args
	for _, arg := range args {
		if arg == "-V" || arg == "--version" {
			fmt.Printf("lnk %s\n", version)
			return
		}
	}

	// Handle bare `lnk`
	if len(args) == 0 {
		printUsage()
		return
	}

	// Extract the command name and remaining args
	command, remaining := extractCommand(args)

	// Handle --help: if no command found, show main help; otherwise defer to per-command help
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			if command == "" || !isValidCommand(command) {
				printUsage()
				return
			}
			printCommandHelp(command)
			return
		}
	}

	if command == "" {
		lnk.PrintErrorWithHint(lnk.WithHint(
			fmt.Errorf("no command specified"),
			"Run 'lnk --help' to see available commands"))
		os.Exit(lnk.ExitUsage)
	}

	// Validate command name
	if !isValidCommand(command) {
		suggestion := suggestCommand(command)
		if suggestion != "" {
			lnk.PrintErrorWithHint(lnk.WithHint(
				fmt.Errorf("unknown command: %q", command),
				fmt.Sprintf("Did you mean %q?", suggestion)))
		} else {
			lnk.PrintErrorWithHint(lnk.WithHint(
				fmt.Errorf("unknown command: %q", command),
				"Run 'lnk --help' to see available commands"))
		}
		os.Exit(lnk.ExitUsage)
	}

	// Parse flags and positional arguments from remaining args
	var ignorePatterns []string
	var dryRun bool
	var verbose bool
	var positional []string

	for i := 0; i < len(remaining); i++ {
		arg := remaining[i]

		// Stop parsing flags after --
		if arg == "--" {
			positional = append(positional, remaining[i+1:]...)
			break
		}

		// Non-flag argument = positional
		if !strings.HasPrefix(arg, "-") {
			positional = append(positional, arg)
			continue
		}

		// Parse potential flag with value
		flag, value, hasValue, consumed := parseFlagValue(arg, remaining, i)

		switch flag {
		case "--ignore":
			if !hasValue {
				lnk.PrintErrorWithHint(lnk.WithHint(
					fmt.Errorf("--ignore requires a pattern argument"),
					"Example: lnk create --ignore '*.swp' ."))
				os.Exit(lnk.ExitUsage)
			}
			ignorePatterns = append(ignorePatterns, value)
			i += consumed
		case "-n", "--dry-run":
			dryRun = true
		case "-v", "--verbose":
			verbose = true
		case "--no-color":
			// Already handled above
		case "-h", "--help":
			// Already handled above
		default:
			lnk.PrintErrorWithHint(lnk.WithHint(
				fmt.Errorf("unknown flag: %s", flag),
				fmt.Sprintf("Run 'lnk %s --help' to see available flags", command)))
			os.Exit(lnk.ExitUsage)
		}
	}

	// Set verbosity level
	if verbose {
		lnk.SetVerbosity(lnk.VerbosityVerbose)
	}

	// All commands require source-dir as first positional argument
	if len(positional) == 0 {
		lnk.PrintErrorWithHint(lnk.WithHint(
			fmt.Errorf("missing required argument: <source-dir>"),
			fmt.Sprintf("Usage: lnk %s [flags] <source-dir>", command)))
		os.Exit(lnk.ExitUsage)
	}

	sourceDir := positional[0]
	paths := positional[1:] // remaining positional args (for adopt/orphan)

	// Load configuration (resolves sourceDir, loads ignore patterns)
	config, err := lnk.LoadConfig(sourceDir, ignorePatterns)
	if err != nil {
		lnk.PrintErrorWithHint(err)
		os.Exit(lnk.ExitError)
	}

	// Dispatch to command handler
	switch command {
	case "create":
		handleCreate(config, dryRun, paths)
	case "remove":
		handleRemove(config, dryRun, paths)
	case "status":
		handleStatus(config, paths)
	case "prune":
		handlePrune(config, dryRun, paths)
	case "adopt":
		handleAdopt(config, dryRun, paths)
	case "orphan":
		handleOrphan(config, dryRun, paths)
	}
}

func handleCreate(config *lnk.Config, dryRun bool, extra []string) {
	if len(extra) > 0 {
		lnk.PrintErrorWithHint(lnk.WithHint(
			fmt.Errorf("create takes exactly one argument: <source-dir>"),
			"Usage: lnk create [flags] <source-dir>"))
		os.Exit(lnk.ExitUsage)
	}
	opts := lnk.LinkOptions{
		SourceDir:      config.SourceDir,
		TargetDir:      config.TargetDir,
		IgnorePatterns: config.IgnorePatterns,
		DryRun:         dryRun,
	}
	if err := lnk.CreateLinks(opts); err != nil {
		lnk.PrintErrorWithHint(err)
		os.Exit(lnk.ExitError)
	}
}

func handleRemove(config *lnk.Config, dryRun bool, extra []string) {
	if len(extra) > 0 {
		lnk.PrintErrorWithHint(lnk.WithHint(
			fmt.Errorf("remove takes exactly one argument: <source-dir>"),
			"Usage: lnk remove [flags] <source-dir>"))
		os.Exit(lnk.ExitUsage)
	}
	opts := lnk.LinkOptions{
		SourceDir:      config.SourceDir,
		TargetDir:      config.TargetDir,
		IgnorePatterns: config.IgnorePatterns,
		DryRun:         dryRun,
	}
	if err := lnk.RemoveLinks(opts); err != nil {
		lnk.PrintErrorWithHint(err)
		os.Exit(lnk.ExitError)
	}
}

func handleStatus(config *lnk.Config, extra []string) {
	if len(extra) > 0 {
		lnk.PrintErrorWithHint(lnk.WithHint(
			fmt.Errorf("status takes exactly one argument: <source-dir>"),
			"Usage: lnk status [flags] <source-dir>"))
		os.Exit(lnk.ExitUsage)
	}
	opts := lnk.LinkOptions{
		SourceDir:      config.SourceDir,
		TargetDir:      config.TargetDir,
		IgnorePatterns: config.IgnorePatterns,
	}
	if err := lnk.Status(opts); err != nil {
		lnk.PrintErrorWithHint(err)
		os.Exit(lnk.ExitError)
	}
}

func handlePrune(config *lnk.Config, dryRun bool, extra []string) {
	if len(extra) > 0 {
		lnk.PrintErrorWithHint(lnk.WithHint(
			fmt.Errorf("prune takes exactly one argument: <source-dir>"),
			"Usage: lnk prune [flags] <source-dir>"))
		os.Exit(lnk.ExitUsage)
	}
	opts := lnk.LinkOptions{
		SourceDir:      config.SourceDir,
		TargetDir:      config.TargetDir,
		IgnorePatterns: config.IgnorePatterns,
		DryRun:         dryRun,
	}
	if err := lnk.Prune(opts); err != nil {
		lnk.PrintErrorWithHint(err)
		os.Exit(lnk.ExitError)
	}
}

func handleAdopt(config *lnk.Config, dryRun bool, paths []string) {
	if len(paths) == 0 {
		lnk.PrintErrorWithHint(lnk.WithHint(
			fmt.Errorf("adopt requires at least one file path after <source-dir>"),
			"Usage: lnk adopt [flags] <source-dir> <path...>"))
		os.Exit(lnk.ExitUsage)
	}
	opts := lnk.AdoptOptions{
		SourceDir: config.SourceDir,
		TargetDir: config.TargetDir,
		Paths:     paths,
		DryRun:    dryRun,
	}
	if err := lnk.Adopt(opts); err != nil {
		lnk.PrintErrorWithHint(err)
		os.Exit(lnk.ExitError)
	}
}

func handleOrphan(config *lnk.Config, dryRun bool, paths []string) {
	if len(paths) == 0 {
		lnk.PrintErrorWithHint(lnk.WithHint(
			fmt.Errorf("orphan requires at least one path after <source-dir>"),
			"Usage: lnk orphan [flags] <source-dir> <path...>"))
		os.Exit(lnk.ExitUsage)
	}
	opts := lnk.OrphanOptions{
		SourceDir: config.SourceDir,
		TargetDir: config.TargetDir,
		Paths:     paths,
		DryRun:    dryRun,
	}
	if err := lnk.Orphan(opts); err != nil {
		lnk.PrintErrorWithHint(err)
		os.Exit(lnk.ExitError)
	}
}

// parseFlagValue parses a flag that might be in --flag=value or --flag value format.
// Returns the flag name, value, whether a value was found, and how many extra args were consumed.
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

// extractCommand finds the command name in args, returning it and the remaining args.
// The command is the first non-flag token that matches a valid command name or
// appears to be a command (not starting with -).
func extractCommand(args []string) (string, []string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Stop at --
		if arg == "--" {
			break
		}

		// Skip flags
		if strings.HasPrefix(arg, "-") {
			// Skip value of flags that take values (--ignore pattern or --ignore=pattern)
			if arg == "--ignore" && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++ // skip the value token so it isn't mistaken for a command
			}
			continue
		}

		// Found a non-flag token — this is the command
		remaining := make([]string, 0, len(args)-1)
		remaining = append(remaining, args[:i]...)
		remaining = append(remaining, args[i+1:]...)
		return arg, remaining
	}

	return "", args
}

// isValidCommand returns true if name is a recognized command.
func isValidCommand(name string) bool {
	for _, cmd := range validCommands {
		if cmd == name {
			return true
		}
	}
	return false
}

// suggestCommand returns the closest valid command name to input, or empty string
// if no suggestion is close enough.
func suggestCommand(input string) string {
	threshold := len(input)/2 + 1
	best, bestDist := "", threshold+1
	for _, cmd := range validCommands {
		if d := levenshteinDistance(input, cmd); d < bestDist {
			best, bestDist = cmd, d
		}
	}
	return best
}

// levenshteinDistance computes the edit distance between two strings.
func levenshteinDistance(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	// Use single-row optimization
	prev := make([]int, lb+1)
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr := make([]int, lb+1)
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}
		prev = curr
	}
	return prev[lb]
}

// printUsage prints the top-level usage message.
func printUsage() {
	fmt.Print(`Usage: lnk <command> [flags] <source-dir>

An opinionated symlink manager for dotfiles and more

Commands:
  create <source-dir>           Create symlinks from source to ~
  remove <source-dir>           Remove managed symlinks
  status <source-dir>           Show status of managed symlinks
  prune  <source-dir>           Remove broken symlinks
  adopt  <source-dir> <path...> Adopt files into source directory
  orphan <source-dir> <path...> Remove files from management

Flags:
      --ignore PATTERN  Additional ignore pattern, repeatable
  -n, --dry-run         Preview changes without making them
  -v, --verbose         Enable verbose output
      --no-color        Disable colored output
  -V, --version         Show version information
  -h, --help            Show this help message

Examples:
  lnk create .                        Create links from current directory
  lnk create ~/git/dotfiles           Create from absolute path
  lnk create -n .                     Dry-run preview
  lnk remove .                        Remove links
  lnk status .                        Show status
  lnk prune .                         Prune broken symlinks
  lnk prune ~/git/dotfiles            Prune from specific source
  lnk adopt . ~/.bashrc ~/.vimrc      Adopt files into current directory
  lnk adopt ~/dotfiles ~/.bashrc      Adopt with explicit source
  lnk orphan . ~/.bashrc              Remove file from management
  lnk create --ignore '*.swp' .       Add ignore pattern

Config Files:
  .lnkignore in source directory
    Format: gitignore syntax
    Patterns are combined with built-in defaults and --ignore flags
`)
}

// printCommandHelp prints help for a specific command.
func printCommandHelp(command string) {
	switch command {
	case "create":
		fmt.Print(`Usage: lnk create [flags] <source-dir>

Create symlinks from source directory to home directory.

Arguments:
  source-dir    Source directory to link from (required)

Flags:
  (all global flags apply)

Examples:
  lnk create .
  lnk create ~/git/dotfiles
  lnk create -n .
`)
	case "remove":
		fmt.Print(`Usage: lnk remove [flags] <source-dir>

Remove managed symlinks from home directory.

Arguments:
  source-dir    Source directory whose managed links to remove (required)

Flags:
  (all global flags apply)

Examples:
  lnk remove .
  lnk remove ~/git/dotfiles
  lnk remove -n .
`)
	case "status":
		fmt.Print(`Usage: lnk status [flags] <source-dir>

Show status of managed symlinks in home directory.

Arguments:
  source-dir    Source directory to check (required)

Flags:
  (all global flags apply)

Examples:
  lnk status .
  lnk status ~/git/dotfiles
  lnk status ~/git/dotfiles | grep ^broken
`)
	case "prune":
		fmt.Print(`Usage: lnk prune [flags] <source-dir>

Remove broken managed symlinks from home directory.

Arguments:
  source-dir    Source directory whose broken links to prune (required)

Flags:
  (all global flags apply)

Examples:
  lnk prune .
  lnk prune ~/git/dotfiles
  lnk prune -n .
`)
	case "adopt":
		fmt.Print(`Usage: lnk adopt [flags] <source-dir> <path...>

Adopt files into the source directory.

Arguments:
  source-dir    Source directory to move files into (required)
  path          One or more files or directories to adopt; must be within ~ (required)

Flags:
  (all global flags apply)

Examples:
  lnk adopt . ~/.bashrc
  lnk adopt . ~/.bashrc ~/.vimrc
  lnk adopt ~/git/dotfiles ~/.config/nvim
  lnk adopt -n . ~/.bashrc
`)
	case "orphan":
		fmt.Print(`Usage: lnk orphan [flags] <source-dir> <path...>

Remove files from management.

Arguments:
  source-dir    Source directory that manages the files (required)
  path          One or more managed symlinks or directories to orphan; must be within ~ (required)

Flags:
  (all global flags apply)

Examples:
  lnk orphan . ~/.bashrc
  lnk orphan . ~/.bashrc ~/.vimrc
  lnk orphan ~/git/dotfiles ~/.config/nvim
  lnk orphan -n . ~/.bashrc
`)
	}
}
