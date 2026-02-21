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
