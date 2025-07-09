// Package main provides the command-line interface for cfgman,
// a dotfile management tool that manages configuration files
// across machines using intelligent symlinks.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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

func main() {
	// Parse global flags first
	var globalVerbose, globalQuiet, globalJSON, globalNoColor, globalVersion, globalYes bool
	remainingArgs := []string{}

	// Manual parsing to extract global flags before command
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--verbose", "-v":
			globalVerbose = true
		case "--quiet", "-q":
			globalQuiet = true
		case "--json":
			globalJSON = true
		case "--no-color":
			globalNoColor = true
		case "--version":
			globalVersion = true
		case "--yes", "-y":
			globalYes = true
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
	if globalJSON {
		cfgman.SetOutputFormat(cfgman.FormatJSON)
		// JSON output implies quiet mode for non-data output
		if !globalVerbose {
			cfgman.SetVerbosity(cfgman.VerbosityQuiet)
		}
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

	// Route to command handler with remaining args
	commandArgs := remainingArgs[1:]
	switch command {
	case "status":
		handleStatus(commandArgs)
	case "adopt":
		handleAdopt(commandArgs)
	case "orphan":
		handleOrphan(commandArgs, globalYes)
	case "create":
		handleCreate(commandArgs)
	case "remove":
		handleRemove(commandArgs, globalYes)
	case "prune":
		handlePrune(commandArgs, globalYes)
	case "init":
		handleInit(commandArgs)
	case "version":
		handleVersion(commandArgs)
	default:
		cfgman.PrintErrorWithHint(cfgman.WithHint(
			fmt.Errorf("unknown command: %s", command),
			"Run 'cfgman --help' to see available commands"))
		os.Exit(cfgman.ExitUsage)
	}
}

func handleStatus(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Printf("%s cfgman status [options]\n", cfgman.Bold("Usage:"))
		fmt.Printf("\n%s\n", cfgman.Cyan("Show status of all managed symlinks"))
		fmt.Printf("\n%s\n", cfgman.Bold("Options:"))
		fmt.Print(formatFlags(fs))
		fmt.Printf("\n%s\n", cfgman.Bold("Examples:"))
		fmt.Println(cfgman.Cyan("  cfgman status"))
		fmt.Println(cfgman.Cyan("  cfgman status --json"))
		fmt.Printf("\n%s\n", cfgman.Bold("See also:"))
		fmt.Printf("  %s\n", cfgman.Cyan("create, prune"))
	}
	fs.Parse(args)

	config, err := cfgman.LoadConfig(".")
	if err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(cfgman.ExitError)
	}

	if err := cfgman.Status(".", config); err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(cfgman.ExitError)
	}
}

func handleAdopt(args []string) {
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
		fmt.Printf("\n%s\n", cfgman.Bold("See also:"))
		fmt.Printf("  %s\n", cfgman.Cyan("orphan, create, status"))
	}

	fs.Parse(args)

	if *path == "" || *sourceDir == "" {
		cfgman.PrintErrorWithHint(cfgman.WithHint(
			fmt.Errorf("both --path and --source-dir are required"),
			"Run 'cfgman adopt --help' for usage examples"))
		os.Exit(cfgman.ExitUsage)
	}

	config, err := cfgman.LoadConfig(".")
	if err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(cfgman.ExitError)
	}

	if err := cfgman.Adopt(*path, ".", config, *sourceDir, *dryRun); err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(cfgman.ExitError)
	}
}

func handleOrphan(args []string, globalYes bool) {
	fs := flag.NewFlagSet("orphan", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Preview changes without making them")
	force := fs.Bool("force", false, "Skip confirmation prompt")
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
	config, err := cfgman.LoadConfig(".")
	if err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(cfgman.ExitError)
	}

	if err := cfgman.Orphan(*path, ".", config, *dryRun, *force || globalYes); err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(cfgman.ExitError)
	}
}

func handleCreate(args []string) {
	fs := flag.NewFlagSet("create", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Preview changes without making them")

	fs.Usage = func() {
		fmt.Printf("%s cfgman create [options]\n", cfgman.Bold("Usage:"))
		fmt.Printf("\n%s\n", cfgman.Cyan("Create symlinks from repository to home directory"))
		fmt.Printf("\n%s\n", cfgman.Bold("Options:"))
		fmt.Print(formatFlags(fs))
		fmt.Printf("\n%s\n", cfgman.Bold("Examples:"))
		fmt.Println(cfgman.Cyan("  cfgman create"))
		fmt.Println(cfgman.Cyan("  cfgman create --dry-run"))
		fmt.Printf("\n%s\n", cfgman.Bold("See also:"))
		fmt.Printf("  %s\n", cfgman.Cyan("remove, status, adopt"))
	}

	fs.Parse(args)

	config, err := cfgman.LoadConfig(".")
	if err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(cfgman.ExitError)
	}

	if err := cfgman.CreateLinks(".", config, *dryRun); err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(cfgman.ExitError)
	}
}

func handleRemove(args []string, globalYes bool) {
	fs := flag.NewFlagSet("remove", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Preview changes without making them")
	force := fs.Bool("force", false, "Skip confirmation prompt")

	fs.Usage = func() {
		fmt.Printf("%s cfgman remove [options]\n", cfgman.Bold("Usage:"))
		fmt.Printf("\n%s\n", cfgman.Cyan("Remove all managed symlinks"))
		fmt.Printf("\n%s\n", cfgman.Bold("Options:"))
		fmt.Print(formatFlags(fs))
		fmt.Printf("\n%s\n", cfgman.Bold("Examples:"))
		fmt.Println(cfgman.Cyan("  cfgman remove"))
		fmt.Println(cfgman.Cyan("  cfgman remove --dry-run"))
		fmt.Printf("\n%s\n", cfgman.Bold("See also:"))
		fmt.Printf("  %s\n", cfgman.Cyan("create, prune, orphan"))
	}

	fs.Parse(args)

	config, err := cfgman.LoadConfig(".")
	if err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(cfgman.ExitError)
	}

	if err := cfgman.RemoveLinks(".", config, *dryRun, *force || globalYes); err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(cfgman.ExitError)
	}
}

func handlePrune(args []string, globalYes bool) {
	fs := flag.NewFlagSet("prune", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Preview changes without making them")
	force := fs.Bool("force", false, "Skip confirmation prompt")

	fs.Usage = func() {
		fmt.Printf("%s cfgman prune [options]\n", cfgman.Bold("Usage:"))
		fmt.Printf("\n%s\n", cfgman.Cyan("Remove broken symlinks"))
		fmt.Printf("\n%s\n", cfgman.Bold("Options:"))
		fmt.Print(formatFlags(fs))
		fmt.Printf("\n%s\n", cfgman.Bold("Examples:"))
		fmt.Println(cfgman.Cyan("  cfgman prune"))
		fmt.Println(cfgman.Cyan("  cfgman prune --dry-run"))
		fmt.Printf("\n%s\n", cfgman.Bold("See also:"))
		fmt.Printf("  %s\n", cfgman.Cyan("remove, status"))
	}

	fs.Parse(args)

	config, err := cfgman.LoadConfig(".")
	if err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(cfgman.ExitError)
	}

	if err := cfgman.PruneLinks(".", config, *dryRun, *force || globalYes); err != nil {
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

func handleInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	force := fs.Bool("force", false, "Overwrite existing configuration file")

	fs.Usage = func() {
		fmt.Printf("%s cfgman init [options]\n", cfgman.Bold("Usage:"))
		fmt.Printf("\n%s\n", cfgman.Cyan(fmt.Sprintf("Create a minimal %s configuration template", cfgman.ConfigFileName)))
		fmt.Printf("\n%s\n", cfgman.Bold("Options:"))
		fmt.Print(formatFlags(fs))
		fmt.Printf("\n%s\n", cfgman.Bold("This creates a template configuration file that you must edit to:"))
		fmt.Printf("  • Set the %s (e.g., 'home')\n", cfgman.Bold("source directory"))
		fmt.Printf("  • Set the %s (e.g., '~/')\n", cfgman.Bold("target directory"))
		fmt.Printf("  • Add any %s you need\n", cfgman.Bold("ignore patterns"))
		fmt.Printf("\n%s\n", cfgman.Bold("See also:"))
		fmt.Printf("  %s\n", cfgman.Cyan("adopt, create"))
	}

	fs.Parse(args)

	// Check if config already exists
	cfgmanPath := filepath.Join(".", cfgman.ConfigFileName)
	if !*force {
		if _, err := os.Stat(cfgmanPath); err == nil {
			cfgman.PrintErrorWithHint(cfgman.WithHint(
				fmt.Errorf("%s already exists", cfgman.ConfigFileName),
				"Use '--force' to overwrite the existing configuration"))
			os.Exit(cfgman.ExitError)
		}
	}

	// Create minimal config template
	defaultConfig := cfgman.Config{
		IgnorePatterns: []string{},
		LinkMappings: []cfgman.LinkMapping{
			{
				Source: "",
				Target: "",
			},
		},
	}

	// Write the config file
	data, err := json.MarshalIndent(defaultConfig, "", "  ")
	if err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(cfgman.ExitError)
	}

	if err := os.WriteFile(cfgmanPath, data, 0644); err != nil {
		cfgman.PrintErrorWithHint(cfgman.WithHint(
			fmt.Errorf("failed to write %s: %v", cfgman.ConfigFileName, err),
			"Check that you have write permissions in this directory"))
		os.Exit(cfgman.ExitError)
	}

	cfgman.PrintSuccess("Created %s with a minimal template.", cfgman.ConfigFileName)
	fmt.Printf("\n%s %s to configure:\n", cfgman.Bold("You must edit"), cfgman.Cyan(cfgman.ConfigFileName))
	fmt.Printf("  %s The directory in your repo containing config files (e.g., 'home')\n", cfgman.Bold("source:"))
	fmt.Printf("  %s Where to link files to (e.g., '~/')\n", cfgman.Bold("target:"))
	fmt.Printf("  %s Files/patterns to ignore (e.g., '.DS_Store', '*.swp')\n", cfgman.Bold("ignore_patterns:"))
	fmt.Printf("\n%s\n", cfgman.Bold("Example configuration:"))
	fmt.Println(cfgman.Cyan("  {"))
	fmt.Println(cfgman.Cyan("    \"ignore_patterns\": [\".DS_Store\", \"*.swp\"],"))
	fmt.Println(cfgman.Cyan("    \"link_mappings\": [{"))
	fmt.Println(cfgman.Cyan("      \"source\": \"home\","))
	fmt.Println(cfgman.Cyan("      \"target\": \"~/\""))
	fmt.Println(cfgman.Cyan("    }]"))
	fmt.Println(cfgman.Cyan("  }"))
	fmt.Printf("\n%s After editing the config file:\n", cfgman.Bold("Next steps:"))
	fmt.Printf("  %s\n", cfgman.Cyan("cfgman adopt --path ~/.gitconfig --source-dir home"))
	fmt.Printf("  %s\n", cfgman.Cyan("cfgman create                                      # Create symlinks"))
}

func printUsage() {
	fmt.Printf("%s cfgman [global options] <command> [options]\n", cfgman.Bold("Usage:"))
	fmt.Println()
	fmt.Println(cfgman.Bold("Global Options:"))
	fmt.Printf("  -v, --verbose        Enable verbose output\n")
	fmt.Printf("  -q, --quiet          Suppress all non-error output\n")
	fmt.Printf("  -y, --yes            Assume yes to all prompts\n")
	fmt.Printf("      --json           Output in JSON format (where supported)\n")
	fmt.Printf("      --no-color       Disable colored output\n")
	fmt.Printf("      --version        Show version information\n")
	fmt.Printf("  -h, --help           Show this help message\n")
	fmt.Println()
	fmt.Println(cfgman.Bold("Commands:"))
	fmt.Printf("  %s\n", cfgman.Cyan("Configuration:"))
	fmt.Printf("    %-20s Create a minimal %s template\n", cfgman.Bold("init"), cfgman.ConfigFileName)
	fmt.Println()
	fmt.Printf("  %s\n", cfgman.Cyan("Link Management:"))
	fmt.Printf("    %-20s Show status of all managed symlinks\n", cfgman.Bold("status"))
	fmt.Printf("    %-20s Adopt file/directory into repository\n", cfgman.Bold("adopt"))
	fmt.Printf("    %-20s Remove file/directory from repo management\n", cfgman.Bold("orphan"))
	fmt.Printf("    %-20s Create symlinks from repo to home\n", cfgman.Bold("create"))
	fmt.Printf("    %-20s Remove all managed symlinks\n", cfgman.Bold("remove"))
	fmt.Printf("    %-20s Remove broken symlinks\n", cfgman.Bold("prune"))
	fmt.Println()
	fmt.Printf("  %s\n", cfgman.Cyan("Other:"))
	fmt.Printf("    %-20s Show version information\n", cfgman.Bold("version"))
	fmt.Printf("    %-20s Show help for a command\n", cfgman.Bold("help"))
	fmt.Println()
	fmt.Printf("Use '%s' for more information about a command.\n", cfgman.Bold("cfgman help <command>"))
	fmt.Println()
	fmt.Printf("%s\n", cfgman.Bold("Common workflow:"))
	fmt.Println(cfgman.Cyan("  cfgman init                                             # Create configuration template"))
	fmt.Println(cfgman.Cyan("  cfgman adopt --path ~/.gitconfig --source-dir home     # Adopt existing files"))
	fmt.Println(cfgman.Cyan("  cfgman create                                           # Create symlinks"))
	fmt.Println(cfgman.Cyan("  cfgman status                                           # Check link status"))
	fmt.Println()
	fmt.Printf("%s cfgman must be run from within a cfgman-managed directory\n", cfgman.Bold("Note:"))
	fmt.Printf("      (a directory containing %s)\n", cfgman.Cyan(cfgman.ConfigFileName))
}

func printCommandHelp(command string) {
	switch command {
	case "status":
		handleStatus([]string{"-h"})
	case "adopt":
		handleAdopt([]string{"-h"})
	case "orphan":
		handleOrphan([]string{"-h"}, false)
	case "create":
		handleCreate([]string{"-h"})
	case "remove":
		handleRemove([]string{"-h"}, false)
	case "prune":
		handlePrune([]string{"-h"}, false)
	case "init":
		handleInit([]string{"-h"})
	case "version":
		handleVersion([]string{"-h"})
	default:
		cfgman.PrintErrorWithHint(cfgman.WithHint(
			fmt.Errorf("unknown command: %s", command),
			"Run 'cfgman --help' to see available commands"))
	}
}
