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

- **cmd/lnk/main.go**: CLI entry point with manual flag parsing (not stdlib flags for global flags). Routes to command handlers. Uses Levenshtein distance for command suggestions.

- **internal/lnk/config.go**: Configuration system with precedence chain:
  1. `--config` flag
  2. `$XDG_CONFIG_HOME/lnk/config.json`
  3. `~/.config/lnk/config.json`
  4. `~/.lnk.json`
  5. `./.lnk.json` in current directory
  6. Built-in defaults

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
type Config struct {
    IgnorePatterns []string      // Gitignore-style patterns
    LinkMappings   []LinkMapping  // Source-to-target mappings
}

type LinkMapping struct {
    Source string // Absolute path or ~/path
    Target string // Where symlinks are created
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

### Adding a New Command

1. Add command name to `suggestCommand()` in main.go
2. Add case in main switch statement
3. Create handler function following pattern: `handleXxx(args, globalOptions)`
4. Create FlagSet with `-h/--help` usage function
5. Implement command logic in `internal/lnk/`
6. Add tests in `internal/lnk/xxx_test.go`
7. Add e2e test if appropriate

### Modifying Configuration

- Configuration struct in `config.go` must remain JSON-serializable
- Update `Validate()` method when adding fields
- Add validation with helpful hints using `NewValidationErrorWithHint()`

### Running Single Test

```bash
go test -v ./internal/lnk -run TestFunctionName
go test -v ./e2e -run TestE2EName
```

## Technical Notes

- Version info injected via ldflags during build (version, commit, date)
- No external dependencies - stdlib only
- Git operations are optional (detected at runtime)
- Manual flag parsing allows global flags before command name
