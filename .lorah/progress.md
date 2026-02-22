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
