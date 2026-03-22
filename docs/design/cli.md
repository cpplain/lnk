# CLI Specification

---

## 1. Overview

### Purpose

`lnk` is an opinionated symlink manager for dotfiles. The CLI uses a subcommand-based
interface: a named command selects the operation, and global flags configure behavior
shared across all commands.

### Goals

- **Subcommand-based**: `lnk <command> [flags] <source-dir> [target...]` mirrors conventions of tools like `ln`
- **Shared flags**: all flags are accepted by all commands; irrelevant flags are silently ignored
- **Helpful on error**: unknown commands suggest the closest match; missing args explain correct usage
- **Composable**: machine-readable output when piped; human-friendly output to terminals

### Non-Goals

- Interactive TUI mode
- Shell completion (future consideration)
- Plugin or extension system

---

## 2. Interface

### Usage

```
lnk <command> [flags] <source-dir>
```

The recommended form places flags after the command name. Flags are also accepted
before the command name for convenience (e.g., `lnk --dry-run create .` works),
but `lnk <command> [flags] <source-dir>` is the canonical style. The
`--` separator stops flag parsing; everything after it is treated as positional
arguments.

### Commands

| Command  | Args                     | Description                           |
| -------- | ------------------------ | ------------------------------------- |
| `create` | `<source-dir>`           | Create symlinks from source to target |
| `remove` | `<source-dir>`           | Remove managed symlinks               |
| `status` | `<source-dir>`           | Show status of managed symlinks       |
| `prune`  | `<source-dir>`           | Remove broken symlinks                |
| `adopt`  | `<source-dir> <path...>` | Adopt files into source directory     |
| `orphan` | `<source-dir> <path...>` | Remove files from management          |

For all commands, `source-dir` is the first required positional argument (the dotfiles
repository directory). The target directory is always `~`. Extra positional arguments
beyond those listed are a usage error (exit 2).

For `adopt`: one or more files or directories within `~` to move into the source
directory are required as the second and subsequent positional arguments.

For `orphan`: one or more managed symlinks or directories within `~` containing
managed symlinks are required as the second and subsequent positional arguments.

### Global Flags

All flags are accepted by all commands.

| Flag               | Short | Default | Description                            |
| ------------------ | ----- | ------- | -------------------------------------- |
| `--ignore PATTERN` |       |         | Additional ignore pattern (repeatable) |
| `--dry-run`        | `-n`  | false   | Preview changes without making them    |
| `--verbose`        | `-v`  | false   | Enable verbose output                  |
| `--no-color`       |       | false   | Disable colored output                 |
| `--version`        | `-V`  |         | Print version and exit                 |
| `--help`           | `-h`  |         | Show help and exit                     |

Notes:

- `--ignore` is repeatable; each use appends a pattern. Only has effect on `create`.
- `--dry-run` is accepted by `status` but has no effect (status never modifies anything).

---

## 3. Behavior

### Startup Sequence

1. Parse all flags and the command name from `os.Args[1:]`
2. Apply `--no-color` before any output is produced
3. Handle `--version`: print `lnk <version>` and exit 0
4. Handle `--help` or bare `lnk` (invoked with no arguments at all): print usage and exit 0
5. Set verbosity level
6. Parse positional arguments: for all commands, the first positional argument is
   `source-dir`; for `adopt` and `orphan`, remaining positional arguments are paths
7. Load configuration via `LoadConfig(sourceDir, cliIgnorePatterns)` (see [config.md](config.md))
8. Build the command's options struct (`LinkOptions`, `AdoptOptions`, or `OrphanOptions`)
   by mapping `Config` fields plus CLI flags (`DryRun`, `Paths`) into the struct
9. Dispatch to the command handler

### Command Dispatch

After parsing, the first non-flag argument is the command name. If no command is
given (and `--version`/`--help` were not used), print usage and exit 2.

```
args = [flag...] command [flag...] [positional...]
```

### Unknown Command Handling

When an unrecognized command is given, suggest the closest match using Levenshtein
distance:

```
lnk statsu
error: unknown command: "statsu"
  Try: Did you mean "status"?
```

Suggestion algorithm:

1. Compute Levenshtein distance between input and each valid command name
2. Select the command with the smallest distance
3. Only suggest if `distance <= len(input)/2 + 1`
4. If no suggestion qualifies, show only the error with a pointer to `--help`

Valid command names for suggestion: `create`, `remove`, `status`, `prune`, `adopt`, `orphan`.

```go
func suggestCommand(input string) string {
    commands := []string{"create", "remove", "status", "prune", "adopt", "orphan"}
    threshold := len(input)/2 + 1
    best, bestDist := "", threshold+1
    for _, cmd := range commands {
        if d := levenshteinDistance(input, cmd); d < bestDist {
            best, bestDist = cmd, d
        }
    }
    return best // empty string means no suggestion
}
```

### Per-Command Help

`lnk <command> --help` prints help scoped to that command and exits 0:

```
lnk create --help

Usage: lnk create [flags] <source-dir>

Create symlinks from source directory to home directory.

Arguments:
  source-dir    Source directory to link from (required)

Flags:
  (all global flags apply)

Examples:
  lnk create .
  lnk create ~/git/dotfiles
  lnk create -n .
```

```
lnk remove --help

Usage: lnk remove [flags] <source-dir>

Remove managed symlinks from home directory.

Arguments:
  source-dir    Source directory whose managed links to remove (required)

Flags:
  (all global flags apply)

Examples:
  lnk remove .
  lnk remove ~/git/dotfiles
  lnk remove -n .
```

```
lnk status --help

Usage: lnk status [flags] <source-dir>

Show status of managed symlinks in home directory.

Arguments:
  source-dir    Source directory to check (required)

Flags:
  (all global flags apply)

Examples:
  lnk status .
  lnk status ~/git/dotfiles
  lnk status ~/git/dotfiles | grep ^broken
```

```
lnk prune --help

Usage: lnk prune [flags] <source-dir>

Remove broken managed symlinks from home directory.

Arguments:
  source-dir    Source directory whose broken links to prune (required)

Flags:
  (all global flags apply)

Examples:
  lnk prune .
  lnk prune ~/git/dotfiles
  lnk prune -n .
```

```
lnk adopt --help

Usage: lnk adopt [flags] <source-dir> <path...>

Adopt files into the source directory.

Arguments:
  source-dir    Source directory to move files into (required)
  path          One or more files or directories to adopt; must be within ~ (required)

Flags:
  (all global flags apply)

Examples:
  lnk adopt . ~/.bashrc
  lnk adopt . ~/.bashrc ~/.vimrc
  lnk adopt ~/git/dotfiles ~/.config/nvim
  lnk adopt -n . ~/.bashrc
```

```
lnk orphan --help

Usage: lnk orphan [flags] <source-dir> <path...>

Remove files from management.

Arguments:
  source-dir    Source directory that manages the files (required)
  path          One or more managed symlinks or directories to orphan; must be within ~ (required)

Flags:
  (all global flags apply)

Examples:
  lnk orphan . ~/.bashrc
  lnk orphan . ~/.bashrc ~/.vimrc
  lnk orphan ~/git/dotfiles ~/.config/nvim
  lnk orphan -n . ~/.bashrc
```

### Version Output

```
lnk <version>
```

Version is injected at build time via `-ldflags`. In development builds, version is
`dev`.

### Usage Output (bare `lnk` or `lnk --help`)

```
Usage: lnk <command> [flags] <source-dir>

An opinionated symlink manager for dotfiles and more

Commands:
  create <source-dir>           Create symlinks from source to ~
  remove <source-dir>           Remove managed symlinks
  status <source-dir>           Show status of managed symlinks
  prune  <source-dir>           Remove broken symlinks
  adopt  <source-dir> <path...> Adopt files into source directory
  orphan <source-dir> <path...> Remove files from management

Flags:
      --ignore PATTERN  Additional ignore pattern, repeatable
  -n, --dry-run         Preview changes without making them
  -v, --verbose         Enable verbose output
      --no-color        Disable colored output
  -V, --version         Show version information
  -h, --help            Show this help message

Examples:
  lnk create .                        Create links from current directory
  lnk create ~/git/dotfiles           Create from absolute path
  lnk create -n .                     Dry-run preview
  lnk remove .                        Remove links
  lnk status .                        Show status
  lnk prune .                         Prune broken symlinks
  lnk prune ~/git/dotfiles            Prune from specific source
  lnk adopt . ~/.bashrc ~/.vimrc      Adopt files into current directory
  lnk adopt ~/dotfiles ~/.bashrc      Adopt with explicit source
  lnk orphan . ~/.bashrc              Remove file from management
  lnk create --ignore '*.swp' .       Add ignore pattern

Config Files:
  .lnkignore in source directory
    Format: gitignore syntax
    Patterns are combined with built-in defaults and --ignore flags
```

---

## 4. Flag Parsing Rules

- Short flags: single dash + single letter (`-n`, `-v`, `-V`, `-h`)
- Long flags: double dash + name (`--dry-run`, `--verbose`, `--ignore`)
- Value flags accept `--flag=value` or `--flag value` forms
- Boolean flags do not accept values (`--dry-run` not `--dry-run=true`)
- `--` terminates flag parsing; all subsequent tokens are positional arguments
- Unknown flags produce a usage error (exit 2) with a hint to run `lnk --help`
- Flags requiring a value but given none produce a usage error

---

## 5. Exit Codes

| Code | Meaning                                                |
| ---- | ------------------------------------------------------ |
| 0    | Success                                                |
| 1    | Runtime error (operation failed)                       |
| 2    | Usage error (bad flags, missing args, unknown command) |

---

## 6. Examples

```sh
# Basic operations
lnk create .                        # Create links from cwd
lnk create ~/git/dotfiles           # Create from explicit path
lnk remove .                        # Remove links from cwd
lnk status .                        # Show status
lnk prune .                         # Prune broken links from cwd
lnk prune ~/git/dotfiles            # Prune from explicit source

# File management
lnk adopt . ~/.bashrc ~/.vimrc      # Adopt files into cwd
lnk adopt ~/dotfiles ~/.bashrc      # Adopt with explicit source dir
lnk orphan . ~/.bashrc              # Orphan file

# Flags
lnk create -n .                     # Dry-run preview
lnk create -v .                     # Verbose output
lnk create --no-color .             # No colored output
lnk create --ignore '*.swp' .       # Extra ignore pattern

# Help
lnk --help                          # Full help
lnk create --help                   # Command-specific help
lnk --version                       # Print version
```

---

## 7. Related Specifications

- [config.md](config.md) — Configuration loading and precedence
- [error-handling.md](error-handling.md) — Error types and exit codes
- [output.md](output.md) — Output formatting and verbosity
