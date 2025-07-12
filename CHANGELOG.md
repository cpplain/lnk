# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- **BREAKING: Reverted command names** - Renamed commands back to their original names:
  - `link` command renamed back to `create` for creating symlinks
  - `unlink` command renamed back to `remove` for removing symlinks
  - Commands now use simple verbs that are clear in context
- **Improved help documentation UX** - Standardized help output formatting:
  - Removed excessive colors and formatting from help text
  - Fixed inconsistent spacing and alignment across all help sections
  - Commands now appear before options (following standard CLI conventions)
  - Removed `help` command from command list (use `lnk <command> --help` syntax)
  - All help output now uses consistent helper functions for proper alignment
  - Examples section simplified to show commands without descriptions

### Fixed

- **Test environment isolation** - Fixed failing tests that were loading system configuration files:
  - Added proper environment variable isolation for `XDG_CONFIG_HOME` and `HOME`
  - Tests now run reliably regardless of developer's local configuration
  - Ensures consistent test results across different environments

### Added

- **Command improvements** - Enhanced command structure for clarity:
  - Added "did you mean?" suggestions for mistyped commands
- **Flag improvements** - Enhanced flag handling for better usability:
  - Removed `--force` flags from individual commands in favor of global `--yes` flag
  - Replaced `--json` flag with `--output FORMAT` flag (supports text and json formats)
  - Added support for `--flag=value` syntax in addition to `--flag value`
  - Improved flag consistency across all commands
  - **BREAKING**: Removed `--repo-dir` flag and `LNK_REPO_DIR` environment variable
- **BREAKING: Directory-based architecture** - lnk now uses absolute paths for source directories:
  - Changed from repository-based to directory-based tool
  - Source directories in config must now be absolute paths (e.g., `~/dotfiles/home` instead of `home`)
  - Adopt command now requires absolute path for --source-dir (e.g., `--source-dir ~/dotfiles/home`)
  - Removed concept of "repository directory" - lnk can be run from anywhere
  - Each source directory is independent - you can manage configs from multiple locations
- **Works without configuration** - lnk works from any directory without requiring a config file:
  - Added configuration discovery system with precedence order: --config flag > XDG config > user config > home > current dir > built-in defaults
  - Added global configuration flags: --config, --source-dir, --target-dir, --ignore
  - Added environment variable support: LNK_CONFIG, LNK_SOURCE_DIR, LNK_TARGET_DIR, LNK_IGNORE
  - Added built-in sensible defaults (~/dotfiles/home->~/, ~/dotfiles/config->~/.config/, common ignore patterns)
  - Added XDG Base Directory Specification support ($XDG_CONFIG_HOME/lnk/config.json)
  - All commands now work with flexible configuration loading and override system
  - Updated help text to document new configuration discovery and examples
- **Enhanced user guidance** - Added next-step suggestions after successful operations:
  - `adopt` command suggests running `create` to establish symlinks
  - `create` command suggests running `status` to verify links
  - `remove` command suggests recreating links or checking status
  - `prune` command suggests checking remaining links with status
  - `orphan` command suggests checking remaining managed files
  - Follows CLI design principle of being "Helpful Over Minimal"
- CLI design documentation following [cpplain/cli-design](https://github.com/cpplain/cli-design) principles
  - Added CLI Design Guidelines section to CONTRIBUTING.md for contributors
  - Added CLI Design Principles section to CLAUDE.md for implementation guidance
  - Documents four core principles: Obvious Over Clever, Helpful Over Minimal, Consistent Over Special, Human-First Machine-Friendly
- **Global verbosity control** - Added `--verbose`/`-v` and `--quiet`/`-q` global flags:
  - `--verbose` enables detailed debug output showing internal operations
  - `--quiet` suppresses all non-error output for scripting
  - Flags are mutually exclusive and work with all commands
  - Verbose mode includes configuration loading details and operation progress
- **JSON output support** - Added `--json` flag for machine-readable output:
  - Currently supported by the `status` command
  - Outputs structured JSON with links array and summary statistics
  - Automatically enables quiet mode to ensure clean JSON output
  - Includes link details (path, target, broken status, source mapping)
- **Progress indicators** - Added progress indicators for operations that may take more than 1 second:
  - Spinner animation with file counts for long-running operations
  - Automatically appears after 1 second (following CLI best practices)
  - Shows progress for: searching managed links (status/remove/prune), creating multiple symlinks, adopting directories
  - Respects terminal detection, quiet mode, and JSON output settings
- **Color control** - Added `--no-color` flag to disable colored output:
  - Works as a global flag alongside existing NO_COLOR environment variable support
  - Flag takes precedence over environment variable for explicit control
  - Useful for CI/CD environments and output parsing
- **Enhanced error messages** - All errors now include actionable "Try:" suggestions:
  - Configuration errors guide users to create config files or use built-in defaults
  - File conflicts suggest using `lnk adopt` first
  - Invalid paths show correct format examples
  - Unknown commands direct to help documentation
  - Leverages existing hint infrastructure throughout the codebase
- **Confirmation prompts for destructive operations** - Added interactive confirmation prompts:
  - `orphan`, `remove`, and `prune` now ask for confirmation before proceeding
  - Prompts clearly indicate what will be affected (e.g., "This will remove 3 symlink(s). Continue? (y/N): ")
  - Default answer is "No" for safety
  - Added `--force` flag to all three commands to skip confirmation prompts
  - Automatically skips prompts when not in a terminal (safe for scripts)
- **Global --yes flag** - Added `--yes`/`-y` global flag for automation:
  - Works with all destructive commands (orphan, remove, prune)
  - Assumes "yes" to all confirmation prompts
  - Can be used instead of or in addition to command-specific `--force` flags
  - Follows common CLI conventions (similar to apt-get -y, npm -y)
- **Default value support in help text** - Enhanced flag help formatting:
  - Updated `formatFlags()` function to display default values when meaningful
  - Boolean flags that default to false don't show defaults (implied behavior)
  - Prepared infrastructure for future non-boolean flags to show defaults
  - Maintains clean, uncluttered help text for current boolean-only flags
- **"See also" sections in command help** - Added cross-references between related commands:
  - Each command's help text now includes a "See also" section listing related commands
  - Helps users discover complementary functionality (e.g., adopt ↔ orphan, create ↔ remove)
  - Improves command discoverability and navigation
  - Follows CLI best practices for help text organization
- **Automatic output format adaptation for piping** - Output adapts based on TTY detection:
  - When output is piped, automatically switches to simplified, parseable format
  - Interactive terminal: Uses icons (✓, ✗, !) and colors for human-friendly output
  - Piped output: Uses simple text markers (success, error, warning) for easy parsing
  - Status command outputs `active <path>` or `broken <path>` format when piped
  - Makes lnk compatible with grep, awk, and other text processing tools
  - JSON output (--json) takes precedence over automatic adaptation
  - Follows CLI best practices for human-first, machine-friendly design
- **Specific exit codes for different error types** - Following GNU/POSIX conventions:
  - Exit code 0: Success
  - Exit code 1: General runtime errors (file operations, config issues, etc.)
  - Exit code 2: Command usage errors (unknown command, wrong arguments, invalid flags)
  - Makes it easier for scripts to distinguish between usage and operational errors
  - Follows standard CLI conventions for better automation support

### Changed

- **BREAKING: Command argument structure** - Replaced positional arguments with explicit flags to follow CLI design best practices:
  - `adopt PATH SOURCE_DIR` → `adopt --path PATH --source-dir SOURCE_DIR`
  - `orphan PATH` → `orphan --path PATH`
  - Follows "flags over positional arguments" principle for better usability
  - Provides clearer error messages when required flags are missing
  - All help text and workflow examples updated to reflect new syntax
- **BREAKING: Standardized command naming** - Renamed hyphenated commands to single-word verbs:
  - `create-links` → `create` (context makes it clear we're creating symlinks)
  - `remove-links` → `remove` (context makes it clear we're removing symlinks)
  - `prune-links` → `prune` (already implies removing broken symlinks)
  - All commands now follow consistent single-word verb pattern
  - Simplifies command usage and improves consistency
  - Updated all documentation and help text to reflect new names
- **Major refactoring for better maintainability** - Simplified codebase architecture:
  - Removed logger abstraction (`logger.go`) in favor of direct output functions
  - Removed unnecessary interface abstractions (`interfaces.go`) for cleaner, more idiomatic Go code
  - Added centralized output helpers (`output.go`) for consistent formatting
  - Added path utilities (`path_utils.go`) for home directory contraction
- **Improved user messages** - Replaced log-style output with user-friendly CLI messages:
  - Error messages now show as "Error: description" without timestamps
  - Success messages use visual indicators (✓) for completed actions
  - Warning messages are clearly marked without log prefixes
  - Debug output only shown when `CFGMAN_DEBUG` environment variable is set
- **Better status output** - Status command now uses tabwriter for aligned table format with summary statistics
- **Cleaner error handling** - All errors in main.go now use consistent format without log.Fatal timestamps
- **Consistent home directory display** - All paths shown to users now consistently display home directory as `~` instead of the full path
- **Cleaner orphan output** - Removed redundant initial file listing; progress is shown as files are processed
- **Simplified adopt output** - Adopt command now uses single "✓ Adopted: <path>" line per file, matching the pattern of other commands
- **Standardized all output helpers** - All commands now use consistent output helper functions (PrintError, PrintSuccess, PrintWarning, etc.) for uniform formatting across the entire CLI
- **Simplified status output** - Status command now uses the same single-line format as other commands (e.g., "✓ Active: ~/.bashrc") for consistency
- **Standardized error messages** - All error messages now follow consistent "Failed to <action>: <reason>" format with lowercase actions for better readability
- **Enhanced error context** - Error messages now include actionable suggestions where appropriate (e.g., "Use 'lnk adopt' to adopt this file first", "Create a config file or use built-in defaults")
- **Added summary output for bulk operations** - Commands that operate on multiple files now show clear summaries:
  - `create` shows "Created X symlink(s) successfully" and failure counts
  - `remove` shows "Removed X symlink(s) successfully" and failure counts
  - `prune` shows "Pruned X broken symlink(s) successfully" and failure counts
  - `adopt` (directories) shows "Successfully adopted X file(s)" with skip counts
  - `orphan` (directories) shows "Successfully orphaned X file(s)"
- **Improved skip messages** - Skip messages now clearly explain why files were skipped (e.g., "file already exists in repository at <path>")
- **Enhanced help output formatting** - Improved help text structure and formatting:
  - All command help uses consistent structure: Usage, Description, Arguments (if applicable), Options, Examples (if applicable)
  - Command descriptions use cyan color for better readability
  - Arguments and options are properly aligned with bold formatting
  - Added "(none)" placeholder when commands have no options
  - Main usage output organized into clear sections: Link Management, Other

### Removed

- **BREAKING: Removed `init` command** - Following advanced CLI tool patterns, lnk no longer provides config file creation:
  - Advanced users can create `.lnk.json` manually when customization is needed
  - lnk works with built-in defaults and doesn't require a config file
  - Removed all `init` command references from help text and documentation
  - Updated error messages to suggest using built-in defaults instead of init
  - Simplified CLI interface by removing unnecessary config template generation

### Fixed

- **CLI design conformance improvements** - Enhanced CLI to better follow established design guidelines:
  - Standardized exit code usage to use constants consistently instead of hard-coded values
  - Improved help text examples for `status` and `version` commands
  - Added common workflow section to main help text for better discoverability
  - Enhanced error messages with more actionable hints and structured error types
  - Cleaned up global flag handling by removing unused code
  - All commands now provide comprehensive examples and usage guidance
  - Fixed function naming consistency: `handleCreateLinks` → `handleCreate`, `handleRemoveLinks` → `handleRemove`, `handlePruneLinks` → `handlePrune`
- **Circular symlink validation** - Fixed incorrect circular reference error when symlinks already exist and point to the correct target. The validation now properly allows existing symlinks that point to their intended source.
- **Test expectations** - Updated validation tests to reflect the new behavior where relinking (creating a symlink that already points to the correct location) is considered valid and not an error.
- **Error message consistency** - Fixed inconsistent error message formats across different commands (e.g., "could not" vs "Failed to")
- **Summary output for orphan command** - Added missing summary output when orphaning directories with multiple files, matching the pattern of other bulk operations

## [0.3.0] - 2025-06-28

### Changed - MAJOR REWRITE

This release represents a major rewrite of lnk's core functionality.

- **BREAKING: File-only linking** - Removed directory linking feature. Cfgman now ONLY creates individual file symlinks, never directory symlinks. This ensures:
  - Consistent behavior across all operations
  - No conflicts between different source mappings
  - Ability to mix files from different sources in the same directory
  - Local-only files can coexist with managed configs
- **Configuration-driven design** - All behavior now controlled by `.lnk.json` with no hardcoded defaults
- **Simplified codebase** - Removed obsolete features and redundant logic:
  - Removed LinkStrategy (file/directory linking modes)
  - Removed hardcoded directory list
  - Removed platform-specific home directory logic
  - Consolidated redundant orphan command logic
  - Simplified git operations to basic rm-with-fallback pattern
- **Improved create-links workflow** - Refactored to use clear three-phase approach:
  1. Discovery phase - collect all files to link
  2. Validation phase - validate all targets before making changes
  3. Execution phase - create all symlinks
- **Better error messages** - More descriptive errors throughout, especially for adopt command when source directory is not in mappings

## [0.2.0] - 2025-06-28

### Changed

- Switched from beta versioning (v1.0.0-beta.x) to standard pre-1.0 versioning (v0.x.x) to better align with semantic versioning practices for pre-release software

## [0.1.1] - 2025-06-27

### Fixed

- **Orphan command message order** - Messages now correctly reflect the actual operation order:
  1. Remove symlink
  2. Copy content back to original location
  3. Remove file from repository
- **Redundant orphan messages** - Fixed duplicate confirmation messages when orphaning directories with multiple symlinks
- **Untracked file removal** - Fixed issue where files not tracked by git were left in the repository after orphaning. The orphan command now properly removes all files from the repository regardless of git tracking status.

## [0.1.0] - 2025-06-24

Initial release of lnk.

- **Directory-based operation** - Works from repository directory (like git, npm, make)
- **Simple configuration format** - Single `.lnk.json` file with link mappings
- **Built-in ignore patterns** - Gitignore-style pattern matching without git dependency
- **Flexible link mappings** - Map any source directory to any target location
- **Smart linking strategies** - Choose between file-level or directory-level linking
- **Safety features**:
  - Dry-run mode for all operations
  - Confirmation prompts for destructive actions
  - Cross-repository symlink protection
- **Core commands**:
  - `status` - Show all managed symlinks with their state
  - `adopt` - Move existing files into repository management
  - `orphan` - Remove files from management (restore to original location)
  - `create-links` - Create symlinks based on configuration
  - `remove-links` - Remove all managed symlinks
  - `prune-links` - Remove broken symlinks
- **Performance** - Concurrent operations for status checking
- **Zero dependencies** - Pure Go implementation using only standard library

[unreleased]: https://github.com/cpplain/lnk/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/cpplain/lnk/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/cpplain/lnk/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/cpplain/lnk/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/cpplain/lnk/releases/tag/v0.1.0
