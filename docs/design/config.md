# Configuration System Specification

---

## 1. Overview

### Purpose

The `lnk` configuration system merges settings from multiple sources — built-in
defaults, an optional `.lnkignore` file, and CLI arguments — into a single resolved
`Config` that all operations use.

### Goals

- **Single ignore file**: gitignore-style `.lnkignore` for per-repo ignore patterns
- **Additive ignore patterns**: all sources contribute; CLI can negate with `!`
- **Simple discovery**: `.lnkignore` is always loaded from the source directory only

### Non-Goals

- GUI configuration editor
- Remote or synchronized configuration
- Per-command configuration (all config applies globally)

---

## 2. Configuration Sources

### Target Directory

The target directory is always `~` (the user's home directory) for all commands.
It is not configurable via CLI argument. `TargetDir` is retained in the Go-level
options structs for testability.

### Ignore Patterns

All sources are **combined** (not overridden). Patterns are appended in order,
allowing later patterns to negate earlier ones using `!prefix`:

```
final = built-in defaults + .lnkignore patterns + CLI --ignore patterns
```

This ordering means CLI `--ignore` patterns are processed last and can negate
earlier patterns using `!pattern` syntax.

---

## 3. .lnkignore Format

The `.lnkignore` file is always loaded from `<source-dir>/.lnkignore`. It uses
gitignore syntax.

### Rules

- Empty lines and lines beginning with `#` are ignored
- Each non-comment line is a pattern
- Patterns are appended to the ignore list after built-in patterns
- Negation with `!` is supported

### Example

```
# Machine-specific files
local/
*.secret

# Temporary files
*.swp
*.tmp
```

---

## 4. Built-in Ignore Patterns

These patterns are always present at the start of the pattern list. Later `!pattern`
entries (from `.lnkignore` or `--ignore`) can negate them if needed:

```
.git
.gitignore
.DS_Store
*.swp
*.tmp
README*
LICENSE*
CHANGELOG*
.lnkignore
```

---

## 5. Configuration Types

```go
// Config is the final merged configuration used by all operations
type Config struct {
    SourceDir      string   // source directory (from CLI positional arg)
    TargetDir      string   // target directory (always ~ from CLI; configurable in tests)
    IgnorePatterns []string // combined ignore patterns from all sources
}
```

---

## 6. LoadConfig Algorithm

```go
func LoadConfig(sourceDir string, cliIgnorePatterns []string) (*Config, error)
```

1. Call `LoadIgnoreFile(sourceDir)` to parse `<sourceDir>/.lnkignore` (if it exists)
2. Build combined ignore patterns:
   ```
   patterns = getBuiltInIgnorePatterns()
            + ignoreFilePatterns
            + cliIgnorePatterns
   ```
3. Expand `~` to the user's home directory via `ExpandPath("~")`
4. Return `Config{SourceDir: sourceDir, TargetDir: homeDir, IgnorePatterns: patterns}`

---

## 7. Path Handling

### ExpandPath

`ExpandPath(path string) (string, error)` expands `~` to the user home directory:

- `~` → `/home/user`
- `~/foo` → `/home/user/foo`
- Absolute paths and relative paths are returned unchanged
- Returns error if home directory cannot be determined

### ContractPath

`ContractPath(path string) string` contracts home directory back to `~` for display:

- `/home/user/foo` → `~/foo`
- `/home/user` → `~`
- Other paths returned unchanged
- On error looking up home directory, returns the original path unchanged

---

## 8. Verbose Logging

When `--verbose` is active, `LoadConfig` logs:

- Whether `.lnkignore` was found in the source directory
- Count of patterns from each source and total

---

## 9. Examples

### Minimal (no .lnkignore)

```sh
lnk create .
# Uses: target=~, built-in ignores only
```

### With .lnkignore

```
# ~/git/dotfiles/.lnkignore
local/
*.secret
```

```sh
lnk create ~/git/dotfiles
# Uses: target=~, built-in + local/ + *.secret ignores
```

### Negating a built-in pattern

```sh
lnk create --ignore '!README*' .
# README files are now included (negates built-in README* pattern)
```

---

## 10. Related Specifications

- [cli.md](cli.md) — Flag definitions and parsing
- [create.md](create.md) — How ignore patterns are applied during link collection
- [output.md](output.md) — Verbose logging conventions
