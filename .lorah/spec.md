# lnk CLI Refactor - Stow-like Interface

## Context

lnk currently requires a JSON config file with explicit `LinkMappings` before first use. This adds friction compared to stow's convention-based approach where you just run `stow package` from your dotfiles directory.

## Goals

1. Make config optional - use CLI flags and positional arguments
2. Switch from subcommands to flags (like stow) - lnk is a focused symlink tool
3. Breaking change (pre-v1, no migration needed)

## New Usage Model

```bash
# From dotfiles directory (at least one package required, like stow)
cd ~/git/dotfiles
lnk .                              # Flat repo: link everything (matches stow .)
lnk home                           # Nested repo: links ./home/* -> ~/*
lnk home private/home              # Multiple packages

# Action flags (mutually exclusive)
lnk home                           # default = create links
lnk -C home, lnk --create home     # explicit create
lnk -R home, lnk --remove home     # remove links
lnk -S home, lnk --status home     # show status of links
lnk -P,      lnk --prune           # remove broken symlinks
lnk -A home, lnk --adopt home      # adopt existing files into source
lnk -O path, lnk --orphan path     # move file from source to target

# Directory flags
-s, --source DIR   # Source directory (default: current directory)
-t, --target DIR   # Target directory (default: ~)

# Other flags
--ignore PATTERN   # Additional ignore pattern (gitignore syntax, repeatable)
-n, --dry-run      # Preview mode (stow uses -n)
-v, --verbose      # Verbose output
-q, --quiet        # Quiet mode
--no-color         # Disable colors
-V, --version      # Show version
-h, --help         # Show help
```

**Package argument:** At least one required for link operations. Can be `.` (current directory contents) or subdirectory paths.

## Differences from Stow

| Aspect                 | Stow                            | lnk                              |
| ---------------------- | ------------------------------- | -------------------------------- |
| Action flags           | `-S`, `-D`, `-R`                | `-C/--create`, `-R/--remove`     |
| Source flag            | `-d, --dir`                     | `-s, --source`                   |
| Ignore syntax          | Perl regex                      | Gitignore syntax                 |
| Ignore file            | `.stow-local-ignore`            | `.lnkignore`                     |
| Tree folding           | Yes (`--no-folding` to disable) | No (always links files)          |
| `--restow`             | Yes                             | No (just run remove then create) |
| `--defer`/`--override` | Yes                             | No (not needed)                  |
| `--dotfiles`           | Yes                             | No (not needed)                  |
| `-S/--status`          | No                              | Yes                              |
| `-P/--prune`           | No                              | Yes                              |
| `-O/--orphan`          | No                              | Yes                              |

## Config File (Optional)

Config provides default flags, like stow's `.stowrc`. Discovery order:

1. `.lnkconfig` in source directory (repo-specific)
2. `$XDG_CONFIG_HOME/lnk/config` or `~/.config/lnk/config`
3. `~/.lnkconfig`

Format (CLI flags, one per line):

```
--target=~
--ignore=local/
--ignore=*.private
```

## Ignore File (Optional)

`.lnkignore` in source directory. Gitignore syntax (same as current lnk):

```
# Comments supported
.git
*.swp
README.md
scripts/
```

## Implementation

### Phase 1: Config file support

**Modify: `internal/lnk/config.go`**

- `LoadConfig(sourceDir string) (*Config, error)` - discovers and parses config
- Parse CLI flag format (stow-style, one flag per line)
- Merge with CLI flags (CLI takes precedence)
- Built-in ignore defaults still apply

### Phase 2: Create options-based Linker API

**Modify: `internal/lnk/linker.go`**

Add new struct and functions:

```go
type LinkOptions struct {
    SourceDir      string   // base directory (e.g., ~/git/dotfiles)
    TargetDir      string   // where to create links (default: ~)
    Packages       []string // subdirs to process (e.g., ["home", "private/home"])
    IgnorePatterns []string
    DryRun         bool
}

func CreateLinksWithOptions(opts LinkOptions) error
func RemoveLinksWithOptions(opts LinkOptions) error
func StatusWithOptions(opts LinkOptions) error
func PruneWithOptions(opts LinkOptions) error
```

Refactor `collectPlannedLinks` to take `ignorePatterns []string` instead of `*Config`.

### Phase 3: Rewrite CLI (flag-based)

**Rewrite: `cmd/lnk/main.go`**

Replace subcommand routing with flag-based parsing:

```go
// Action flags (mutually exclusive)
-C, --create    // create links (default if no action flag)
-R, --remove    // remove links
-S, --status    // show status
-P, --prune     // remove broken links
-A, --adopt     // adopt files into source
-O, --orphan PATH   // orphan specific file

// Directory flags
-s, --source DIR   // source directory (default: ".")
-t, --target DIR   // target directory (default: "~")

// Other flags
--ignore PATTERN   // additional ignore pattern
-n, --dry-run      // preview mode
-v, --verbose      // verbose output
-q, --quiet        // quiet mode
--no-color         // disable colors
--version          // show version
-h, --help         // show help
```

Positional args after flags = packages.

### Phase 4: Update internal functions

**`adopt`**: Change to work with `--adopt` flag + packages
**`orphan`**: Change to work with `--orphan PATH` flag
**`prune`**: Change to work with `--prune` flag + optional packages

## Files to Modify

| File                         | Changes                                                              |
| ---------------------------- | -------------------------------------------------------------------- |
| `cmd/lnk/main.go`            | Rewrite - flag-based parsing, remove subcommand routing              |
| `internal/lnk/config.go`     | Rewrite - new config format, discovery, parsing                      |
| `internal/lnk/linker.go`     | Add `LinkOptions`, `*WithOptions()` functions                        |
| `internal/lnk/link_utils.go` | Add `FindManagedLinksForSources(startPath string, sources []string)` |
| `internal/lnk/adopt.go`      | Update to work with new options pattern                              |
| `internal/lnk/orphan.go`     | Update to work with new options pattern                              |

## Test Strategy

1. Add unit tests for new config parsing
2. Add unit tests for `.lnkignore` parsing
3. Add unit tests for `*WithOptions` functions
4. Rewrite e2e tests for new CLI syntax

## Verification

After implementation:

```bash
# Test flat repo
cd ~/git/dotfiles
lnk . -n                           # dry-run create (default action)

# Test nested packages
lnk home -n
lnk home private/home -n

# Test from anywhere
lnk -s ~/git/dotfiles home -n

# Test action flags
lnk -R home -n                     # dry-run remove
lnk -S home                        # status
lnk -P -n                          # dry-run prune

# Test config file
echo "--ignore=local/" > ~/git/dotfiles/.lnkconfig
lnk . -n

# Test adopt/orphan
lnk -A home -n
lnk -O ~/.bashrc -n
```

## Technology Stack

- **Language**: Go (stdlib only, no external dependencies)
- **Build**: Makefile with targets (build, test, test-unit, test-e2e, fmt, lint)
- **Testing**: Go standard testing + e2e test suite
- **Version**: Injected via ldflags

## Success Criteria

- All unit tests pass
- All e2e tests pass with new CLI syntax
- `make build` succeeds
- Verification examples work as expected
- Breaking change is acceptable (pre-v1.0)
