# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- **Progress indicator consistency** - Fixed progress indicators showing prematurely and leaving terminal artifacts:
  - Progress indicators now only show for operations taking longer than 1 second
  - Removed duplicate progress tracking that caused immediate output
  - Fixed terminal cleanup to prevent `⏎` symbols from appearing
  - Consistent behavior across all commands (create, status, adopt, orphan, remove, prune)

- **Output formatting consistency** - Standardized output across all commands:
  - All commands now display a newline after headers for consistent spacing
  - Empty result messages are consistent (e.g., "No files to link found.")
  - Summary sections have consistent spacing with newlines before them
  - Added helper functions to enforce output patterns
  - Fixed `create` command to show feedback when all symlinks already exist

### Breaking Changes

- **Renamed project from cfgman to lnk** - The project has been renamed to better reflect its focused purpose as a symlink management tool:
  - Binary name: `cfgman` → `lnk`
  - Config file: `.cfgman.json` → `.lnk.json`
  - Environment variables: `CFGMAN_*` → `LNK_*`
  - Go module: `github.com/cpplain/cfgman` → `github.com/cpplain/lnk`

- **Command names reverted to original form**:
  - `link` → `create` (for creating symlinks)
  - `unlink` → `remove` (for removing symlinks)

- **Directory-based architecture with absolute paths** - lnk now uses absolute paths for source directories:
  - Source directories in config must now be absolute paths (e.g., `~/dotfiles/home` instead of `home`)
  - Adopt command requires absolute path: `--source-dir ~/dotfiles/home`
  - Removed `--repo-dir` flag and `LNK_REPO_DIR` environment variable
  - Can run lnk from any directory - no longer requires a "repository directory"

- **Command arguments replaced with explicit flags**:
  - `adopt PATH SOURCE_DIR` → `adopt --path PATH --source-dir SOURCE_DIR`
  - `orphan PATH` → `orphan --path PATH`

- **Flag changes**:
  - Removed `--force` flags from commands in favor of global `--yes` flag
  - Replaced `--json` flag with `--output FORMAT` flag (supports text/json)

- **Removed `init` command** - lnk works with built-in defaults and doesn't require a config file

### Added

- **Works without configuration** - Configuration discovery with flexible precedence:
  - Built-in defaults: `~/dotfiles/home`→`~/`, `~/dotfiles/config`→`~/.config/`
  - XDG Base Directory support: `$XDG_CONFIG_HOME/lnk/config.json`
  - Global flags: `--config`, `--source-dir`, `--target-dir`, `--ignore`
  - Environment variables: `LNK_CONFIG`, `LNK_SOURCE_DIR`, `LNK_TARGET_DIR`, `LNK_IGNORE`

- **Global flags for better control**:
  - `--verbose`/`-v`: Detailed debug output
  - `--quiet`/`-q`: Suppress non-error output
  - `--yes`/`-y`: Skip confirmation prompts
  - `--no-color`: Disable colored output
  - `--output FORMAT`: Choose output format (text/json)

- **Enhanced user experience features**:
  - "Did you mean?" suggestions for mistyped commands
  - Progress indicators for operations taking more than 1 second
  - Next-step suggestions after successful operations
  - Confirmation prompts for destructive operations
  - Error messages with actionable "Try:" suggestions

- **Machine-friendly features**:
  - JSON output support for `status` command
  - Automatic output adaptation for piping (simplified format)
  - Specific exit codes: 0 (success), 1 (runtime errors), 2 (usage errors)

- **CLI design documentation** following [cpplain/cli-design](https://github.com/cpplain/cli-design) principles

### Changed

- **Improved help system**:
  - Standardized formatting across all commands
  - Commands appear before options (standard convention)
  - "See also" sections for related commands
  - Removed `help` command (use `lnk <command> --help`)

- **Better output formatting**:
  - Consistent visual indicators: ✓ (success), ✗ (error), ! (warning)
  - Home directory displayed as `~` consistently
  - Summary statistics for bulk operations
  - Table-aligned status output

### Fixed

- **Circular symlink validation** - Now correctly allows existing symlinks that already point to their intended target
- **Test environment isolation** - Tests no longer load system configuration files

## [0.3.0] - 2025-06-28

### Changed

- **BREAKING: File-only linking** - Removed directory linking feature. cfgman now ONLY creates individual file symlinks, never directory symlinks. This ensures:
  - Consistent behavior across all operations
  - No conflicts between different source mappings
  - Ability to mix files from different sources in the same directory
  - Local-only files can coexist with managed configs
- **Configuration-driven design** - All behavior now controlled by `.cfgman.json` with no hardcoded defaults
- **Simplified codebase** - Removed obsolete features and redundant logic for cleaner, more maintainable code
- **Improved create-links workflow** - Better performance and reliability when creating multiple symlinks
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

Initial release of cfgman.

- **Directory-based operation** - Works from repository directory (like git, npm, make)
- **Simple configuration format** - Single `.cfgman.json` file with link mappings
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
