// Package main provides the command-line interface for cfgman,
// a dotfile management tool that manages configuration files
// across machines using intelligent symlinks.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/cpplain/cfgman/internal/cfgman"
)

// Version variables set via ldflags during build
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

// formatFlags returns a formatted string of all flags in the FlagSet
func formatFlags(fs *flag.FlagSet) string {
	var b strings.Builder
	count := 0
	fs.VisitAll(func(f *flag.Flag) {
		// For boolean flags that default to false, we don't show the default
		// as it's implied. For other types, we would show: (default: value)
		if f.DefValue != "" && f.DefValue != "false" {
			fmt.Fprintf(&b, "  --%s\t%s (default: %s)\n", f.Name, f.Usage, f.DefValue)
		} else {
			fmt.Fprintf(&b, "  --%s\t%s\n", f.Name, f.Usage)
		}
		count++
	})
	if count == 0 {
		return "  (none)\n"
	}
	return b.String()
}

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
	commands := []string{"status", "adopt", "orphan", "link", "unlink", "prune", "version", "help"}

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

func main() {
	// Parse global flags first
	var globalVerbose, globalQuiet, globalNoColor, globalVersion, globalYes bool
	var globalConfig, globalRepoDir, globalSourceDir, globalTargetDir, globalIgnore, globalOutput string
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
		case "--repo-dir":
			if hasValue {
				globalRepoDir = value
				i += consumed
			}
		case "--source-dir":
			if hasValue {
				globalSourceDir = value
				i += consumed
			}
		case "--target-dir":
			if hasValue {
				globalTargetDir = value
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
		cfgman.SetNoColor(true)
	}

	// Handle --version after processing color settings
	if globalVersion {
		cfgman.PrintInfo("%s %s", cfgman.Bold("cfgman version"), cfgman.Green(version))
		return
	}

	// Set verbosity level based on flags
	if globalQuiet && globalVerbose {
		cfgman.PrintErrorWithHint(cfgman.WithHint(
			fmt.Errorf("cannot use --quiet and --verbose together"),
			"Use either --quiet or --verbose, not both"))
		os.Exit(cfgman.ExitUsage)
	}
	if globalQuiet {
		cfgman.SetVerbosity(cfgman.VerbosityQuiet)
	} else if globalVerbose {
		cfgman.SetVerbosity(cfgman.VerbosityVerbose)
	}

	// Set output format
	switch globalOutput {
	case "json":
		cfgman.SetOutputFormat(cfgman.FormatJSON)
		// JSON output implies quiet mode for non-data output
		if !globalVerbose {
			cfgman.SetVerbosity(cfgman.VerbosityQuiet)
		}
	case "text", "":
		// Default is already text/human format
	default:
		cfgman.PrintErrorWithHint(cfgman.WithHint(
			fmt.Errorf("invalid output format: %s", globalOutput),
			"Valid formats are: text, json"))
		os.Exit(cfgman.ExitUsage)
	}

	if len(remainingArgs) < 1 {
		printUsage()
		os.Exit(cfgman.ExitUsage)
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
	globalOptions := &cfgman.ConfigOptions{
		ConfigPath:     globalConfig,
		RepoDir:        globalRepoDir,
		SourceDir:      globalSourceDir,
		TargetDir:      globalTargetDir,
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
	case "link":
		handleLink(commandArgs, globalOptions)
	case "unlink":
		handleUnlink(commandArgs, globalOptions, globalYes)
	case "prune":
		handlePrune(commandArgs, globalOptions, globalYes)
	case "version":
		handleVersion(commandArgs)
	default:
		suggestion := suggestCommand(command)
		if suggestion != "" {
			cfgman.PrintErrorWithHint(cfgman.WithHint(
				fmt.Errorf("unknown command: %s", command),
				fmt.Sprintf("Did you mean '%s'? Run 'cfgman --help' to see available commands", suggestion)))
		} else {
			cfgman.PrintErrorWithHint(cfgman.WithHint(
				fmt.Errorf("unknown command: %s", command),
				"Run 'cfgman --help' to see available commands"))
		}
		os.Exit(cfgman.ExitUsage)
	}
}

func handleStatus(args []string, globalOptions *cfgman.ConfigOptions) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Printf("%s cfgman status [options]\n", cfgman.Bold("Usage:"))
		fmt.Printf("\n%s\n", cfgman.Cyan("Show status of all managed symlinks"))
		fmt.Printf("\n%s\n", cfgman.Bold("Options:"))
		fmt.Print(formatFlags(fs))
		fmt.Printf("\n%s\n", cfgman.Bold("Examples:"))
		fmt.Println(cfgman.Cyan("  cfgman status"))
		fmt.Println(cfgman.Cyan("  cfgman status --output json"))
		fmt.Println(cfgman.Cyan("  cfgman status --repo-dir ~/dotfiles"))
		fmt.Printf("\n%s\n", cfgman.Bold("See also:"))
		fmt.Printf("  %s\n", cfgman.Cyan("link, prune"))
	}
	fs.Parse(args)

	config, configSource, err := cfgman.LoadConfigWithOptions(globalOptions)
	if err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(cfgman.ExitError)
	}

	// Show config source in verbose mode
	cfgman.PrintVerbose("Using configuration from: %s", configSource)

	if err := cfgman.Status(globalOptions.RepoDir, config); err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(cfgman.ExitError)
	}
}

func handleAdopt(args []string, globalOptions *cfgman.ConfigOptions) {
	fs := flag.NewFlagSet("adopt", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Preview changes without making them")
	path := fs.String("path", "", "The file or directory to adopt")
	sourceDir := fs.String("source-dir", "", "The source directory in the repository (e.g., home, private/home)")

	fs.Usage = func() {
		fmt.Printf("%s cfgman adopt [options]\n", cfgman.Bold("Usage:"))
		fmt.Printf("\n%s\n", cfgman.Cyan("Adopt a file or directory into the repository"))
		fmt.Printf("\n%s\n", cfgman.Bold("Options:"))
		fmt.Print(formatFlags(fs))
		fmt.Printf("\n%s\n", cfgman.Bold("Examples:"))
		fmt.Println(cfgman.Cyan("  cfgman adopt --path ~/.gitconfig --source-dir home"))
		fmt.Println(cfgman.Cyan("  cfgman adopt --path ~/.ssh/config --source-dir private/home"))
		fmt.Println(cfgman.Cyan("  cfgman adopt --path ~/.bashrc --source-dir home --repo-dir ~/dotfiles"))
		fmt.Printf("\n%s\n", cfgman.Bold("See also:"))
		fmt.Printf("  %s\n", cfgman.Cyan("orphan, link, status"))
	}

	fs.Parse(args)

	if *path == "" || *sourceDir == "" {
		cfgman.PrintErrorWithHint(cfgman.WithHint(
			fmt.Errorf("both --path and --source-dir are required"),
			"Run 'cfgman adopt --help' for usage examples"))
		os.Exit(cfgman.ExitUsage)
	}

	config, configSource, err := cfgman.LoadConfigWithOptions(globalOptions)
	if err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(cfgman.ExitError)
	}

	// Show config source in verbose mode
	cfgman.PrintVerbose("Using configuration from: %s", configSource)

	if err := cfgman.Adopt(*path, globalOptions.RepoDir, config, *sourceDir, *dryRun); err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(cfgman.ExitError)
	}
}

func handleOrphan(args []string, globalOptions *cfgman.ConfigOptions, globalYes bool) {
	fs := flag.NewFlagSet("orphan", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Preview changes without making them")
	path := fs.String("path", "", "The file or directory to orphan")

	fs.Usage = func() {
		fmt.Printf("%s cfgman orphan [options]\n", cfgman.Bold("Usage:"))
		fmt.Printf("\n%s\n", cfgman.Cyan("Remove a file or directory from repository management"))
		fmt.Println("For directories, recursively orphans all managed symlinks within")
		fmt.Printf("\n%s\n", cfgman.Bold("Options:"))
		fmt.Print(formatFlags(fs))
		fmt.Printf("\n%s\n", cfgman.Bold("Examples:"))
		fmt.Println(cfgman.Cyan("  cfgman orphan --path ~/.gitconfig"))
		fmt.Println(cfgman.Cyan("  cfgman orphan --path ~/.config/nvim"))
		fmt.Println(cfgman.Cyan("  cfgman orphan --path ~/.bashrc --repo-dir ~/dotfiles"))
		fmt.Printf("\n%s\n", cfgman.Bold("See also:"))
		fmt.Printf("  %s\n", cfgman.Cyan("adopt, status"))
	}

	fs.Parse(args)

	if *path == "" {
		cfgman.PrintErrorWithHint(cfgman.WithHint(
			fmt.Errorf("--path is required"),
			"Run 'cfgman orphan --help' for usage examples"))
		os.Exit(cfgman.ExitUsage)
	}

	config, configSource, err := cfgman.LoadConfigWithOptions(globalOptions)
	if err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(cfgman.ExitError)
	}

	// Show config source in verbose mode
	cfgman.PrintVerbose("Using configuration from: %s", configSource)

	if err := cfgman.Orphan(*path, globalOptions.RepoDir, config, *dryRun, globalYes); err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(cfgman.ExitError)
	}
}

func handleLink(args []string, globalOptions *cfgman.ConfigOptions) {
	fs := flag.NewFlagSet("link", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Preview changes without making them")

	fs.Usage = func() {
		fmt.Printf("%s cfgman link [options]\n", cfgman.Bold("Usage:"))
		fmt.Printf("\n%s\n", cfgman.Cyan("Create symlinks from repository to target directories"))
		fmt.Printf("\n%s\n", cfgman.Bold("Options:"))
		fmt.Print(formatFlags(fs))
		fmt.Printf("\n%s\n", cfgman.Bold("Examples:"))
		fmt.Println(cfgman.Cyan("  cfgman link"))
		fmt.Println(cfgman.Cyan("  cfgman link --dry-run"))
		fmt.Println(cfgman.Cyan("  cfgman link --repo-dir ~/dotfiles"))
		fmt.Printf("\n%s\n", cfgman.Bold("See also:"))
		fmt.Printf("  %s\n", cfgman.Cyan("unlink, status, adopt"))
	}

	fs.Parse(args)

	config, configSource, err := cfgman.LoadConfigWithOptions(globalOptions)
	if err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(cfgman.ExitError)
	}

	// Show config source in verbose mode
	cfgman.PrintVerbose("Using configuration from: %s", configSource)

	if err := cfgman.CreateLinks(globalOptions.RepoDir, config, *dryRun); err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(cfgman.ExitError)
	}
}

func handleUnlink(args []string, globalOptions *cfgman.ConfigOptions, globalYes bool) {
	fs := flag.NewFlagSet("unlink", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Preview changes without making them")

	fs.Usage = func() {
		fmt.Printf("%s cfgman unlink [options]\n", cfgman.Bold("Usage:"))
		fmt.Printf("\n%s\n", cfgman.Cyan("Remove all managed symlinks"))
		fmt.Printf("\n%s\n", cfgman.Bold("Options:"))
		fmt.Print(formatFlags(fs))
		fmt.Printf("\n%s\n", cfgman.Bold("Examples:"))
		fmt.Println(cfgman.Cyan("  cfgman unlink"))
		fmt.Println(cfgman.Cyan("  cfgman unlink --dry-run"))
		fmt.Println(cfgman.Cyan("  cfgman unlink --repo-dir ~/dotfiles"))
		fmt.Printf("\n%s\n", cfgman.Bold("See also:"))
		fmt.Printf("  %s\n", cfgman.Cyan("link, prune, orphan"))
	}

	fs.Parse(args)

	config, configSource, err := cfgman.LoadConfigWithOptions(globalOptions)
	if err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(cfgman.ExitError)
	}

	// Show config source in verbose mode
	cfgman.PrintVerbose("Using configuration from: %s", configSource)

	if err := cfgman.RemoveLinks(globalOptions.RepoDir, config, *dryRun, globalYes); err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(cfgman.ExitError)
	}
}

func handlePrune(args []string, globalOptions *cfgman.ConfigOptions, globalYes bool) {
	fs := flag.NewFlagSet("prune", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Preview changes without making them")

	fs.Usage = func() {
		fmt.Printf("%s cfgman prune [options]\n", cfgman.Bold("Usage:"))
		fmt.Printf("\n%s\n", cfgman.Cyan("Remove broken symlinks"))
		fmt.Printf("\n%s\n", cfgman.Bold("Options:"))
		fmt.Print(formatFlags(fs))
		fmt.Printf("\n%s\n", cfgman.Bold("Examples:"))
		fmt.Println(cfgman.Cyan("  cfgman prune"))
		fmt.Println(cfgman.Cyan("  cfgman prune --dry-run"))
		fmt.Println(cfgman.Cyan("  cfgman prune --repo-dir ~/dotfiles"))
		fmt.Printf("\n%s\n", cfgman.Bold("See also:"))
		fmt.Printf("  %s\n", cfgman.Cyan("unlink, status"))
	}

	fs.Parse(args)

	config, configSource, err := cfgman.LoadConfigWithOptions(globalOptions)
	if err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(cfgman.ExitError)
	}

	// Show config source in verbose mode
	cfgman.PrintVerbose("Using configuration from: %s", configSource)

	if err := cfgman.PruneLinks(globalOptions.RepoDir, config, *dryRun, globalYes); err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(cfgman.ExitError)
	}
}

func handleVersion(args []string) {
	fs := flag.NewFlagSet("version", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Printf("%s cfgman version [options]\n", cfgman.Bold("Usage:"))
		fmt.Printf("\n%s\n", cfgman.Cyan("Show version information"))
		fmt.Printf("\n%s\n", cfgman.Bold("Options:"))
		fmt.Print(formatFlags(fs))
		fmt.Printf("\n%s\n", cfgman.Bold("Examples:"))
		fmt.Println(cfgman.Cyan("  cfgman version"))
		fmt.Println(cfgman.Cyan("  cfgman --version"))
	}
	fs.Parse(args)

	cfgman.PrintInfo("%s %s", cfgman.Bold("cfgman version"), cfgman.Green(version))
	cfgman.PrintInfo("  commit: %s", cfgman.Cyan(commit))
	cfgman.PrintInfo("  built:  %s", cfgman.Cyan(date))
}

func printUsage() {
	fmt.Printf("%s cfgman [global options] <command> [options]\n", cfgman.Bold("Usage:"))
	fmt.Println()
	fmt.Println(cfgman.Bold("Global Options:"))
	fmt.Printf("  -v, --verbose        Enable verbose output\n")
	fmt.Printf("  -q, --quiet          Suppress all non-error output\n")
	fmt.Printf("  -y, --yes            Assume yes to all prompts\n")
	fmt.Printf("      --output FORMAT  Output format: text (default), json\n")
	fmt.Printf("      --no-color       Disable colored output\n")
	fmt.Printf("      --version        Show version information\n")
	fmt.Printf("  -h, --help           Show this help message\n")
	fmt.Println()
	fmt.Println(cfgman.Bold("Configuration Options:"))
	fmt.Printf("      --config PATH    Path to configuration file\n")
	fmt.Printf("      --repo-dir PATH  Repository directory (default: current directory)\n")
	fmt.Printf("      --source-dir DIR Override source directory for operations\n")
	fmt.Printf("      --target-dir DIR Override target directory for operations\n")
	fmt.Printf("      --ignore LIST    Comma-separated list of ignore patterns\n")
	fmt.Println()
	fmt.Println(cfgman.Bold("Commands:"))
	fmt.Printf("  %s\n", cfgman.Cyan("Link Management:"))
	fmt.Printf("    %-20s Show status of all managed symlinks\n", cfgman.Bold("status"))
	fmt.Printf("    %-20s Adopt file/directory into repository\n", cfgman.Bold("adopt"))
	fmt.Printf("    %-20s Remove file/directory from repo management\n", cfgman.Bold("orphan"))
	fmt.Printf("    %-20s Create symlinks from repo to target dirs\n", cfgman.Bold("link"))
	fmt.Printf("    %-20s Remove all managed symlinks\n", cfgman.Bold("unlink"))
	fmt.Printf("    %-20s Remove broken symlinks\n", cfgman.Bold("prune"))
	fmt.Println()
	fmt.Printf("  %s\n", cfgman.Cyan("Other:"))
	fmt.Printf("    %-20s Show version information\n", cfgman.Bold("version"))
	fmt.Printf("    %-20s Show help for a command\n", cfgman.Bold("help"))
	fmt.Println()
	fmt.Printf("Use '%s' for more information about a command.\n", cfgman.Bold("cfgman help <command>"))
	fmt.Println()
	fmt.Printf("%s\n", cfgman.Bold("Common workflow:"))
	fmt.Println(cfgman.Cyan("  cfgman adopt --path ~/.gitconfig --source-dir home     # Adopt existing files"))
	fmt.Println(cfgman.Cyan("  cfgman link                                             # Create symlinks"))
	fmt.Println(cfgman.Cyan("  cfgman status                                           # Check link status"))
	fmt.Println()
	fmt.Printf("%s\n", cfgman.Bold("Configuration Discovery:"))
	fmt.Println("  Configuration is loaded from the first available source:")
	fmt.Printf("    1. %s flag\n", cfgman.Cyan("--config"))
	fmt.Printf("    2. %s in repository directory\n", cfgman.Cyan(cfgman.ConfigFileName))
	fmt.Printf("    3. %s\n", cfgman.Cyan("$XDG_CONFIG_HOME/cfgman/config.json"))
	fmt.Printf("    4. %s\n", cfgman.Cyan("~/.config/cfgman/config.json"))
	fmt.Printf("    5. %s\n", cfgman.Cyan("~/.cfgman.json"))
	fmt.Printf("    6. %s in current directory\n", cfgman.Cyan(cfgman.ConfigFileName))
	fmt.Printf("    7. %s\n", cfgman.Cyan("Built-in defaults"))
	fmt.Println()
	fmt.Printf("%s\n", cfgman.Bold("Environment Variables:"))
	fmt.Printf("  %s      Configuration file path\n", cfgman.Cyan("CFGMAN_CONFIG"))
	fmt.Printf("  %s    Repository directory\n", cfgman.Cyan("CFGMAN_REPO_DIR"))
	fmt.Printf("  %s  Source directory override\n", cfgman.Cyan("CFGMAN_SOURCE_DIR"))
	fmt.Printf("  %s  Target directory override\n", cfgman.Cyan("CFGMAN_TARGET_DIR"))
	fmt.Printf("  %s       Ignore patterns (comma-separated)\n", cfgman.Cyan("CFGMAN_IGNORE"))
	fmt.Println()
	fmt.Printf("%s\n", cfgman.Bold("Examples:"))
	fmt.Println(cfgman.Cyan("  cfgman --repo-dir ~/dotfiles status                    # Use specific repo"))
	fmt.Println(cfgman.Cyan("  cfgman --config ~/.config/cfgman/work.json link       # Use specific config"))
	fmt.Println(cfgman.Cyan("  cfgman --source-dir work --target-dir ~/.config link  # Override directories"))
}

func printCommandHelp(command string) {
	// Create empty options for help display
	emptyOptions := &cfgman.ConfigOptions{}

	switch command {
	case "status":
		handleStatus([]string{"-h"}, emptyOptions)
	case "adopt":
		handleAdopt([]string{"-h"}, emptyOptions)
	case "orphan":
		handleOrphan([]string{"-h"}, emptyOptions, false)
	case "link":
		handleLink([]string{"-h"}, emptyOptions)
	case "unlink":
		handleUnlink([]string{"-h"}, emptyOptions, false)
	case "prune":
		handlePrune([]string{"-h"}, emptyOptions, false)
	case "version":
		handleVersion([]string{"-h"})
	default:
		suggestion := suggestCommand(command)
		if suggestion != "" {
			cfgman.PrintErrorWithHint(cfgman.WithHint(
				fmt.Errorf("unknown command: %s", command),
				fmt.Sprintf("Did you mean 'cfgman help %s'?", suggestion)))
		} else {
			cfgman.PrintErrorWithHint(cfgman.WithHint(
				fmt.Errorf("unknown command: %s", command),
				"Run 'cfgman --help' to see available commands"))
		}
	}
}
