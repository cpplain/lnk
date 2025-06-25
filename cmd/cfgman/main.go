package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/cpplain/cfgman/internal/cfgman"
)

// Version variables set via ldflags during build
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	// Handle --version flag before any other parsing
	for _, arg := range os.Args[1:] {
		if arg == "--version" {
			fmt.Printf("cfgman version %s\n", version)
			return
		}
	}

	// No global flags needed anymore - cfgman works from current directory
	remainingArgs := os.Args[1:]

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
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func handleStatus(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Println("Usage: cfgman status")
		fmt.Println("\nShow status of all managed symlinks")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	config, err := cfgman.LoadConfig(".")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	if err := cfgman.Status(".", config); err != nil {
		log.Fatal(err)
	}
}

func handleAdopt(args []string) {
	fs := flag.NewFlagSet("adopt", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Preview changes without making them")

	fs.Usage = func() {
		fmt.Println("Usage: cfgman adopt [options] PATH [SOURCE_DIR]")
		fmt.Println("\nAdopt a file or directory into the repository")
		fmt.Println("\nArguments:")
		fmt.Println("  PATH        The file or directory to adopt")
		fmt.Println("  SOURCE_DIR  The source directory in the repository (e.g., home, private/home)")
		fmt.Println("              If not provided, you will be prompted to enter it")
		fmt.Println("\nOptions:")
		fs.PrintDefaults()
		fmt.Println("\nExamples:")
		fmt.Println("  cfgman adopt ~/.gitconfig home")
		fmt.Println("  cfgman adopt ~/.ssh/config private/home")
		fmt.Println("  cfgman adopt ~/.zshrc  # Will prompt for source directory")
	}

	fs.Parse(args)

	if fs.NArg() < 1 || fs.NArg() > 2 {
		fs.Usage()
		os.Exit(1)
	}

	path := fs.Arg(0)
	var sourceDir string

	// Get source directory from args or prompt
	if fs.NArg() == 2 {
		sourceDir = fs.Arg(1)
	} else {
		var err error
		sourceDir, err = cfgman.ReadUserInputWithDefault("Enter source directory (e.g., home, private/home, work)", "home")
		if err != nil {
			log.Fatalf("Error reading input: %v", err)
		}
	}

	config, err := cfgman.LoadConfig(".")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	if err := cfgman.Adopt(path, ".", config, sourceDir, *dryRun); err != nil {
		log.Fatal(err)
	}
}

func handleOrphan(args []string) {
	fs := flag.NewFlagSet("orphan", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Preview changes without making them")

	fs.Usage = func() {
		fmt.Println("Usage: cfgman orphan [options] PATH")
		fmt.Println("\nRemove a file or directory from repository management")
		fmt.Println("For directories, recursively orphans all managed symlinks within")
		fmt.Println("\nOptions:")
		fs.PrintDefaults()
	}

	fs.Parse(args)

	if fs.NArg() != 1 {
		fs.Usage()
		os.Exit(1)
	}

	path := fs.Arg(0)
	config, err := cfgman.LoadConfig(".")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	if err := cfgman.Orphan(path, ".", config, *dryRun); err != nil {
		log.Fatal(err)
	}
}

func handleCreateLinks(args []string) {
	fs := flag.NewFlagSet("create-links", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Preview changes without making them")

	fs.Usage = func() {
		fmt.Println("Usage: cfgman create-links [options]")
		fmt.Println("\nCreate symlinks from repository to home directory")
		fmt.Println("\nOptions:")
		fs.PrintDefaults()
	}

	fs.Parse(args)

	config, err := cfgman.LoadConfig(".")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	if err := cfgman.CreateLinks(".", config, *dryRun); err != nil {
		log.Fatal(err)
	}
}

func handleRemoveLinks(args []string) {
	fs := flag.NewFlagSet("remove-links", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Preview changes without making them")

	fs.Usage = func() {
		fmt.Println("Usage: cfgman remove-links [options]")
		fmt.Println("\nRemove all managed symlinks")
		fmt.Println("\nOptions:")
		fs.PrintDefaults()
	}

	fs.Parse(args)

	config, err := cfgman.LoadConfig(".")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	if err := cfgman.RemoveLinks(".", config, *dryRun); err != nil {
		log.Fatal(err)
	}
}

func handlePruneLinks(args []string) {
	fs := flag.NewFlagSet("prune-links", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Preview changes without making them")

	fs.Usage = func() {
		fmt.Println("Usage: cfgman prune-links [options]")
		fmt.Println("\nRemove broken symlinks")
		fmt.Println("\nOptions:")
		fs.PrintDefaults()
	}

	fs.Parse(args)

	config, err := cfgman.LoadConfig(".")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	if err := cfgman.PruneLinks(".", config, *dryRun); err != nil {
		log.Fatal(err)
	}
}

func handleVersion(args []string) {
	fs := flag.NewFlagSet("version", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Println("Usage: cfgman version")
		fmt.Println("\nShow version information")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	fmt.Printf("cfgman version %s\n", version)
	fmt.Printf("  commit: %s\n", commit)
	fmt.Printf("  built:  %s\n", date)
}

func handleInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	force := fs.Bool("force", false, "Overwrite existing configuration file")

	fs.Usage = func() {
		fmt.Println("Usage: cfgman init [options]")
		fmt.Println("\nCreate a minimal .cfgman.json configuration template")
		fmt.Println("\nOptions:")
		fs.PrintDefaults()
		fmt.Println("\nThis creates a template configuration file that you must edit to:")
		fmt.Println("  - Set the source directory (e.g., 'home')")
		fmt.Println("  - Set the target directory (e.g., '~/')")
		fmt.Println("  - Add any ignore patterns you need")
		fmt.Println("  - Add directories to link_as_directory if needed")
	}

	fs.Parse(args)

	// Check if config already exists
	cfgmanPath := filepath.Join(".", ".cfgman.json")
	if !*force {
		if _, err := os.Stat(cfgmanPath); err == nil {
			fmt.Println("Error: .cfgman.json already exists. Use --force to overwrite.")
			os.Exit(1)
		}
	}

	// Create minimal config template
	defaultConfig := cfgman.Config{
		IgnorePatterns: []string{},
		LinkMappings: []cfgman.LinkMapping{
			{
				Source:          "",
				Target:          "",
				LinkAsDirectory: []string{},
			},
		},
	}

	// Write the config file
	data, err := json.MarshalIndent(defaultConfig, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling config: %v", err)
	}

	if err := os.WriteFile(cfgmanPath, data, 0644); err != nil {
		log.Fatalf("Error writing .cfgman.json: %v", err)
	}

	fmt.Println("Created .cfgman.json with a minimal template.")
	fmt.Println("\nYou must edit .cfgman.json to configure:")
	fmt.Println("  - source: The directory in your repo containing config files (e.g., 'home')")
	fmt.Println("  - target: Where to link files to (e.g., '~/')")
	fmt.Println("  - ignore_patterns: Files/patterns to ignore (e.g., '.DS_Store', '*.swp')")
	fmt.Println("  - link_as_directory: Directories to link as whole directories instead of individual files")
	fmt.Println("\nExample configuration:")
	fmt.Println("  {")
	fmt.Println("    \"ignore_patterns\": [\".DS_Store\", \"*.swp\"],")
	fmt.Println("    \"link_mappings\": [{")
	fmt.Println("      \"source\": \"home\",")
	fmt.Println("      \"target\": \"~/\",")
	fmt.Println("      \"link_as_directory\": [\".config/nvim\"]")
	fmt.Println("    }]")
	fmt.Println("  }")
}

func printUsage() {
	fmt.Println("Usage: cfgman <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  Configuration:")
	fmt.Println("    init                Create a minimal .cfgman.json template")
	fmt.Println()
	fmt.Println("  Link Management:")
	fmt.Println("    status              Show status of all managed symlinks")
	fmt.Println("    adopt PATH [SOURCE_DIR]")
	fmt.Println("                        Adopt file/directory into repository")
	fmt.Println("    orphan [--dry-run] PATH")
	fmt.Println("                        Remove file/directory from repo management")
	fmt.Println("    create-links        Create symlinks from repo to home")
	fmt.Println("    remove-links        Remove all managed symlinks")
	fmt.Println("    prune-links         Remove broken symlinks")
	fmt.Println()
	fmt.Println("  Other:")
	fmt.Println("    version             Show version information")
	fmt.Println("    help [command]      Show help for a command")
	fmt.Println()
	fmt.Println("Use 'cfgman help <command>' for more information about a command.")
	fmt.Println()
	fmt.Println("Note: cfgman must be run from within a cfgman-managed directory")
	fmt.Println("      (a directory containing .cfgman.json)")
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
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
	}
}
