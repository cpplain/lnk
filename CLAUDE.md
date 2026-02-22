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
make test-unit              # Run unit tests only (internal/lnk)
make test-e2e               # Run e2e tests only (e2e/)
make test-coverage          # Generate coverage report (coverage.html)

# Code Quality
make fmt                    # Format code (prefers goimports, falls back to gofmt)
make lint                   # Run go vet
make check                  # Run fmt, test, and lint in sequence
```

## Architecture

### Core Components

- **cmd/lnk/main.go**: CLI entry point with flag-based interface using stdlib `flag` package. Action flags (-C/--create, -R/--remove, -S/--status, -P/--prune, -A/--adopt, -O/--orphan) determine the operation. Positional arguments specify packages to process.

- **internal/lnk/config.go**: Configuration system with `.lnkconfig` file support. Config files can specify target directory and ignore patterns using stow-style format (one flag per line). CLI flags override config file values. Config file search locations:
  1. `.lnkconfig` in source directory
  2. `.lnkconfig` in home directory (~/.lnkconfig)
  3. Built-in defaults if no config found

- **internal/lnk/linker.go**: Symlink operations with 3-phase execution:
  1. Collect planned links (recursive file traversal)
  2. Validate all targets
  3. Execute or show dry-run

- **internal/lnk/adopt.go**: Moves files from target to source directory and creates symlinks

- **internal/lnk/orphan.go**: Removes symlinks and restores actual files to target locations

### Key Design Patterns

**Recursive File Linking**: lnk creates symlinks for individual files, NOT directories. This allows:

- Multiple source directories can map to the same target
- Local-only files can coexist with managed configs
- Parent directories are created as regular directories, never symlinks

**Error Handling**: Uses custom error types in `errors.go`:

- `PathError`: for file operation errors
- `ValidationError`: for validation failures
- `WithHint()`: adds actionable hints to errors

**Output System**: Centralized in `output.go` with support for:

- Text format (default, colorized)
- JSON format (`--output json`)
- Verbosity levels: quiet, normal, verbose

**Terminal Detection**: `terminal.go` detects TTY for conditional formatting (colors, progress bars)

### Configuration Structure

```go
// Config loaded from .lnkconfig file
type FileConfig struct {
    Target         string   // Target directory (default: ~)
    IgnorePatterns []string // Ignore patterns from config file
}

// Final resolved configuration
type Config struct {
    SourceDir      string   // Source directory (from CLI)
    TargetDir      string   // Target directory (CLI > config > default)
    IgnorePatterns []string // Combined ignore patterns from all sources
}

// Options for linking operations
type LinkOptions struct {
    SourceDir      string   // base directory (e.g., ~/git/dotfiles)
    TargetDir      string   // where to create links (default: ~)
    Packages       []string // subdirs to process (e.g., ["home", "private/home"])
    IgnorePatterns []string // combined ignore patterns from all sources
    DryRun         bool     // preview mode without making changes
}

// Options for adopt operations
type AdoptOptions struct {
    SourceDir string   // base directory for dotfiles (e.g., ~/git/dotfiles)
    TargetDir string   // where files currently are (default: ~)
    Package   string   // package to adopt into (e.g., "home" or ".")
    Paths     []string // files to adopt (e.g., ["~/.bashrc", "~/.vimrc"])
    DryRun    bool     // preview mode
}

// Options for orphan operations
type OrphanOptions struct {
    SourceDir string   // base directory for dotfiles (e.g., ~/git/dotfiles)
    TargetDir string   // where symlinks are (default: ~)
    Paths     []string // symlink paths to orphan (e.g., ["~/.bashrc", "~/.vimrc"])
    DryRun    bool     // preview mode
}
```

### Testing Structure

- **Unit tests**: `internal/lnk/*_test.go` - use `testutil_test.go` helpers for temp dirs
- **E2E tests**: `e2e/e2e_test.go` - full workflow testing
- Test data: Use `e2e/helpers_test.go` for creating test repositories

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
- Expand paths with `ExpandPath()` to handle `~/` notation
- Validate paths early using `validation.go` functions
- Always support JSON output mode (`--output json`) for scripting

## Common Tasks

### Adding a New Operation

1. Add new action flag to `main.go` (e.g., `-X/--new-operation`)
2. Create options struct in `internal/lnk/` following the pattern (e.g., `NewOperationOptions`)
3. Implement operation function in `internal/lnk/` (e.g., `func NewOperation(opts NewOperationOptions) error`)
4. Add case in `main.go` to handle the new flag and construct options
5. Add tests in `internal/lnk/xxx_test.go`
6. Add e2e test if appropriate

### Modifying Configuration

- Config types in `config.go` are simple structs for holding configuration
- Add validation with helpful hints using `NewValidationErrorWithHint()`
- Config files use stow-style format: one flag per line (e.g., `--target=~`)

### Running Single Test

```bash
go test -v ./internal/lnk -run TestFunctionName
go test -v ./e2e -run TestE2EName
```

## Technical Notes

- Version info injected via ldflags during build (version, commit, date)
- No external dependencies - stdlib only
- Git operations are optional (detected at runtime)
- Uses stdlib `flag` package for command-line parsing
