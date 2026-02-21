# lnk CLI Refactor - Progress Notes

## Session 1: Initialization (2026-02-21)

### Initial Inventory

**Existing Codebase:**
- `cmd/lnk/main.go` - Subcommand-based CLI with manual global flag parsing
  - Commands: status, adopt, orphan, create, remove, prune, version
  - Global flags: --verbose, --quiet, --output, --no-color, --version, --yes, --config, --ignore
  - Uses Levenshtein distance for command suggestions

- `internal/lnk/config.go` - JSON-based configuration system
  - Precedence: --config flag → XDG → ~/.config → ~/.lnk.json → ./.lnk.json → defaults
  - Config struct: LinkMapping{Source, Target} + IgnorePatterns
  - Built-in ignore defaults for .git, .DS_Store, README*, etc.

- `internal/lnk/linker.go` - Core linking operations
  - CreateLinks(config, dryRun) - 3-phase execution: collect, validate, execute
  - RemoveLinks(config, dryRun, force)
  - PruneLinks(config, dryRun, force)
  - collectPlannedLinks() - recursive file traversal with ignore pattern support

- `internal/lnk/adopt.go` - File adoption into repo
  - Adopt(source, config, sourceDir, dryRun)
  - Handles both files and directories
  - Validates source, moves to repo, creates symlink

- `internal/lnk/orphan.go` - Remove files from repo management
  - Orphan(link, config, dryRun, force)
  - Removes symlink, copies file back, removes from repo
  - Supports orphaning entire directories

- `internal/lnk/link_utils.go` - Managed link discovery
  - FindManagedLinks(startPath, config) - walks directory tree
  - checkManagedLink() - validates symlink points to configured source
  - ManagedLink struct: Path, Target, IsBroken, Source

**Testing Infrastructure:**
- Unit tests in `internal/lnk/*_test.go` with testutil_test.go helpers
- E2E tests in `e2e/e2e_test.go` with helpers_test.go
- Makefile targets: test, test-unit, test-e2e, test-coverage

### Refactoring Plan Summary

**Goal:** Transform lnk from config-required, subcommand-based CLI to stow-like convention-based, flag-based CLI.

**Breaking Changes (acceptable for pre-v1.0):**
- Config file format changes from JSON mappings to CLI flag format
- CLI switches from subcommands to action flags
- No migration path needed

**4 Implementation Phases:**

1. **Config File Support** - New .lnkconfig format with CLI flags
   - Discovery order: .lnkconfig in source → XDG → ~/.lnkconfig
   - Parse stow-style flags (--target=~, --ignore=*.swp)
   - Support .lnkignore file with gitignore syntax
   - Merge with CLI flags (CLI takes precedence)

2. **Options-Based API** - Package-centric linking
   - LinkOptions{SourceDir, TargetDir, Packages, IgnorePatterns, DryRun}
   - New functions: CreateLinksWithOptions, RemoveLinksWithOptions, StatusWithOptions, PruneWithOptions
   - Refactor collectPlannedLinks to work with packages instead of config mappings

3. **CLI Rewrite** - Flag-based interface
   - Action flags: -C/--create (default), -R/--remove, -S/--status, -P/--prune, -A/--adopt, -O/--orphan
   - Directory flags: -s/--source (default: .), -t/--target (default: ~)
   - Positional args = packages (at least one required)
   - Remove subcommand routing completely

4. **Update Internal Functions** - Adapt to new interface
   - adopt: work with packages instead of explicit paths
   - orphan: use --orphan PATH flag
   - prune: support optional packages
   - FindManagedLinksForSources: filter by source packages

**Key Differences from Stow:**
- Action flags: -C/--create, -R/--remove vs stow's -S, -D, -R
- Source flag: -s/--source vs stow's -d/--dir
- Ignore syntax: gitignore vs Perl regex
- Ignore file: .lnkignore vs .stow-local-ignore
- No tree folding (always links files individually)
- Added: -S/--status, -P/--prune, -O/--orphan (not in stow)

### Tasks Created

Created `.lorah/tasks.json` with 24 testable requirements:
- 3 tasks for Phase 1 (config file support)
- 7 tasks for Phase 2 (options-based API)
- 6 tasks for Phase 3 (CLI rewrite)
- 3 tasks for Phase 4 (internal function updates)
- 4 tasks for testing
- 1 task for verification

All tasks marked as `"passes": false` initially.

### Technology Stack

- **Language:** Go (stdlib only, no external dependencies)
- **Build:** Makefile (build, test, test-unit, test-e2e, fmt, lint, check)
- **Testing:** Go standard testing + e2e test suite
- **Version:** Injected via ldflags from git tags
- **Git:** Already initialized on branch `config-command-refactor`

### Session Complete

✅ Read and understood spec.md refactoring requirements
✅ Explored existing codebase architecture
✅ Created comprehensive task list (tasks.json)
✅ Documented initial inventory and plan (progress.md)
⏭️ Ready to commit initialization work

**Next Steps:**
1. Commit this initialization work
2. Begin Phase 1: Config file support

---

## Session 2: Phase 1 - Config File Support (2026-02-21)

### Tasks Completed

✅ **Task 1: LoadConfig for .lnkconfig format**
- Added `FlagConfig` struct to hold flag-based config data (Target, IgnorePatterns)
- Implemented `parseFlagConfigFile()` to parse stow-style flag format:
  - Format: `--flag=value` or `--flag value` (one per line)
  - Supports comments (`#`) and blank lines
  - Handles `--target` and `--ignore` flags
  - Ignores unknown flags for forward compatibility
- Implemented `LoadFlagConfig(sourceDir)` with discovery precedence:
  1. `.lnkconfig` in source directory (repo-specific)
  2. `$XDG_CONFIG_HOME/lnk/config`
  3. `~/.config/lnk/config`
  4. `~/.lnkconfig`
- Returns empty config (not error) if no config file found
- Comprehensive unit tests: 6 test cases for parsing, 2 for discovery

✅ **Task 2: Parse .lnkignore file**
- Implemented `parseIgnoreFile()` to parse gitignore-style ignore files:
  - Supports comments and blank lines
  - Handles all gitignore patterns (already supported by patterns.go)
  - Supports negation patterns (`!pattern`)
- Implemented `LoadIgnoreFile(sourceDir)` to load `.lnkignore` from source directory
- Returns empty array (not error) if no ignore file found
- Comprehensive unit tests: 5 test cases for parsing, 2 for loading

### Implementation Details

**Files Modified:**
- `internal/lnk/constants.go`: Added `FlagConfigFileName` and `IgnoreFileName` constants
- `internal/lnk/config.go`: Added new config structures and functions (~150 lines)
- `internal/lnk/config_test.go`: Added comprehensive unit tests (~280 lines)

**Key Design Decisions:**
1. **Graceful degradation**: Functions return empty configs/arrays instead of errors when files don't exist
2. **Forward compatibility**: Unknown flags in config files are logged but not rejected
3. **Reused existing patterns.go**: Leveraged existing gitignore pattern matching for .lnkignore
4. **Verbose logging**: Added PrintVerbose calls for debugging config discovery

### Testing Results

```bash
$ go test ./internal/lnk -run "TestParseFlagConfigFile|TestParseIgnoreFile|TestLoadFlagConfig|TestLoadIgnoreFile"
PASS
ok      github.com/cpplain/lnk/internal/lnk     0.278s

$ go test ./internal/lnk
PASS
ok      github.com/cpplain/lnk/internal/lnk     1.664s
```

All unit tests pass including:
- Existing config tests (JSON format still works)
- New flag config parsing tests
- New ignore file parsing tests
- Config discovery precedence tests

### Build Status

✅ Build succeeds: `make build` completes successfully

### Notes

- Task 3 (Merge config with CLI flags) is the next logical step
- The new config format is separate from the existing JSON config (both supported)
- CLI merging will need to combine FlagConfig + CLI args + built-in defaults

**Next Steps:**
1. Commit these changes
2. Implement Task 3: Merge config with CLI flags

---

## Session 3: Phase 1 - Config Merging (2026-02-21)

### Tasks Completed

✅ **Task 3: Merge config with CLI flags**
- Added `MergedConfig` struct to hold final merged configuration:
  - `SourceDir`: Source directory (from CLI)
  - `TargetDir`: Target directory (CLI > config > default)
  - `IgnorePatterns`: Combined patterns from all sources
- Implemented `MergeFlagConfig(sourceDir, cliTarget, cliIgnorePatterns)`:
  - Loads flag-based config from .lnkconfig (if exists)
  - Loads ignore patterns from .lnkignore (if exists)
  - Merges with CLI flags using correct precedence
  - Returns unified MergedConfig structure
- Extracted `getBuiltInIgnorePatterns()` function for reusability
- Added `.lnkconfig` and `.lnkignore` to built-in ignore patterns
- Comprehensive unit tests: 9 test cases covering all merging scenarios
  - Tests for default behavior (no config files)
  - Tests for config file only
  - Tests for CLI override precedence
  - Tests for .lnkignore patterns
  - Tests for CLI ignore patterns
  - Tests for combined sources
  - Tests for subdirectory configs
  - Tests for precedence verification

### Implementation Details

**Files Modified:**
- `internal/lnk/config.go`:
  - Added `MergedConfig` struct (~5 lines)
  - Added `getBuiltInIgnorePatterns()` function (~15 lines)
  - Modified `getDefaultConfig()` to use `getBuiltInIgnorePatterns()`
  - Added `MergeFlagConfig()` function (~50 lines)
- `internal/lnk/config_test.go`:
  - Added `TestMergeFlagConfig()` (~145 lines)
  - Added `TestMergeFlagConfigPrecedence()` (~60 lines)
  - Updated `TestLoadConfigWithOptions_DefaultConfig` to expect 11 patterns

**Merging Logic:**
- **Target directory precedence**: CLI flag > .lnkconfig > default (~)
- **Ignore patterns**: All sources combined in order:
  1. Built-in defaults (11 patterns)
  2. .lnkconfig patterns
  3. .lnkignore patterns
  4. CLI flag patterns
- This order allows later patterns to override earlier ones using negation (!)

**Key Design Decisions:**
1. **Additive ignore patterns**: Unlike target (which uses precedence), ignore patterns are combined from all sources for maximum flexibility
2. **Order matters**: CLI patterns come last so users can negate built-in or config patterns
3. **Verbose logging**: Added detailed logging at each merge step for debugging
4. **Graceful handling**: Missing config files don't cause errors, just return empty values

### Testing Results

```bash
$ go test ./internal/lnk -run "TestMergeFlagConfig"
PASS
ok      github.com/cpplain/lnk/internal/lnk     0.570s

$ go test ./internal/lnk
ok      github.com/cpplain/lnk/internal/lnk     1.728s
```

All unit tests pass including:
- Existing config tests (JSON format still works)
- New flag config parsing tests
- New ignore file parsing tests
- New config merging tests

### Build Status

✅ Build succeeds: `make build` completes successfully
✅ Binary created: `bin/lnk` (3.6M)

### Phase 1 Status

Phase 1 (Config file support) is now **COMPLETE**:
- ✅ Task 1: LoadConfig for .lnkconfig format
- ✅ Task 2: Parse .lnkignore file
- ✅ Task 3: Merge config with CLI flags

### Notes

- The merging logic is ready for Phase 2 (Options-based API)
- Next task should be Task 4: LinkOptions struct
- The new config system is fully backward compatible with JSON configs
- All built-in ignore patterns now include the new config files

**Next Steps:**
1. Commit these changes
2. Begin Phase 2: Create LinkOptions struct and *WithOptions functions

---

## Session 4: Phase 2 - LinkOptions Struct (2026-02-21)

### Tasks Completed

✅ **Task 4: LinkOptions struct**
- Added `LinkOptions` struct to `internal/lnk/linker.go`
- Struct includes all required fields from spec:
  - `SourceDir string`: base directory for dotfiles (e.g., ~/git/dotfiles)
  - `TargetDir string`: where to create symlinks (default: ~)
  - `Packages []string`: subdirectories to process (e.g., ["home", "private/home"])
  - `IgnorePatterns []string`: combined ignore patterns from all sources
  - `DryRun bool`: preview mode flag
- Placed after `PlannedLink` struct for logical organization
- Added comprehensive documentation comments for each field

### Implementation Details

**Files Modified:**
- `internal/lnk/linker.go`: Added LinkOptions struct definition (~10 lines)

**Design Decisions:**
1. **Field types match spec exactly**: Used []string for Packages and IgnorePatterns to support multiple values
2. **Documentation**: Added clear comments explaining purpose of each field with examples
3. **Location**: Placed struct early in file after PlannedLink for visibility
4. **Naming**: Used LinkOptions (not CreateOptions) to be generic for all operations

### Syntax Verification

```bash
$ gofmt -e /Users/christopherplain/git/lnk/internal/lnk/linker.go
Syntax OK
```

Code parses correctly. Go build cache permission issue prevents compilation, but syntax is valid.

### Notes

- LinkOptions struct is foundation for Phase 2 functions
- Next tasks will implement *WithOptions functions that use this struct
- All field names and types match spec.md requirements
- Ready for CreateLinksWithOptions, RemoveLinksWithOptions, etc.

**Next Steps:**
1. Commit this change
2. Implement Task 5: CreateLinksWithOptions function

---

## Session 5: Phase 2 - CreateLinksWithOptions Function (2026-02-21)

### Tasks Completed

✅ **Task 5: CreateLinksWithOptions function**
- Implemented `collectPlannedLinksWithPatterns()` helper function:
  - Takes ignorePatterns []string instead of *Config
  - Uses MatchesPattern directly for ignore checking
  - Same recursive file traversal as original collectPlannedLinks
  - Returns []PlannedLink with source/target pairs
- Implemented `CreateLinksWithOptions(opts LinkOptions) error`:
  - Validates inputs (packages required, source dir exists)
  - Expands source and target paths (handles ~/)
  - Supports multiple packages in single operation
  - Handles package "." for flat repository structure
  - Handles nested package paths (e.g., "private/home")
  - Skips non-existent packages with PrintSkip (doesn't error)
  - Follows same 3-phase execution as CreateLinks:
    1. Collect planned links from all packages
    2. Validate all targets
    3. Execute or show dry-run
  - Reuses existing executePlannedLinks for actual linking
- Added comprehensive unit tests (9 test cases):
  - Single package linking
  - Multiple packages
  - Package with "." (current directory)
  - Nested package paths (private/home)
  - Ignore patterns
  - Dry-run mode
  - Non-existent package skipped gracefully
  - Error: no packages specified
  - Error: source directory does not exist

### Implementation Details

**Files Modified:**
- `internal/lnk/linker.go`:
  - Added `collectPlannedLinksWithPatterns()` (~40 lines)
  - Added `CreateLinksWithOptions()` (~145 lines)
- `internal/lnk/linker_test.go`:
  - Added `TestCreateLinksWithOptions()` (~155 lines)

**Key Design Decisions:**
1. **Helper function**: Created collectPlannedLinksWithPatterns to avoid coupling to Config
2. **Package "." handling**: Special case to use source directory directly for flat repos
3. **Graceful package skipping**: Non-existent packages print skip message, don't error
4. **Validation order**: Validate source dir exists, then validate each package
5. **Reused executePlannedLinks**: Leveraged existing linking logic for consistency
6. **Verbose logging**: Added logging at each phase for debugging

### Testing Results

```bash
$ GOCACHE=$TMPDIR/go-cache go test ./internal/lnk -run "TestCreateLinksWithOptions"
PASS
ok      github.com/cpplain/lnk/internal/lnk     0.419s

$ GOCACHE=$TMPDIR/go-cache go test ./internal/lnk
PASS
ok      github.com/cpplain/lnk/internal/lnk     1.727s
```

All unit tests pass including:
- Existing CreateLinks tests (old config-based API)
- New CreateLinksWithOptions tests (package-based API)
- All other internal/lnk tests

### Build Status

✅ Syntax valid: `gofmt -e` succeeds
✅ All unit tests pass

### Notes

- collectPlannedLinksWithPatterns is a stepping stone toward Task 9 (refactoring original collectPlannedLinks)
- The old CreateLinks and new CreateLinksWithOptions coexist - both work
- Package-based API is more flexible than config mappings
- Next task should be Task 6: RemoveLinksWithOptions

**Next Steps:**
1. Commit this change
2. Implement Task 6: RemoveLinksWithOptions function
