# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

### Changed

### Deprecated

### Removed

### Fixed

### Security

## [0.4.0] - 2026-02-14

### Added

- Comprehensive end-to-end testing suite
- Zero-configuration defaults with flexible config discovery (`~/.config/lnk/config.json`, `~/.lnk.json`)
- Global flags: `--verbose`, `--quiet`, `--yes`, `--no-color`, `--output`
- Command suggestions for typos, progress indicators, confirmation prompts
- JSON output and specific exit codes for scripting

### Changed

- Improved help system with standardized formatting and "See also" sections
- Better output formatting with consistent indicators (✓, ✗, !) and `~` for home
- **BREAKING: Renamed project from cfgman to lnk** (binary, config, module)
- **BREAKING: Command names** `link`→`create`, `unlink`→`remove`
- **BREAKING: Source directories must be absolute paths**; removed `--repo-dir`
- **BREAKING: Explicit flags required** (`adopt --path PATH --source-dir DIR`)
- **BREAKING: Replaced `--force` with global `--yes`**; `--json` with `--output FORMAT`

### Removed

- **BREAKING: Custom environment variables** (`LNK_CONFIG`, `LNK_IGNORE`, `LNK_DEBUG`)
- **BREAKING: `init` command** (built-in defaults work without config)

### Fixed

- `--yes` flag now correctly skips confirmation in `remove` and `prune`
- Adopt command flag parsing (previously consumed by global flags)
- Config file errors now reported instead of silently falling back to defaults
- Progress indicators only show after 1 second; no terminal artifacts
- Output spacing consistency across all commands
- Circular symlink validation for existing correct symlinks
- Test environment isolation from system config

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

[unreleased]: https://github.com/cpplain/lnk/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/cpplain/lnk/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/cpplain/lnk/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/cpplain/lnk/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/cpplain/lnk/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/cpplain/lnk/releases/tag/v0.1.0
