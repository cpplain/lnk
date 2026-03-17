# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`lnk` is an opinionated symlink manager for dotfiles written in Go. It recursively traverses source directories and creates individual symlinks for each file (not directories), allowing mixed file sources in the same target directory.

## Development Commands

```bash
# Build
make build                  # Build binary to bin/lnk with version from git tags

# Testing
make test                   # Run all tests (unit + e2e)
make test-unit              # Run unit tests only (lnk/)
make test-e2e               # Run e2e tests only (test/)
make test-coverage          # Generate coverage report (coverage.html)

# Code Quality
make fmt                    # Format code (prefers goimports, falls back to gofmt)
make lint                   # Run go vet
make check                  # Run fmt, test, and lint in sequence
```

## Architecture

### Core Components

**Commands (`main.go` and `lnk/`):**

- **main.go**: CLI entry point with subcommand-based interface (`lnk <command> [flags] <source-dir>`). Commands: `create`, `remove`, `status`, `prune`, `adopt`, `orphan`. For all commands, `source-dir` is the first required positional argument. For `adopt`/`orphan`: one or more file paths are required as additional positional arguments (`<source-dir> <path...>`). Uses stdlib `flag` package with `extractCommand()` to support flags before or after the command name.
- **lnk/create.go**: 3-phase execution: collect (walk source dir, apply `PatternMatcher`), validate all targets via `ValidateSymlinkCreation` (all-or-nothing), execute via `CreateSymlink`. Continue-on-failure during execution.
- **lnk/remove.go**: Walks source dir to compute expected symlink paths, verifies each is a managed symlink, removes matches. Continue-on-failure. Calls `CleanEmptyDirs` on parent dirs afterward.
- **lnk/status.go**: Calls `FindManagedLinks`, categorizes links as active/broken, reports results. Read-only.
- **lnk/prune.go**: Calls `FindManagedLinks`, filters to broken-only, removes them. Continue-on-failure. Calls `CleanEmptyDirs`.
- **lnk/adopt.go**: 2-phase transactional: validate all paths first, then execute with full rollback on any failure. Uses `MoveFile`, `CreateSymlink`, `validateAdoptSource`.
- **lnk/orphan.go**: 2-phase transactional (inverse of adopt): validate, then execute with rollback. Uses `FindManagedLinks`, `RemoveSymlink`, `MoveFile`, `CleanEmptyDirs`. Permission restoration is best-effort.

**Configuration (`lnk/config.go`):**

Loads and merges configuration from all sources. `LoadIgnoreFile(sourceDir)` parses `<sourceDir>/.lnkignore`. `LoadConfig(sourceDir, cliIgnorePatterns)` merges: built-in defaults + `.lnkignore` patterns + CLI `--ignore` patterns, in that order (later patterns can negate earlier ones with `!pattern`).

**Shared internals:**

- **lnk/symlink.go**: `ManagedLink` struct (`Path`, `Target`, `IsBroken`, `Source`), `FindManagedLinks(startPath, sources)`, `CreateSymlink`, `RemoveSymlink`
- **lnk/file_ops.go**: `MoveFile` (`os.Rename` fast path, cross-device copy+verify+delete fallback), `CleanEmptyDirs(dirs, boundaryDir)`
- **lnk/patterns.go**: `PatternMatcher` with gitignore-style matching (`**`, `!` negation, trailing `/` for directories)
- **lnk/validation.go**: `ValidateSymlinkCreation(source, target)` (checks same-path, circular reference, overlapping paths)

**Infrastructure:**

- **lnk/errors.go**: Custom error types (see Error Handling)
- **lnk/exit_codes.go**: `ExitError` (1), `ExitUsage` (2)
- **lnk/output.go**: All print functions (see Output System)
- **lnk/terminal.go**: `isTerminal()`, `ShouldSimplifyOutput()`
- **lnk/color.go**: ANSI color functions (`Red`, `Green`, `Yellow`, `Cyan`, `Bold`), lazy init via `sync.Once`
- **lnk/verbosity.go**: `VerbosityNormal`, `VerbosityVerbose`
- **lnk/constants.go**: Shared constants (skip dirs, icons, formatting)

### Key Design Patterns

**Recursive File Linking**: lnk creates symlinks for individual files, NOT directories. This allows:

- Multiple source directories can map to the same target
- Local-only files can coexist with managed configs
- Parent directories are created as regular directories, never symlinks

**Traversal Strategies**: Two approaches depending on the command:

- **Source walk** (`create`, `remove`): Walk `SourceDir` with `filepath.WalkDir`, compute expected target paths. Efficient because source directories are small.
- **Target walk** via `FindManagedLinks` (`status`, `prune`, `orphan`): Walk `TargetDir` to discover all symlinks pointing into `SourceDir`. Necessary when the source files may no longer exist. Skips `Library` and `.Trash` on macOS.

**Error Handling**: Uses custom error types in `lnk/errors.go`, all implementing the `HintableError` interface (`GetHint() string`):

- `PathError`: filesystem path failures (file not found, permission denied)
- `LinkError`: symlink operation failures with source and target
- `ValidationError`: invalid configuration or arguments
- `HintedError`: wraps any arbitrary error with a hint via `WithHint(err, hint)`
- `LinkExistsError`: non-fatal signal that symlink already points to correct target; caller skips silently
- Sentinel errors: `ErrNotSymlink`, `ErrAlreadyAdopted` (used with `errors.Is`)
- **Error propagation models:**
  - Continue-on-failure (`create`, `remove`, `prune`): validation is all-or-nothing; execution continues on per-item failure, counts failures, returns aggregate error
  - Transactional rollback (`adopt`, `orphan`): all validations pass before any changes; any execution failure triggers reverse-order rollback
- **Exit codes**: 0 (success), 1 (`ExitError` — runtime error), 2 (`ExitUsage` — bad flags, missing args, unknown command)

**Output System**: Centralized in `lnk/output.go`:

- **Print functions**: `PrintSuccess`, `PrintError`, `PrintWarning`, `PrintSkip`, `PrintDryRun`, `PrintInfo`, `PrintDetail`, `PrintVerbose`, `PrintCommandHeader`, `PrintErrorWithHint(err)`
- **Specialized**: `PrintSummary(format, args...)`, `PrintNextStep(command, sourceDir, description)`, `PrintDryRunSummary()`, `PrintEmptyResult(itemType)`
- **Terminal vs piped**: `ShouldSimplifyOutput()` gates icons and colors. Piped output uses plain prefixes (`success`, `error:`, `warning:`, `dry-run:`). `PrintCommandHeader` outputs nothing when piped.
- **Streams**: stdout for normal output; stderr for errors and warnings
- **Color**: enabled when no `--no-color`, no `NO_COLOR` env var, and stdout is a TTY. Colors computed lazily via `sync.Once` in `color.go`.
- **Verbosity**: two levels — `VerbosityNormal` (default) and `VerbosityVerbose` (`-v`). Only `PrintVerbose` is suppressed at normal level.

**Terminal Detection**: `terminal.go` detects TTY for conditional formatting (colors, piped output simplification)

### Configuration Structure

**`LoadConfig(sourceDir, cliIgnorePatterns)` algorithm** (from `docs/design/config.md`):
1. Call `LoadIgnoreFile(sourceDir)` to parse `<sourceDir>/.lnkignore` (if present)
2. Combine: `getBuiltInIgnorePatterns()` + `.lnkignore` patterns + CLI `--ignore` patterns
3. Expand `~` to home directory via `ExpandPath("~")`
4. Return `Config{SourceDir, TargetDir, IgnorePatterns}`

**Path helpers** (in `lnk/config.go`):
- `ExpandPath(path)`: expands `~` to home directory; other paths returned unchanged
- `ContractPath(path)`: contracts home directory to `~` for display output

```go
// Final resolved configuration used by all operations
type Config struct {
    SourceDir      string   // Source directory (from CLI positional arg)
    TargetDir      string   // Target directory (always ~; configurable in tests)
    IgnorePatterns []string // Combined ignore patterns from all sources
}

// Options for linking operations
type LinkOptions struct {
    SourceDir      string   // source directory - what to link from (e.g., ~/git/dotfiles)
    TargetDir      string   // where to create links (always ~; configurable in tests)
    IgnorePatterns []string // combined ignore patterns from all sources
    DryRun         bool     // preview mode without making changes
}

// Options for adopt operations
type AdoptOptions struct {
    SourceDir string   // base directory for dotfiles (e.g., ~/git/dotfiles)
    TargetDir string   // where files currently are (always ~; configurable in tests)
    Paths     []string // files to adopt (e.g., ["~/.bashrc", "~/.vimrc"])
    DryRun    bool     // preview mode
}

// Options for orphan operations
type OrphanOptions struct {
    SourceDir string   // base directory for dotfiles (e.g., ~/git/dotfiles)
    TargetDir string   // where symlinks are (always ~; configurable in tests)
    Paths     []string // symlink paths to orphan (e.g., ["~/.bashrc", "~/.vimrc"])
    DryRun    bool     // preview mode
}
```

### Testing Structure

- **Unit tests**: `lnk/*_test.go` - use `testutil_test.go` helpers for temp dirs
- **E2E tests**: `test/e2e_test.go` - full workflow testing
- Test data: Use `test/helpers_test.go` for creating test repositories

## Development Guidelines

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` - new feature
- `fix:` - bug fix
- `docs:` - documentation only
- `refactor:` - code restructuring
- `test:` - adding/updating tests
- `chore:` - build/tooling changes

Breaking changes use `!` suffix: `feat!:` or `BREAKING CHANGE:` in footer.

### CLI Design Principles

From [cpplain/cli-design](https://github.com/cpplain/cli-design):

- **Obvious Over Clever**: Make intuitive paths easiest
- **Helpful Over Minimal**: Provide clear guidance and error messages
- **Consistent Over Special**: Follow CLI conventions
- All destructive operations support `--dry-run`

### Code Standards

- Use `PrintVerbose()` for debug output (hidden unless --verbose)
- Use `PrintErrorWithHint()` for user-facing errors with actionable hints
- Use `PrintWarningWithHint()` for non-fatal per-item errors in continue-on-failure commands
- Expand paths with `ExpandPath()` to handle `~/` notation
- Use `ContractPath()` for all user-facing display paths
- Validate paths early using functions in `validation.go`
- Return typed errors with hints from library functions; let `main` handle display

## Common Tasks

### Adding a New Operation

1. Add new subcommand case to the `switch` in `main.go` and write a `handleX()` function
2. Create options struct in `lnk/` following the pattern (e.g., `NewOperationOptions`)
3. Implement operation function in `lnk/` (e.g., `func NewOperation(opts NewOperationOptions) error`)
4. Add the command name to `suggestCommand()` valid commands list in `main.go`
5. Add `printCommandHelp()` case for the new command in `main.go`
6. Add tests in `lnk/xxx_test.go`
7. Add e2e test if appropriate

### Modifying Configuration

- Config types in `config.go` are simple structs for holding configuration
- Add validation with helpful hints using `NewValidationErrorWithHint()`
- Ignore files use gitignore-style format: one pattern per line (e.g., `local/`, `*.secret`)

### Running Single Test

```bash
go test -v ./lnk -run TestFunctionName
go test -v ./test -run TestE2EName
```

## Technical Notes

- Version info injected via ldflags during build (version, commit, date)
- No external dependencies - stdlib only
- Git operations are optional (detected at runtime)
- Uses stdlib `flag` package for command-line parsing
