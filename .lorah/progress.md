# lnk Legacy Code Cleanup - Progress Notes

## Session 1: Initialization (Complete)

### Context

The lnk CLI was recently refactored from a subcommand-based interface with JSON config to a stow-like flag-based interface with `.lnkconfig` files. All 24 refactoring tasks passed. However, legacy code was retained for "backward compatibility" which was explicitly NOT desired. This cleanup project will remove ~900 lines of legacy code.

### Initial Inventory

**Legacy Code Identified:**

1. **constants.go (1 line)**
   - `ConfigFileName = ".lnk.json"` (line 18)

2. **errors.go (2 lines)**
   - `ErrNoLinkMappings` error constant (lines 16-17)

3. **config.go (~250 lines)**
   - Types: `LinkMapping`, old `Config`, `ConfigOptions` (lines 15-31)
   - Functions: 9 legacy functions including `LoadConfig()`, `Config.Validate()`, etc.

4. **linker.go (~200 lines)**
   - `CreateLinks(config *Config)` (lines 26-102)
   - `RemoveLinks(config *Config)` and `removeLinks()` (lines 243-322)
   - `PruneLinks(config *Config)` (lines 584-668)

5. **status.go (~90 lines)**
   - `Status(config *Config)` (lines 29-120)

6. **adopt.go (~120 lines)**
   - `ensureSourceDirExists()` (lines 86-105)
   - `Adopt(source string, config *Config)` (lines 456-552)

7. **orphan.go (~120 lines)**
   - `Orphan(link string, config *Config)` (lines 202-322)

8. **link_utils.go (~90 lines)**
   - `FindManagedLinks(startPath string, config *Config)` (lines 18-54)
   - `checkManagedLink(linkPath string, config *Config)` (lines 57-108)

9. **Test files**
   - Significant cleanup needed across all test files for legacy functions

**Current State:**

- New API exists with `WithOptions` suffixes: ✅
- Legacy API still present: ❌ (to be removed)
- Tests cover both old and new APIs: ❌ (to be cleaned up)
- Documentation references old patterns: ❌ (to be updated)

### Refactoring Plan

The cleanup follows 6 phases in dependency order:

**Phase 0: Simplify Naming (13 tasks)**
- Rename 10 functions (drop `WithOptions`, `Flag` prefixes)
- Rename 2 types (`FlagConfig` → `FileConfig`, `MergedConfig` → `Config`)
- Rename 1 constant (`FlagConfigFileName` → `ConfigFileName`)

**Phase 1: Remove Legacy Types (3 tasks)**
- Remove old types from config.go
- Remove legacy error from errors.go
- Remove legacy constant from constants.go

**Phase 2: Remove Legacy Functions (6 tasks)**
- Clean up config.go (9 functions)
- Clean up linker.go (3 functions)
- Clean up status.go (1 function)
- Clean up adopt.go (2 functions)
- Clean up orphan.go (1 function)
- Clean up link_utils.go (2 functions)

**Phase 3: Status Command**
- Keep `StatusWithOptions` (will be renamed to `Status` in Phase 0)

**Phase 4: Update Documentation (1 task)**
- Complete README.md rewrite

**Phase 5: Clean Up Tests (7 tasks)**
- Remove legacy tests from all test files

**Phase 6: Update CLAUDE.md (1 task)**
- Update configuration structure documentation

**Total: 34 tasks**

### Task Organization

Tasks are ordered to respect dependencies:
1. Remove legacy functions first (Phase 2)
2. Remove legacy types second (Phase 1)
3. Rename new code to simplified names (Phase 0)
4. Update tests (Phase 5)
5. Verify build and tests pass
6. Update documentation (Phases 4 and 6)

### Files Created

- ✅ `.lorah/tasks.json` - 34 testable tasks
- ✅ `.lorah/progress.md` - This file

### Success Criteria

- All unit tests pass (`make test-unit`)
- All e2e tests pass (`make test-e2e`)
- Binary builds successfully (`make build`)
- No legacy references remain:
  - No `LinkMapping` in codebase
  - No `WithOptions` suffixes
  - No `FlagConfig` type name
  - No `.lnk.json` references

### Verification Commands

```bash
make build
make test
grep -r "LinkMapping" internal/
grep -r "WithOptions" internal/
grep -r "FlagConfig" internal/
grep -r "MergedConfig" internal/
grep -r "\.lnk\.json" .
```

All greps should return no results after cleanup.

### Session Complete

Initialization complete. Ready for cleanup execution.

**Next session:** Begin Phase 2 (Remove Legacy Functions)

## Session 2: Remove Legacy Functions from config.go (Complete)

### Task: Remove legacy functions from config.go

**Removed 9 legacy functions:**
1. `getDefaultConfig()` - lines 284-292
2. `LoadConfig()` - lines 296-325 (deprecated)
3. `loadConfigFromFile()` - lines 328-358
4. `LoadConfigWithOptions()` - lines 361-414
5. `Config.Save()` - lines 417-429
6. `Config.GetMapping()` - lines 432-439
7. `Config.ShouldIgnore()` - lines 442-444
8. `Config.Validate()` - lines 460-503
9. `DetermineSourceMapping()` - lines 506-522

**Additional cleanup:**
- Removed unused `encoding/json` import from config.go
- Removed `ensureSourceDirExists()` from adopt.go (dead code that depended on removed `GetMapping()`)
  - This function was scheduled for removal in Task 4 but was never called anywhere
  - Removing it now prevented build breakage

**Status:**
- ✅ Binary builds successfully
- ⚠️ Tests fail as expected (using removed functions - will be fixed in Task 23)
- Test failures are all in config_test.go and status_test.go, as expected

**Next task:** Task 2 - Remove legacy functions from linker.go

## Session 3: Remove Legacy Functions from linker.go (Complete)

### Task: Remove legacy functions from linker.go

**Removed 4 legacy functions:**
1. `CreateLinks(config *Config, dryRun bool)` - lines 25-102
2. `RemoveLinks(config *Config, dryRun bool, force bool)` - lines 243-245
3. `removeLinks(config *Config, dryRun bool, skipConfirm bool)` - lines 248-322
4. `PruneLinks(config *Config, dryRun bool, force bool)` - lines 584-668

**Verification:**
- ✅ No references to removed functions in production code (cmd/ or internal/lnk/*.go)
- ✅ Only test files reference these functions (will be cleaned up in Task 24)
- ✅ LSP diagnostics show no errors in production code
- ⚠️ Build verification blocked by sandbox restrictions on go build cache
  - Cannot run `make build` due to permission issues with cache directories
  - However, verified via grep and LSP that no production code uses removed functions
  - All compilation errors are in test files only (expected)

**Legacy code removed so far:**
- Session 2: ~250 lines from config.go
- Session 3: ~200 lines from linker.go
- **Total: ~450 lines removed (50% of goal)**

**Next task:** Task 3 - Remove legacy Status function from status.go

## Session 4: Remove Legacy Status Function from status.go (Complete)

### Task: Remove legacy Status function from status.go

**Removed 1 legacy function:**
1. `Status(config *Config)` - lines 28-120 (93 lines)

**Verification:**
- ✅ No references to removed function in production code (cmd/ or internal/lnk/*.go)
- ✅ Only test files reference this function (will be cleaned up in Task 25)
- ✅ LSP diagnostics show errors only in test files:
  - status_test.go:52 - undefined: Status
  - status_json_test.go:63 - undefined: Status
  - status_json_test.go:135 - undefined: Status
- ⚠️ Build verification blocked by sandbox restrictions on go build cache
  - However, verified via grep and LSP that no production code uses removed function

**Legacy code removed so far:**
- Session 2: ~250 lines from config.go
- Session 3: ~200 lines from linker.go
- Session 4: ~93 lines from status.go
- **Total: ~543 lines removed (60% of goal)**

**Next task:** Task 4 - Remove legacy functions from adopt.go

## Session 5: Remove Legacy Functions from adopt.go (Complete)

### Task: Remove legacy functions from adopt.go

**Removed 1 legacy function:**
1. `Adopt(source string, config *Config, sourceDir string, dryRun bool)` - lines 433-530 (98 lines)

**Notes:**
- `ensureSourceDirExists()` was already removed in Session 2 (it was dead code that depended on removed `GetMapping()`)
- The legacy `Adopt` function used the old `*Config` type with `LinkMappings`
- This function is still referenced in adopt_test.go but will be cleaned up in Task 26

**Verification:**
- ✅ No references to legacy `Adopt` in production code (cmd/ or internal/lnk/*.go)
- ✅ Only test file references found (adopt_test.go has 3 calls - expected)
- ✅ LSP diagnostics show errors only in adopt_test.go:
  - Line 109: undefined: Adopt
  - Line 221: undefined: Adopt
  - Line 501: undefined: Adopt
- ⚠️ Build verification blocked by sandbox restrictions on go build cache
  - However, verified via grep and LSP that no production code uses removed function
  - All compilation errors are in test files only (expected)

**Legacy code removed so far:**
- Session 2: ~250 lines from config.go
- Session 3: ~200 lines from linker.go
- Session 4: ~93 lines from status.go
- Session 5: ~98 lines from adopt.go
- **Total: ~641 lines removed (71% of goal)**

**Next task:** Task 5 - Remove legacy Orphan function from orphan.go

## Session 6: Remove Legacy Orphan Function from orphan.go (Complete)

### Task: Remove legacy Orphan function from orphan.go

**Removed 1 legacy function:**
1. `Orphan(link string, config *Config, dryRun bool, force bool)` - lines 202-322 (121 lines)

**Notes:**
- The legacy `Orphan` function used the old `*Config` type with `LinkMappings`
- This function called legacy functions `FindManagedLinks()` and `checkManagedLink()` from link_utils.go (which will be removed in Task 6)
- The new `OrphanWithOptions` function uses `FindManagedLinksForSources` instead
- Production code (cmd/lnk/main.go) uses `OrphanWithOptions`, not the legacy function

**Verification:**
- ✅ No references to legacy `Orphan` in production code (cmd/ or internal/lnk/*.go)
- ✅ Only test file references found (orphan_test.go has 6 calls - expected)
- ✅ LSP diagnostics show errors only in orphan_test.go:
  - Line 159: undefined: Orphan
  - Line 234: undefined: Orphan
  - Line 301: undefined: Orphan
  - Line 350: undefined: Orphan
  - Line 381: undefined: Orphan
  - Line 427: undefined: Orphan
- All compilation errors are in test files only (expected)

**Legacy code removed so far:**
- Session 2: ~250 lines from config.go
- Session 3: ~200 lines from linker.go
- Session 4: ~93 lines from status.go
- Session 5: ~98 lines from adopt.go
- Session 6: ~121 lines from orphan.go
- **Total: ~762 lines removed (85% of goal)**

**Next task:** Task 6 - Remove legacy functions from link_utils.go

## Session 7: Remove Legacy Functions from link_utils.go (Complete)

### Task: Remove legacy functions from link_utils.go

**Removed 2 legacy functions:**
1. `FindManagedLinks(startPath string, config *Config)` - lines 18-54
2. `checkManagedLink(linkPath string, config *Config)` - lines 57-108

**Notes:**
- Both functions used the old `*Config` type with `LinkMappings`
- `checkManagedLink` was only called by `FindManagedLinks`
- The new `FindManagedLinksForSources` function is used by production code instead
- Production code (orphan.go) uses `FindManagedLinksForSources`, not the legacy functions

**Verification:**
- ✅ No references to removed functions in production code (cmd/ or internal/lnk/*.go)
- ✅ Only test file references found (link_utils_test.go - expected)
- ✅ LSP diagnostics show errors only in link_utils_test.go:
  - Line 206: undefined: FindManagedLinks
  - Line 292: undefined: checkManagedLink
- All compilation errors are in test files only (expected)

**Legacy code removed so far:**
- Session 2: ~250 lines from config.go
- Session 3: ~200 lines from linker.go
- Session 4: ~93 lines from status.go
- Session 5: ~98 lines from adopt.go
- Session 6: ~121 lines from orphan.go
- Session 7: ~92 lines from link_utils.go
- **Total: ~854 lines removed (95% of goal)**

**Next task:** Task 7 - Remove legacy types from config.go
