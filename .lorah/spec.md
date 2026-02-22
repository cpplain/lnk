# lnk Legacy Code Cleanup

## Overview

Remove legacy code from the lnk CLI codebase. The CLI was recently refactored from a subcommand-based interface with JSON config to a stow-like flag-based interface with `.lnkconfig` files. However, all legacy code was retained for "backward compatibility" which was explicitly NOT desired.

## Goals

1. **Remove ~900 lines of legacy code** that is no longer used
2. **Simplify naming** - drop `WithOptions` suffixes and `Flag` prefixes
3. **Clean up tests** - remove tests for legacy functions
4. **Update documentation** - reflect new simplified API

## Implementation Phases

### Phase 0: Simplify Naming

After removing legacy code, simplify the remaining API:

**Function Renames:**

- `CreateLinksWithOptions` → `CreateLinks`
- `RemoveLinksWithOptions` → `RemoveLinks`
- `StatusWithOptions` → `Status`
- `PruneWithOptions` → `Prune`
- `AdoptWithOptions` → `Adopt`
- `OrphanWithOptions` → `Orphan`
- `FindManagedLinksForSources` → `FindManagedLinks`
- `MergeFlagConfig` → `LoadConfig`
- `LoadFlagConfig` → `loadConfigFile` (unexported)
- `parseFlagConfigFile` → `parseConfigFile` (unexported)

**Type Renames:**

- `FlagConfig` → `FileConfig` (config from .lnkconfig file)
- `MergedConfig` → `Config` (final resolved config)
- `LinkOptions`, `AdoptOptions`, `OrphanOptions` - keep as-is

**Constant Renames:**

- `FlagConfigFileName` → `ConfigFileName`

### Phase 1: Remove Legacy Types and Constants

**File: `internal/lnk/constants.go`**

- Remove old `ConfigFileName = ".lnk.json"` (line 18)

**File: `internal/lnk/errors.go`**

- Remove `ErrNoLinkMappings` error (lines 16-17)

**File: `internal/lnk/config.go`**
Remove these types:

- `LinkMapping` struct (lines 15-19)
- Old `Config` struct with `LinkMappings` (lines 21-25)
- `ConfigOptions` struct (lines 27-31)

### Phase 2: Remove Legacy Functions

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

### Phase 3: Clean Up Status Command

**Decision:** Keep status command - provides quick view of current links.

**File: `internal/lnk/status.go`**

- Remove only the legacy `Status(config *Config)` function
- Keep `StatusWithOptions()` (will be renamed to `Status` in Phase 0)

### Phase 4: Update Documentation

**File: `README.md`**
Complete rewrite to reflect new CLI:

- Installation section
- Quick start examples (`lnk .`, `lnk home`, `lnk home private/home`)
- Usage section with flag descriptions
- Examples for all operations
- Config file documentation (.lnkconfig, .lnkignore)
- How it works section

See `mutable-gliding-flame.md` lines 100-212 for detailed README content.

### Phase 5: Clean Up Tests

**Files to modify:**

- `internal/lnk/config_test.go` - Remove tests for legacy JSON config
- `internal/lnk/linker_test.go` - Remove tests using old `*Config`
- `internal/lnk/status_test.go` - Remove tests using old `*Config`
- `internal/lnk/adopt_test.go` - Remove tests for legacy `Adopt()`
- `internal/lnk/orphan_test.go` - Remove tests for legacy `Orphan()`
- `internal/lnk/link_utils_test.go` - Remove tests for legacy `FindManagedLinks()`
- `internal/lnk/errors_test.go` - Remove `ErrNoLinkMappings` tests

### Phase 6: Update CLAUDE.md

**File: `CLAUDE.md`**
Update the "Configuration Structure" section to reflect new types:

```go
// Config loaded from .lnkconfig file
type FileConfig struct {
    Target         string
    IgnorePatterns []string
}

// Final resolved configuration
type Config struct {
    SourceDir      string
    TargetDir      string
    IgnorePatterns []string
}

// Options for operations
type LinkOptions struct {
    SourceDir      string
    TargetDir      string
    Packages       []string
    IgnorePatterns []string
    DryRun         bool
}
```

Remove references to:

- Subcommand routing
- `handleXxx(args, globalOptions)` pattern
- JSON config file

## Summary of Changes

### Deletions

| File          | Lines to Remove (approx) |
| ------------- | ------------------------ |
| config.go     | ~250 lines               |
| linker.go     | ~200 lines               |
| status.go     | ~90 lines                |
| adopt.go      | ~120 lines               |
| orphan.go     | ~120 lines               |
| link_utils.go | ~90 lines                |
| constants.go  | 1 line                   |
| errors.go     | 2 lines                  |
| Test files    | Significant cleanup      |

**Total: ~900+ lines of legacy code to remove**

### Renames

| Count | Type             |
| ----- | ---------------- |
| 10    | Function renames |
| 2     | Type renames     |
| 1     | Constant rename  |

## Technology Stack

- **Language**: Go (stdlib only, no external dependencies)
- **Build**: Makefile with targets (build, test, test-unit, test-e2e, fmt, lint)
- **Testing**: Go standard testing + e2e test suite
- **Version**: Injected via ldflags

## Success Criteria

- All unit tests pass (`make test-unit`)
- All e2e tests pass (`make test-e2e`)
- `make build` succeeds
- Binary size reduced (fewer lines of code)
- No references to legacy code remain:
  - No `LinkMapping` in codebase
  - No `WithOptions` suffixes in function names
  - No `FlagConfig` type name
  - No `.lnk.json` references

## Verification Commands

After cleanup:

```bash
# Build
make build

# Run tests
make test

# Verify no legacy references
grep -r "LinkMapping" internal/
grep -r "WithOptions" internal/
grep -r "FlagConfig" internal/
grep -r "MergedConfig" internal/
grep -r "\\.lnk\\.json" .
```

All greps should return no results (or only in test files that will be cleaned up).
