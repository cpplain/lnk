# Plan: Complete Cleanup of Legacy Code

## Context

The lnk CLI was refactored from a subcommand-based interface with JSON config (`.lnk.json`) to a stow-like flag-based interface with `.lnkconfig` files. However, all legacy code was retained for "backward compatibility" which was explicitly NOT desired. This plan removes all legacy code, simplifies naming, and updates documentation.

## Phase 0: Simplify Naming

After removing legacy code, the `*WithOptions` and `FlagConfig` naming becomes unnecessary. Rename for clarity:

### Function Renames

| Current Name                 | New Name                       |
| ---------------------------- | ------------------------------ |
| `CreateLinksWithOptions`     | `CreateLinks`                  |
| `RemoveLinksWithOptions`     | `RemoveLinks`                  |
| `StatusWithOptions`          | `Status`                       |
| `PruneWithOptions`           | `Prune`                        |
| `AdoptWithOptions`           | `Adopt`                        |
| `OrphanWithOptions`          | `Orphan`                       |
| `FindManagedLinksForSources` | `FindManagedLinks`             |
| `MergeFlagConfig`            | `LoadConfig`                   |
| `LoadFlagConfig`             | `loadConfigFile` (unexported)  |
| `parseFlagConfigFile`        | `parseConfigFile` (unexported) |

### Type Renames

| Current Name    | New Name                                   |
| --------------- | ------------------------------------------ |
| `FlagConfig`    | `FileConfig` (config from .lnkconfig file) |
| `MergedConfig`  | `Config` (the final resolved config)       |
| `LinkOptions`   | Keep as-is (clear purpose)                 |
| `AdoptOptions`  | Keep as-is                                 |
| `OrphanOptions` | Keep as-is                                 |

### Constant Renames

| Current Name         | New Name                          |
| -------------------- | --------------------------------- |
| `FlagConfigFileName` | `ConfigFileName` (reuse the name) |

## Phase 1: Remove Legacy Types and Constants

**File: `internal/lnk/constants.go`**

- Remove `ConfigFileName = ".lnk.json"` (line 18)

**File: `internal/lnk/errors.go`**

- Remove `ErrNoLinkMappings` error (lines 16-17)

**File: `internal/lnk/config.go`**
Remove these types and keep only the new ones:

| Remove                                            | Keep                  |
| ------------------------------------------------- | --------------------- |
| `LinkMapping` struct (lines 15-19)                | `FlagConfig` struct   |
| `Config` struct with `LinkMappings` (lines 21-25) | `MergedConfig` struct |
| `ConfigOptions` struct (lines 27-31)              |                       |

## Phase 2: Remove Legacy Functions

**File: `internal/lnk/config.go`**

- Remove `getDefaultConfig()` (lines 284-292)
- Remove `LoadConfig()` (lines 296-325) - marked deprecated
- Remove `loadConfigFromFile()` (lines 328-358)
- Remove `LoadConfigWithOptions()` (lines 361-414)
- Remove `Config.Save()` (lines 417-429)
- Remove `Config.GetMapping()` (lines 432-439)
- Remove `Config.ShouldIgnore()` (lines 442-444)
- Remove `Config.Validate()` (lines 460-503)
- Remove `DetermineSourceMapping()` (lines 506-522)

**File: `internal/lnk/linker.go`**

- Remove `CreateLinks(config *Config, ...)` (lines 26-102)
- Remove `RemoveLinks(config *Config, ...)` and `removeLinks()` (lines 243-322)
- Remove `PruneLinks(config *Config, ...)` (lines 584-668)

**File: `internal/lnk/status.go`**

- Remove `Status(config *Config)` (lines 29-120)

**File: `internal/lnk/adopt.go`**

- Remove `ensureSourceDirExists()` (lines 86-105)
- Remove `Adopt(source string, config *Config, ...)` (lines 456-552)

**File: `internal/lnk/orphan.go`**

- Remove `Orphan(link string, config *Config, ...)` (lines 202-322)

**File: `internal/lnk/link_utils.go`**

- Remove `FindManagedLinks(startPath string, config *Config)` (lines 18-54)
- Remove `checkManagedLink(linkPath string, config *Config)` (lines 57-108)

## Phase 3: Clean Up Status Command

**Decision: Keep status command** - provides quick view of current links.

**File: `internal/lnk/status.go`**

- Remove only the legacy `Status(config *Config)` function (lines 29-120)
- Keep `StatusWithOptions()` and supporting types (`LinkInfo`, `StatusOutput`)

## Phase 4: Update Documentation

**File: `README.md`**
Complete rewrite to reflect new CLI:

````markdown
# lnk

An opinionated symlink manager for dotfiles.

## Installation

```bash
brew install cpplain/tap/lnk
```
````

## Quick Start

```bash
# From your dotfiles directory
cd ~/git/dotfiles
lnk .                    # Flat repo: link everything
lnk home                 # Nested repo: link home/ package
lnk home private/home    # Multiple packages
```

## Usage

```bash
lnk [flags] <packages...>
```

### Action Flags (mutually exclusive)

| Flag                | Description                 |
| ------------------- | --------------------------- |
| `-C, --create`      | Create symlinks (default)   |
| `-R, --remove`      | Remove symlinks             |
| `-S, --status`      | Show status of symlinks     |
| `-P, --prune`       | Remove broken symlinks      |
| `-A, --adopt`       | Adopt files into package    |
| `-O, --orphan PATH` | Remove file from management |

### Directory Flags

| Flag               | Description                     |
| ------------------ | ------------------------------- |
| `-s, --source DIR` | Source directory (default: `.`) |
| `-t, --target DIR` | Target directory (default: `~`) |

### Other Flags

| Flag               | Description                            |
| ------------------ | -------------------------------------- |
| `--ignore PATTERN` | Additional ignore pattern (repeatable) |
| `-n, --dry-run`    | Preview changes                        |
| `-v, --verbose`    | Verbose output                         |
| `-q, --quiet`      | Quiet mode                             |
| `--no-color`       | Disable colors                         |
| `-V, --version`    | Show version                           |
| `-h, --help`       | Show help                              |

## Examples

```bash
# Create links
lnk .                           # Flat repo
lnk home                        # Nested repo
lnk -s ~/dotfiles home          # Specify source

# Remove links
lnk -R home

# Prune broken symlinks
lnk -P

# Adopt existing files
lnk -A home ~/.bashrc ~/.vimrc

# Remove file from management
lnk -O ~/.bashrc
```

## Config Files

### .lnkconfig (optional)

Place in source directory. Format: CLI flags, one per line.

```
--target=~
--ignore=local/
--ignore=*.private
```

### .lnkignore (optional)

Place in source directory. Gitignore syntax.

```
.git
*.swp
README.md
scripts/
```

## How It Works

lnk creates symlinks for individual files (not directories). This allows:

- Multiple packages to map to the same target
- Local files to coexist with managed configs
- Parent directories are created as regular directories

````

## Phase 5: Clean Up Tests

**Files to modify:**
- `internal/lnk/config_test.go` - Remove tests for legacy JSON config
- `internal/lnk/linker_test.go` - Remove tests using `*Config`
- `internal/lnk/status_test.go` - Remove tests using `*Config`
- `internal/lnk/adopt_test.go` - Remove tests for legacy `Adopt()`
- `internal/lnk/orphan_test.go` - Remove tests for legacy `Orphan()`
- `internal/lnk/link_utils_test.go` - Remove tests for legacy `FindManagedLinks()`
- `internal/lnk/errors_test.go` - Remove `ErrNoLinkMappings` tests

## Phase 6: Update CLAUDE.md

**File: `CLAUDE.md`**
Update the "Configuration Structure" section to reflect new types:

```go
// Config loaded from .lnkconfig file
type FileConfig struct {
    Target         string   // Target directory
    IgnorePatterns []string // Ignore patterns
}

// Final resolved configuration (CLI > file > defaults)
type Config struct {
    SourceDir      string
    TargetDir      string
    IgnorePatterns []string
}

// Options for link operations
type LinkOptions struct {
    SourceDir      string
    TargetDir      string
    Packages       []string
    IgnorePatterns []string
    DryRun         bool
}
````

Update "Adding a New Command" section for flag-based approach.

Remove references to:

- Subcommand routing
- `handleXxx(args, globalOptions)` pattern
- JSON config file

## Verification

After cleanup:

```bash
# Build
make build

# Run tests
make test

# Test CLI
cd ~/git/dotfiles  # or any test dotfiles repo
lnk . -n
lnk home -n
lnk -R home -n
lnk -P -n
lnk -A home ~/.testfile -n
lnk -O ~/.testfile -n

# Verify no references to legacy code
grep -r "LinkMapping" internal/
grep -r "WithOptions" internal/
grep -r "FlagConfig" internal/
grep -r "MergedConfig" internal/
grep -r "\.lnk\.json" .
```

## Summary of Changes

### Deletions

| File          | Lines to Remove (approx)         |
| ------------- | -------------------------------- |
| config.go     | ~250 lines                       |
| linker.go     | ~200 lines                       |
| status.go     | ~90 lines (legacy function only) |
| adopt.go      | ~120 lines                       |
| orphan.go     | ~120 lines                       |
| link_utils.go | ~90 lines                        |
| constants.go  | 1 line                           |
| errors.go     | 2 lines                          |
| Test files    | Significant cleanup              |

**Total: ~900+ lines of legacy code to remove**

### Renames

| Count | Type                                                                  |
| ----- | --------------------------------------------------------------------- |
| 10    | Function renames (drop `WithOptions`, `Flag` prefixes)                |
| 2     | Type renames (`FlagConfig` → `FileConfig`, `MergedConfig` → `Config`) |
| 1     | Constant rename (`FlagConfigFileName` → `ConfigFileName`)             |

### Files Modified

All test files will need updates to use new function/type names.
