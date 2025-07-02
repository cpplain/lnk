# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- **Circular symlink validation** - Fixed incorrect circular reference error when symlinks already exist and point to the correct target. The validation now properly allows existing symlinks that point to their intended source.

## [0.3.0] - 2025-06-28

### Changed - MAJOR REWRITE

This release represents a major rewrite of cfgman's core functionality.

- **BREAKING: File-only linking** - Removed directory linking feature. Cfgman now ONLY creates individual file symlinks, never directory symlinks. This ensures:
  - Consistent behavior across all operations
  - No conflicts between different source mappings
  - Ability to mix files from different sources in the same directory
  - Local-only files can coexist with managed configs
- **Configuration-driven design** - All behavior now controlled by `.cfgman.json` with no hardcoded defaults
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
  - `init` - Create minimal configuration template
  - `status` - Show all managed symlinks with their state
  - `adopt` - Move existing files into repository management
  - `orphan` - Remove files from management (restore to original location)
  - `create-links` - Create symlinks based on configuration
  - `remove-links` - Remove all managed symlinks
  - `prune-links` - Remove broken symlinks
- **Performance** - Concurrent operations for status checking
- **Zero dependencies** - Pure Go implementation using only standard library

[0.3.0]: https://github.com/cpplain/cfgman/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/cpplain/cfgman/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/cpplain/cfgman/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/cpplain/cfgman/releases/tag/v0.1.0
