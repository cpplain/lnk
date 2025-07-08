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
		fmt.Fprintf(&b, "  --%s\t%s\n", f.Name, f.Usage)
		count++
	})
	if count == 0 {
		return "  (none)\n"
	}
	return b.String()
}

func main() {
	// Parse global flags first
	var globalVerbose, globalQuiet, globalJSON, globalNoColor, globalVersion bool
	remainingArgs := []string{}

	// Manual parsing to extract global flags before command
	skipNext := false
	for _, arg := range os.Args[1:] {
		if skipNext {
			skipNext = false
			continue
		}

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
		os.Exit(1)
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
		os.Exit(1)
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
		handleOrphan(commandArgs)
	case "create-links":
		handleCreateLinks(commandArgs)
	case "remove-links":
		handleRemoveLinks(commandArgs)
	case "prune-links":
		handlePruneLinks(commandArgs)
	case "init":
		handleInit(commandArgs)
	case "version":
		handleVersion(commandArgs)
	default:
		cfgman.PrintErrorWithHint(cfgman.WithHint(
			fmt.Errorf("unknown command: %s", command),
			"Run 'cfgman --help' to see available commands"))
		os.Exit(1)
	}
}

func handleStatus(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Printf("%s cfgman status [options]\n", cfgman.Bold("Usage:"))
		fmt.Printf("\n%s\n", cfgman.Cyan("Show status of all managed symlinks"))
		fmt.Printf("\n%s\n", cfgman.Bold("Options:"))
		fmt.Print(formatFlags(fs))
	}
	fs.Parse(args)

	config, err := cfgman.LoadConfig(".")
	if err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(1)
	}

	if err := cfgman.Status(".", config); err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(1)
	}
}

func handleAdopt(args []string) {
	fs := flag.NewFlagSet("adopt", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Preview changes without making them")

	fs.Usage = func() {
		fmt.Printf("%s cfgman adopt [options] PATH SOURCE_DIR\n", cfgman.Bold("Usage:"))
		fmt.Printf("\n%s\n", cfgman.Cyan("Adopt a file or directory into the repository"))
		fmt.Printf("\n%s\n", cfgman.Bold("Arguments:"))
		fmt.Printf("  %-20s The file or directory to adopt\n", cfgman.Bold("PATH"))
		fmt.Printf("  %-20s The source directory in the repository (e.g., home, private/home)\n", cfgman.Bold("SOURCE_DIR"))
		fmt.Printf("\n%s\n", cfgman.Bold("Options:"))
		fmt.Print(formatFlags(fs))
		fmt.Printf("\n%s\n", cfgman.Bold("Examples:"))
		fmt.Println(cfgman.Cyan("  cfgman adopt ~/.gitconfig home"))
		fmt.Println(cfgman.Cyan("  cfgman adopt ~/.ssh/config private/home"))
	}

	fs.Parse(args)

	if fs.NArg() != 2 {
		fs.Usage()
		os.Exit(1)
	}

	path := fs.Arg(0)
	sourceDir := fs.Arg(1)

	config, err := cfgman.LoadConfig(".")
	if err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(1)
	}

	if err := cfgman.Adopt(path, ".", config, sourceDir, *dryRun); err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(1)
	}
}

func handleOrphan(args []string) {
	fs := flag.NewFlagSet("orphan", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Preview changes without making them")

	fs.Usage = func() {
		fmt.Printf("%s cfgman orphan [options] PATH\n", cfgman.Bold("Usage:"))
		fmt.Printf("\n%s\n", cfgman.Cyan("Remove a file or directory from repository management"))
		fmt.Println("For directories, recursively orphans all managed symlinks within")
		fmt.Printf("\n%s\n", cfgman.Bold("Arguments:"))
		fmt.Printf("  %-20s The file or directory to orphan\n", cfgman.Bold("PATH"))
		fmt.Printf("\n%s\n", cfgman.Bold("Options:"))
		fmt.Print(formatFlags(fs))
		fmt.Printf("\n%s\n", cfgman.Bold("Examples:"))
		fmt.Println(cfgman.Cyan("  cfgman orphan ~/.gitconfig"))
		fmt.Println(cfgman.Cyan("  cfgman orphan ~/.config/nvim"))
	}

	fs.Parse(args)

	if fs.NArg() != 1 {
		fs.Usage()
		os.Exit(1)
	}

	path := fs.Arg(0)
	config, err := cfgman.LoadConfig(".")
	if err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(1)
	}

	if err := cfgman.Orphan(path, ".", config, *dryRun); err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(1)
	}
}

func handleCreateLinks(args []string) {
	fs := flag.NewFlagSet("create-links", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Preview changes without making them")

	fs.Usage = func() {
		fmt.Printf("%s cfgman create-links [options]\n", cfgman.Bold("Usage:"))
		fmt.Printf("\n%s\n", cfgman.Cyan("Create symlinks from repository to home directory"))
		fmt.Printf("\n%s\n", cfgman.Bold("Options:"))
		fmt.Print(formatFlags(fs))
		fmt.Printf("\n%s\n", cfgman.Bold("Examples:"))
		fmt.Println(cfgman.Cyan("  cfgman create-links"))
		fmt.Println(cfgman.Cyan("  cfgman create-links --dry-run"))
	}

	fs.Parse(args)

	config, err := cfgman.LoadConfig(".")
	if err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(1)
	}

	if err := cfgman.CreateLinks(".", config, *dryRun); err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(1)
	}
}

func handleRemoveLinks(args []string) {
	fs := flag.NewFlagSet("remove-links", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Preview changes without making them")

	fs.Usage = func() {
		fmt.Printf("%s cfgman remove-links [options]\n", cfgman.Bold("Usage:"))
		fmt.Printf("\n%s\n", cfgman.Cyan("Remove all managed symlinks"))
		fmt.Printf("\n%s\n", cfgman.Bold("Options:"))
		fmt.Print(formatFlags(fs))
		fmt.Printf("\n%s\n", cfgman.Bold("Examples:"))
		fmt.Println(cfgman.Cyan("  cfgman remove-links"))
		fmt.Println(cfgman.Cyan("  cfgman remove-links --dry-run"))
	}

	fs.Parse(args)

	config, err := cfgman.LoadConfig(".")
	if err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(1)
	}

	if err := cfgman.RemoveLinks(".", config, *dryRun); err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(1)
	}
}

func handlePruneLinks(args []string) {
	fs := flag.NewFlagSet("prune-links", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Preview changes without making them")

	fs.Usage = func() {
		fmt.Printf("%s cfgman prune-links [options]\n", cfgman.Bold("Usage:"))
		fmt.Printf("\n%s\n", cfgman.Cyan("Remove broken symlinks"))
		fmt.Printf("\n%s\n", cfgman.Bold("Options:"))
		fmt.Print(formatFlags(fs))
		fmt.Printf("\n%s\n", cfgman.Bold("Examples:"))
		fmt.Println(cfgman.Cyan("  cfgman prune-links"))
		fmt.Println(cfgman.Cyan("  cfgman prune-links --dry-run"))
	}

	fs.Parse(args)

	config, err := cfgman.LoadConfig(".")
	if err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(1)
	}

	if err := cfgman.PruneLinks(".", config, *dryRun); err != nil {
		cfgman.PrintErrorWithHint(err)
		os.Exit(1)
	}
}

func handleVersion(args []string) {
	fs := flag.NewFlagSet("version", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Printf("%s cfgman version [options]\n", cfgman.Bold("Usage:"))
		fmt.Printf("\n%s\n", cfgman.Cyan("Show version information"))
		fmt.Printf("\n%s\n", cfgman.Bold("Options:"))
		fmt.Print(formatFlags(fs))
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
	}

	fs.Parse(args)

	// Check if config already exists
	cfgmanPath := filepath.Join(".", cfgman.ConfigFileName)
	if !*force {
		if _, err := os.Stat(cfgmanPath); err == nil {
			cfgman.PrintErrorWithHint(cfgman.WithHint(
				fmt.Errorf("%s already exists", cfgman.ConfigFileName),
				"Use '--force' to overwrite the existing configuration"))
			os.Exit(1)
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
		os.Exit(1)
	}

	if err := os.WriteFile(cfgmanPath, data, 0644); err != nil {
		cfgman.PrintErrorWithHint(cfgman.WithHint(
			fmt.Errorf("failed to write %s: %v", cfgman.ConfigFileName, err),
			"Check that you have write permissions in this directory"))
		os.Exit(1)
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
}

func printUsage() {
	fmt.Printf("%s cfgman [global options] <command> [options]\n", cfgman.Bold("Usage:"))
	fmt.Println()
	fmt.Println(cfgman.Bold("Global Options:"))
	fmt.Printf("  -v, --verbose        Enable verbose output\n")
	fmt.Printf("  -q, --quiet          Suppress all non-error output\n")
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
	fmt.Printf("    %-20s Create symlinks from repo to home\n", cfgman.Bold("create-links"))
	fmt.Printf("    %-20s Remove all managed symlinks\n", cfgman.Bold("remove-links"))
	fmt.Printf("    %-20s Remove broken symlinks\n", cfgman.Bold("prune-links"))
	fmt.Println()
	fmt.Printf("  %s\n", cfgman.Cyan("Other:"))
	fmt.Printf("    %-20s Show version information\n", cfgman.Bold("version"))
	fmt.Printf("    %-20s Show help for a command\n", cfgman.Bold("help"))
	fmt.Println()
	fmt.Printf("Use '%s' for more information about a command.\n", cfgman.Bold("cfgman help <command>"))
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
		handleOrphan([]string{"-h"})
	case "create-links":
		handleCreateLinks([]string{"-h"})
	case "remove-links":
		handleRemoveLinks([]string{"-h"})
	case "prune-links":
		handlePruneLinks([]string{"-h"})
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
