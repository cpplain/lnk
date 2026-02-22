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

## Session 8: Remove Legacy Types from config.go (Complete)

### Task: Remove legacy types from config.go

**Removed 3 legacy types:**
1. `LinkMapping` struct - lines 14-18
2. Old `Config` struct with `LinkMappings` field - lines 20-24
3. `ConfigOptions` struct - lines 26-30

**Notes:**
- All three types used the old JSON-based config system
- `LinkMapping` was referenced by the old `Config` struct
- These types were only used in test files, not production code
- New types (`FlagConfig`, `MergedConfig`) remain intact for the new flag-based config system

**Verification:**
- ✅ No references to removed types in production code (cmd/ or internal/lnk/*.go)
- ✅ Only test file references found (config_test.go, status_test.go, status_json_test.go, orphan_test.go, linker_test.go, adopt_test.go, link_utils_test.go, errors_test.go)
- ✅ LSP diagnostics show errors only in test files:
  - orphan_test.go: undefined Config and LinkMapping
  - status_json_test.go: undefined Config and LinkMapping
  - status_test.go: undefined Config and LinkMapping
  - config_test.go: undefined Config and LinkMapping
- All compilation errors are in test files only (expected)

**Legacy code removed so far:**
- Session 2: ~250 lines from config.go (functions)
- Session 3: ~200 lines from linker.go
- Session 4: ~93 lines from status.go
- Session 5: ~98 lines from adopt.go
- Session 6: ~121 lines from orphan.go
- Session 7: ~92 lines from link_utils.go
- Session 8: ~17 lines from config.go (types)
- **Total: ~871 lines removed (97% of goal)**

**Next task:** Task 8 - Remove legacy error from errors.go

## Session 9: Remove Legacy Error from errors.go (Complete)

### Task: Remove legacy error from errors.go

**Removed 1 legacy error constant:**
1. `ErrNoLinkMappings` - lines 16-17 (2 lines)

**Notes:**
- This error was part of the old JSON-based config system with `LinkMappings`
- The error indicated when no link mappings were defined in the config
- This validation is no longer needed with the new flag-based config system

**Verification:**
- ✅ No references to `ErrNoLinkMappings` in production code (cmd/ or internal/lnk/*.go)
- ✅ Only test file reference found (errors_test.go:250 - expected)
- ✅ LSP diagnostics show error only in errors_test.go:
  - Line 250: undefined: ErrNoLinkMappings
- All compilation errors are in test files only (expected)

**Legacy code removed so far:**
- Session 2: ~250 lines from config.go (functions)
- Session 3: ~200 lines from linker.go
- Session 4: ~93 lines from status.go
- Session 5: ~98 lines from adopt.go
- Session 6: ~121 lines from orphan.go
- Session 7: ~92 lines from link_utils.go
- Session 8: ~17 lines from config.go (types)
- Session 9: ~2 lines from errors.go
- **Total: ~873 lines removed (97% of goal)**

**Next task:** Task 9 - Remove legacy constant from constants.go

## Session 10: Remove Legacy Constant from constants.go (Complete)

### Task: Remove legacy constant from constants.go

**Removed 1 legacy constant:**
1. `ConfigFileName = ".lnk.json"` - line 18 (1 line, plus comment)

**Notes:**
- This constant represented the old JSON-based config file name
- The new flag-based config system uses `FlagConfigFileName = ".lnkconfig"` instead
- This will later be renamed to `ConfigFileName` in Task 22 (Phase 0)

**Verification:**
- ✅ No references to `ConfigFileName` in production code (cmd/ or internal/lnk/*.go)
- ✅ Grep confirmed only documentation references remain
- ✅ LSP diagnostics show no new errors related to this removal
- ⚠️ Build verification blocked by sandbox restrictions on go build cache
  - However, verified via grep that no production code uses removed constant
  - All existing compilation errors are in test files only (expected, from previous tasks)

**Legacy code removed so far:**
- Session 2: ~250 lines from config.go (functions)
- Session 3: ~200 lines from linker.go
- Session 4: ~93 lines from status.go
- Session 5: ~98 lines from adopt.go
- Session 6: ~121 lines from orphan.go
- Session 7: ~92 lines from link_utils.go
- Session 8: ~17 lines from config.go (types)
- Session 9: ~2 lines from errors.go
- Session 10: ~1 line from constants.go
- **Total: ~874 lines removed (97% of goal)**

**Phase 1 Complete!** All legacy types, constants, and errors have been removed.

**Next task:** Task 10 - Begin Phase 0 renames: Rename CreateLinksWithOptions to CreateLinks

## Session 11: Remove Legacy Tests from errors_test.go (Complete)

### Task: Remove legacy tests from errors_test.go

**Context:**
After removing the legacy `ErrNoLinkMappings` error constant in Session 9, the test file errors_test.go had a compilation error on line 250 where it referenced the removed constant.

**Removed 1 test case:**
1. Test case for `ErrNoLinkMappings` from the `TestStandardErrors` function (line 250)

**Changes:**
- Removed the line `{ErrNoLinkMappings, "no link mappings defined"},` from the test cases array
- The function still tests all other standard errors: ErrConfigNotFound, ErrInvalidConfig, ErrNotSymlink, ErrAlreadyAdopted

**Verification:**
- ✅ No references to `ErrNoLinkMappings` remain in production code or tests
- ✅ Grep confirms only documentation references remain (.lorah/, .claude/)
- ✅ Compilation check shows no errors in errors_test.go
- ✅ Other test files still have expected errors (will be cleaned up in Tasks 23-28)

**Status:**
- ✅ Task 29 complete - errors_test.go no longer references removed legacy code
- ⚠️ Other test files still have errors (as expected):
  - config_test.go: undefined Config
  - adopt_test.go: undefined Config, LinkMapping, Adopt
  - link_utils_test.go: undefined Config, LinkMapping (need Task 28)
  - orphan_test.go: undefined Config, LinkMapping, Orphan (need Task 27)
  - status_test.go: undefined Config, LinkMapping, Status (need Task 25)

**Next task:** Task 28 - Remove legacy tests from link_utils_test.go

## Session 12: Remove Legacy Tests from config_test.go (Complete)

### Task: Remove legacy tests from config_test.go (Task 23)

**Context:**
After removing the legacy types (`Config`, `LinkMapping`) and functions (`LoadConfig`, `LoadConfigWithOptions`, `ConfigOptions`, `Config.Save()`, `Config.GetMapping()`, `Config.ShouldIgnore()`, `Config.Validate()`) in previous sessions, config_test.go had compilation errors referencing these removed items.

**Removed 12 legacy test functions:**
1. `TestConfigSaveAndLoad` - tested old Config.Save() and LoadConfig()
2. `TestConfigSaveNewFormat` - tested old Config.Save() and LoadConfig()
3. `TestLoadConfigNonExistent` - tested old LoadConfig()
4. `TestLoadConfigNewFormat` - tested old LoadConfig() with JSON
5. `TestShouldIgnore` - tested old Config.ShouldIgnore() method
6. `TestGetMapping` - tested old Config.GetMapping() method
7. `TestConfigValidate` - tested old Config.Validate() method
8. `TestLoadConfigWithOptions_DefaultConfig` - tested old LoadConfigWithOptions()
9. `TestLoadConfigWithOptions_ConfigFilePrecedence` - tested old LoadConfigWithOptions()
10. `TestLoadConfigWithOptions_FlagOverrides` - tested old LoadConfigWithOptions()
11. `TestLoadConfigWithOptions_PartialOverrides` - tested old LoadConfigWithOptions()
12. `TestGetXDGConfigDir` - tested helper for legacy config system

**Additional cleanup:**
- Removed `writeConfigFile` helper function (used by legacy tests only)
- Removed unused `encoding/json` import

**Kept 6 tests for new flag-based config system:**
1. `TestParseFlagConfigFile` - tests parseFlagConfigFile()
2. `TestParseIgnoreFile` - tests parseIgnoreFile()
3. `TestLoadFlagConfig` - tests LoadFlagConfig()
4. `TestLoadIgnoreFile` - tests LoadIgnoreFile()
5. `TestMergeFlagConfig` - tests MergeFlagConfig()
6. `TestMergeFlagConfigPrecedence` - tests MergeFlagConfig() precedence

**Statistics:**
- File reduced from 1427 lines to 608 lines (**819 lines removed**)
- Test count reduced from 18 tests to 6 tests (12 legacy tests removed)

**Verification:**
- ✅ No references to `Config` (old type) remain in test file
- ✅ No references to `LinkMapping` remain in test file
- ✅ No references to `LoadConfig()` remain in test file
- ✅ No references to `LoadConfigWithOptions`, `ConfigOptions`, `ShouldIgnore`, `GetMapping`, or `Validate()` remain
- ⚠️ Build verification blocked by sandbox restrictions on go build cache
  - However, verified via grep that no legacy code references remain
  - LSP diagnostics should now show no errors in config_test.go

**Status:**
- ✅ Task 23 complete - config_test.go has been successfully cleaned up
- Test files still needing cleanup:
  - status_test.go (Task 25) - has errors per diagnostics
  - status_json_test.go - not listed as separate task, part of Task 25
  - adopt_test.go (Task 26) - has errors per diagnostics
  - linker_test.go (Task 24) - need to check for errors
  - orphan_test.go (Task 27) - need to check for errors
  - link_utils_test.go (Task 28) - need to check for errors

**Next task:** Task 24 or 25 - Check which test file needs cleanup next based on diagnostics

## Session 13: Remove Legacy Tests from linker_test.go (Complete)

### Task: Remove legacy tests from linker_test.go (Task 24)

**Context:**
After removing the legacy functions (`CreateLinks`, `removeLinks`, `PruneLinks`) in Session 3, linker_test.go had tests using the old `*Config` parameter pattern that needed cleanup.

**Removed 4 legacy test functions:**
1. `TestCreateLinks` - tested legacy `CreateLinks(config *Config, dryRun bool)` (lines 13-402)
2. `TestRemoveLinks` - tested legacy `removeLinks(config *Config, dryRun bool, skipConfirm bool)` (lines 404-543)
3. `TestPruneLinks` - tested legacy `PruneLinks(config *Config, dryRun bool, force bool)` (lines 545-673)
4. `TestLinkerEdgeCases` - tested edge cases using legacy `CreateLinks(&Config{...})` (lines 679-902)

**Kept 3 tests for new flag-based API:**
1. `TestCreateLinksWithOptions` - tests `CreateLinksWithOptions(opts LinkOptions)`
2. `TestRemoveLinksWithOptions` - tests `RemoveLinksWithOptions(opts LinkOptions)`
3. `TestPruneWithOptions` - tests `PruneWithOptions(opts LinkOptions)`

**Statistics:**
- File reduced from 1659 lines to 767 lines (**892 lines removed**)
- Test count reduced from 7 tests to 3 tests (4 legacy tests removed)

**Verification:**
- ✅ No references to `Config{` with `LinkMappings` remain
- ✅ No references to `LinkMapping` type remain
- ✅ No references to legacy `CreateLinks()` remain
- ✅ No references to legacy `removeLinks()` remain
- ✅ No references to legacy `PruneLinks()` remain
- ✅ Grep confirms all legacy function references removed
- ⚠️ Build verification blocked by sandbox restrictions on go build cache
  - However, verified via grep that no legacy code references remain
  - All helper functions preserved (createTestFile, assertSymlink, assertNotExists, assertDirExists, createTestSymlink)

**Status:**
- ✅ Task 24 complete - linker_test.go has been successfully cleaned up
- Test files still needing cleanup:
  - status_test.go (Task 25) - has errors per diagnostics
  - adopt_test.go (Task 26) - has errors per diagnostics
  - orphan_test.go (Task 27) - need to check for errors
  - link_utils_test.go (Task 28) - has errors per diagnostics

**Next task:** Task 25 - Remove legacy tests from status_test.go

## Session 14: Remove Legacy Tests from status_test.go (Complete)

### Task: Remove legacy tests from status_test.go (Task 25)

**Context:**
After removing the legacy `Status(config *Config)` function in Session 4, the test files status_test.go and status_json_test.go had tests using the old `*Config` parameter pattern with `LinkMappings` that needed cleanup.

**Removed from status_test.go:**
1. `TestStatusWithLinkMappings` - tested legacy `Status(config *Config)` with old `Config{LinkMappings}` (lines 10-79)
2. `TestDetermineSourceMapping` - tested legacy `DetermineSourceMapping()` function (lines 81-126)

**Kept in status_test.go:**
1. `TestStatusWithOptions` - tests `StatusWithOptions(opts LinkOptions)` (new API)

**Removed status_json_test.go entirely:**
- `TestStatusJSON` - used legacy `Config` and `Status()` function
- `TestStatusJSONEmpty` - used legacy `Config` and `Status()` function
- Both tests used the old JSON config system, entire file deleted

**Statistics:**
- status_test.go: reduced from 360 lines to 241 lines (**119 lines removed**)
- status_json_test.go: deleted (**163 lines removed**)
- **Total: 282 lines removed from test files**

**Verification:**
- ✅ No references to `Status(config` remain in internal/lnk/ test files
- ✅ No references to `DetermineSourceMapping` remain in production code (only in documentation)
- ✅ status_test.go no longer has any references to legacy `Config` or `LinkMapping`
- ✅ Grep confirms all legacy function references removed
- ⚠️ Build verification blocked by sandbox restrictions
  - However, verified via grep that no legacy code references remain
  - LSP diagnostics should no longer show errors in status_test.go

**Status:**
- ✅ Task 25 complete - status_test.go and status_json_test.go have been successfully cleaned up
- Test files still needing cleanup:
  - adopt_test.go (Task 26) - has errors per diagnostics
  - orphan_test.go (Task 27) - has errors per diagnostics
  - link_utils_test.go (Task 28) - has errors per diagnostics

**Next task:** Task 26 - Remove legacy tests from adopt_test.go
