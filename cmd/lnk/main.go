// Package main provides the command-line interface for lnk,
// an opinionated symlink manager for dotfiles and more.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/cpplain/lnk/internal/lnk"
)

// Version variables set via ldflags during build
var (
	version = "dev"
)

// actionFlag represents the action to perform
type actionFlag int

const (
	actionCreate actionFlag = iota
	actionRemove
	actionStatus
	actionPrune
	actionAdopt
	actionOrphan
)

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

// printVersion prints the version information
func printVersion() {
	fmt.Printf("lnk %s\n", version)
}

func main() {
	// Parse flags
	var action actionFlag = actionCreate // default action
	var actionSet bool = false           // track if action was explicitly set
	var sourceDir string = "."           // default: current directory
	var targetDir string = "~"           // default: home directory
	var ignorePatterns []string
	var dryRun bool
	var verbose bool
	var quiet bool
	var noColor bool
	var showVersion bool
	var showHelp bool
	var paths []string

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Stop parsing flags after --
		if arg == "--" {
			paths = append(paths, args[i+1:]...)
			break
		}

		// Non-flag argument = path (positional argument)
		if !strings.HasPrefix(arg, "-") {
			paths = append(paths, arg)
			continue
		}

		// Parse potential flag with value
		flag, value, hasValue, consumed := parseFlagValue(arg, args, i)

		switch flag {
		// Action flags (mutually exclusive)
		case "-C", "--create":
			if actionSet {
				lnk.PrintErrorWithHint(lnk.WithHint(
					fmt.Errorf("cannot use multiple action flags"),
					"Use only one of: -C/--create, -R/--remove, -S/--status, -P/--prune, -A/--adopt, -O/--orphan"))
				os.Exit(lnk.ExitUsage)
			}
			action = actionCreate
			actionSet = true
		case "-R", "--remove":
			if actionSet {
				lnk.PrintErrorWithHint(lnk.WithHint(
					fmt.Errorf("cannot use multiple action flags"),
					"Use only one of: -C/--create, -R/--remove, -S/--status, -P/--prune, -A/--adopt, -O/--orphan"))
				os.Exit(lnk.ExitUsage)
			}
			action = actionRemove
			actionSet = true
		case "-S", "--status":
			if actionSet {
				lnk.PrintErrorWithHint(lnk.WithHint(
					fmt.Errorf("cannot use multiple action flags"),
					"Use only one of: -C/--create, -R/--remove, -S/--status, -P/--prune, -A/--adopt, -O/--orphan"))
				os.Exit(lnk.ExitUsage)
			}
			action = actionStatus
			actionSet = true
		case "-P", "--prune":
			if actionSet {
				lnk.PrintErrorWithHint(lnk.WithHint(
					fmt.Errorf("cannot use multiple action flags"),
					"Use only one of: -C/--create, -R/--remove, -S/--status, -P/--prune, -A/--adopt, -O/--orphan"))
				os.Exit(lnk.ExitUsage)
			}
			action = actionPrune
			actionSet = true
		case "-A", "--adopt":
			if actionSet {
				lnk.PrintErrorWithHint(lnk.WithHint(
					fmt.Errorf("cannot use multiple action flags"),
					"Use only one of: -C/--create, -R/--remove, -S/--status, -P/--prune, -A/--adopt, -O/--orphan"))
				os.Exit(lnk.ExitUsage)
			}
			action = actionAdopt
			actionSet = true
		case "-O", "--orphan":
			if actionSet {
				lnk.PrintErrorWithHint(lnk.WithHint(
					fmt.Errorf("cannot use multiple action flags"),
					"Use only one of: -C/--create, -R/--remove, -S/--status, -P/--prune, -A/--adopt, -O/--orphan"))
				os.Exit(lnk.ExitUsage)
			}
			action = actionOrphan
			actionSet = true

		// Directory flags
		case "-s", "--source":
			if !hasValue {
				lnk.PrintErrorWithHint(lnk.WithHint(
					fmt.Errorf("--source requires a directory argument"),
					"Example: lnk --source ~/git/dotfiles"))
				os.Exit(lnk.ExitUsage)
			}
			sourceDir = value
			i += consumed
		case "-t", "--target":
			if !hasValue {
				lnk.PrintErrorWithHint(lnk.WithHint(
					fmt.Errorf("--target requires a directory argument"),
					"Example: lnk --target ~"))
				os.Exit(lnk.ExitUsage)
			}
			targetDir = value
			i += consumed

		// Other flags
		case "--ignore":
			if !hasValue {
				lnk.PrintErrorWithHint(lnk.WithHint(
					fmt.Errorf("--ignore requires a pattern argument"),
					"Example: lnk --ignore '*.swp'"))
				os.Exit(lnk.ExitUsage)
			}
			ignorePatterns = append(ignorePatterns, value)
			i += consumed
		case "-n", "--dry-run":
			dryRun = true
		case "-v", "--verbose":
			verbose = true
		case "-q", "--quiet":
			quiet = true
		case "--no-color":
			noColor = true
		case "-V", "--version":
			showVersion = true
		case "-h", "--help":
			showHelp = true

		default:
			lnk.PrintErrorWithHint(lnk.WithHint(
				fmt.Errorf("unknown flag: %s", flag),
				"Run 'lnk --help' to see available flags"))
			os.Exit(lnk.ExitUsage)
		}
	}

	// Set color preference first
	if noColor {
		lnk.SetNoColor(true)
	}

	// Handle --version
	if showVersion {
		printVersion()
		return
	}

	// Handle --help
	if showHelp {
		printUsage()
		return
	}

	// Handle conflicting verbosity flags
	if quiet && verbose {
		lnk.PrintErrorWithHint(lnk.WithHint(
			fmt.Errorf("cannot use --quiet and --verbose together"),
			"Use either --quiet or --verbose, not both"))
		os.Exit(lnk.ExitUsage)
	}

	// Set verbosity level
	if quiet {
		lnk.SetVerbosity(lnk.VerbosityQuiet)
	} else if verbose {
		lnk.SetVerbosity(lnk.VerbosityVerbose)
	}

	// Validate path requirements based on action
	// For C/R/S: need at least one path (source directory)
	// For A/O: need at least one path (files to operate on)
	// For P: optional (defaults to current source)
	if action != actionPrune && len(paths) == 0 {
		lnk.PrintErrorWithHint(lnk.WithHint(
			fmt.Errorf("at least one path is required"),
			"Example: lnk . (link from current directory) or lnk -A ~/.bashrc (adopt file)"))
		os.Exit(lnk.ExitUsage)
	}

	// For C/R/S actions, use the first path as the source directory
	if action == actionCreate || action == actionRemove || action == actionStatus {
		if len(paths) > 0 {
			sourceDir = paths[0]
		}
	}

	// Merge config from .lnkconfig and .lnkignore
	mergedConfig, err := lnk.LoadConfig(sourceDir, targetDir, ignorePatterns)
	if err != nil {
		lnk.PrintErrorWithHint(err)
		os.Exit(lnk.ExitError)
	}

	// Show effective configuration in verbose mode
	lnk.PrintVerbose("Source directory: %s", mergedConfig.SourceDir)
	lnk.PrintVerbose("Target directory: %s", mergedConfig.TargetDir)
	if len(paths) > 0 {
		lnk.PrintVerbose("Paths: %s", strings.Join(paths, ", "))
	}

	// Execute the appropriate action
	switch action {
	case actionCreate:
		opts := lnk.LinkOptions{
			SourceDir:      mergedConfig.SourceDir,
			TargetDir:      mergedConfig.TargetDir,
			IgnorePatterns: mergedConfig.IgnorePatterns,
			DryRun:         dryRun,
		}
		if err := lnk.CreateLinks(opts); err != nil {
			lnk.PrintErrorWithHint(err)
			os.Exit(lnk.ExitError)
		}

	case actionRemove:
		opts := lnk.LinkOptions{
			SourceDir:      mergedConfig.SourceDir,
			TargetDir:      mergedConfig.TargetDir,
			IgnorePatterns: mergedConfig.IgnorePatterns,
			DryRun:         dryRun,
		}
		if err := lnk.RemoveLinks(opts); err != nil {
			lnk.PrintErrorWithHint(err)
			os.Exit(lnk.ExitError)
		}

	case actionStatus:
		opts := lnk.LinkOptions{
			SourceDir:      mergedConfig.SourceDir,
			TargetDir:      mergedConfig.TargetDir,
			IgnorePatterns: mergedConfig.IgnorePatterns,
			DryRun:         false, // status doesn't use dry-run
		}
		if err := lnk.Status(opts); err != nil {
			lnk.PrintErrorWithHint(err)
			os.Exit(lnk.ExitError)
		}

	case actionPrune:
		// For prune, use current source if no path specified
		pruneSource := mergedConfig.SourceDir
		if len(paths) > 0 {
			pruneSource = paths[0]
			// Re-merge config with the specified source
			pruneConfig, err := lnk.LoadConfig(pruneSource, targetDir, ignorePatterns)
			if err != nil {
				lnk.PrintErrorWithHint(err)
				os.Exit(lnk.ExitError)
			}
			mergedConfig = pruneConfig
		}
		opts := lnk.LinkOptions{
			SourceDir:      mergedConfig.SourceDir,
			TargetDir:      mergedConfig.TargetDir,
			IgnorePatterns: mergedConfig.IgnorePatterns,
			DryRun:         dryRun,
		}
		if err := lnk.Prune(opts); err != nil {
			lnk.PrintErrorWithHint(err)
			os.Exit(lnk.ExitError)
		}

	case actionAdopt:
		// For adopt, all paths are files to adopt
		if len(paths) == 0 {
			lnk.PrintErrorWithHint(lnk.WithHint(
				fmt.Errorf("adopt requires at least one file path"),
				"Example: lnk -A ~/.bashrc ~/.vimrc"))
			os.Exit(lnk.ExitUsage)
		}
		opts := lnk.AdoptOptions{
			SourceDir: mergedConfig.SourceDir,
			TargetDir: mergedConfig.TargetDir,
			Paths:     paths,
			DryRun:    dryRun,
		}
		if err := lnk.Adopt(opts); err != nil {
			lnk.PrintErrorWithHint(err)
			os.Exit(lnk.ExitError)
		}

	case actionOrphan:
		// For orphan, all paths are symlinks to orphan
		if len(paths) == 0 {
			lnk.PrintErrorWithHint(lnk.WithHint(
				fmt.Errorf("orphan requires at least one path"),
				"Example: lnk -O ~/.bashrc"))
			os.Exit(lnk.ExitUsage)
		}
		opts := lnk.OrphanOptions{
			SourceDir: mergedConfig.SourceDir,
			TargetDir: mergedConfig.TargetDir,
			Paths:     paths,
			DryRun:    dryRun,
		}
		if err := lnk.Orphan(opts); err != nil {
			lnk.PrintErrorWithHint(err)
			os.Exit(lnk.ExitError)
		}
	}
}

func printUsage() {
	fmt.Printf("%s lnk [action] [flags] <path(s)>\n", lnk.Bold("Usage:"))
	fmt.Println()
	fmt.Println("An opinionated symlink manager for dotfiles and more")
	fmt.Println()
	fmt.Println("Paths are positional arguments that come last (POSIX-style).")
	fmt.Println("For create/remove/status: path is the source directory to link from.")
	fmt.Println("For adopt/orphan: paths are the files to operate on.")
	fmt.Println()

	lnk.PrintHelpSection("Action Flags (mutually exclusive):")
	lnk.PrintHelpItems([][]string{
		{"-C, --create", "Create symlinks (default action)"},
		{"-R, --remove", "Remove symlinks"},
		{"-S, --status", "Show status of symlinks"},
		{"-P, --prune", "Remove broken symlinks"},
		{"-A, --adopt", "Adopt files into source directory"},
		{"-O, --orphan", "Remove files from management"},
	})
	fmt.Println()

	lnk.PrintHelpSection("Directory Flags:")
	lnk.PrintHelpItems([][]string{
		{"-s, --source DIR", "Source directory (default: cwd for adopt/orphan)"},
		{"-t, --target DIR", "Target directory (default: ~)"},
	})
	fmt.Println()

	lnk.PrintHelpSection("Other Flags:")
	lnk.PrintHelpItems([][]string{
		{"    --ignore PATTERN", "Additional ignore pattern (repeatable)"},
		{"-n, --dry-run", "Preview changes without making them"},
		{"-v, --verbose", "Enable verbose output"},
		{"-q, --quiet", "Suppress all non-error output"},
		{"    --no-color", "Disable colored output"},
		{"-V, --version", "Show version information"},
		{"-h, --help", "Show this help message"},
	})
	fmt.Println()

	lnk.PrintHelpSection("Examples:")
	lnk.PrintHelpItems([][]string{
		{"lnk .", "Create links from current directory"},
		{"lnk -C .", "Explicit create from current directory"},
		{"lnk -C -t /tmp .", "Create with custom target"},
		{"lnk -C ~/git/dotfiles", "Create from absolute path"},
		{"lnk -n .", "Dry-run (preview without changes)"},
		{"lnk -R .", "Remove links"},
		{"lnk -S .", "Show status"},
		{"lnk -P", "Prune broken symlinks from current source"},
		{"lnk -A ~/.bashrc ~/.vimrc", "Adopt files into current directory"},
		{"lnk -A -s ~/dotfiles ~/.bashrc", "Adopt with explicit source"},
		{"lnk -O ~/.bashrc", "Orphan file (remove from management)"},
		{"lnk --ignore '*.swp' .", "Add ignore pattern"},
	})
	fmt.Println()

	lnk.PrintHelpSection("Config Files:")
	fmt.Println("  .lnkconfig in source directory (repo-specific)")
	fmt.Println("    Format: CLI flags, one per line")
	fmt.Println("    Example:")
	fmt.Println("      --target=~")
	fmt.Println("      --ignore=local/")
	fmt.Println()
	fmt.Println("  .lnkignore in source directory")
	fmt.Println("    Format: gitignore syntax")
	fmt.Println("    Example:")
	fmt.Println("      .git")
	fmt.Println("      *.swp")
	fmt.Println("      README.md")
	fmt.Println()
	fmt.Println("  CLI flags take precedence over config files")
}
