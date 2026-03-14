# Configuration System Specification

---

## 1. Overview

### Purpose

The `lnk` configuration system merges settings from multiple sources — built-in
defaults, config files, and CLI arguments — into a single resolved `Config` that all
operations use.

### Goals

- **Layered precedence**: CLI arguments always win; built-in defaults always lose
- **Two config file formats**: flag-style `.lnkconfig` and gitignore-style `.lnkignore`
- **XDG-aware discovery**: finds config files in standard locations
- **Additive ignore patterns**: all sources contribute; CLI can negate with `!`
- **Forward compatible**: unknown keys in `.lnkconfig` are silently ignored

### Non-Goals

- GUI configuration editor
- Remote or synchronized configuration
- Per-command configuration (all config applies globally)

---

## 2. Configuration Sources and Precedence

### Precedence for Target Directory

Higher source wins:

| Priority    | Source                        | Example                          |
| ----------- | ----------------------------- | -------------------------------- |
| 1 (highest) | CLI positional argument       | `lnk create ~/git/dotfiles /tmp` |
| 2           | `.lnkconfig` `--target` value | `--target=~` in config file      |
| 3 (lowest)  | Built-in default              | `~` (user home directory)        |

### Precedence for Ignore Patterns

All sources are **combined** (not overridden). Patterns are appended in order,
allowing later patterns to negate earlier ones using `!prefix`:

```
final = built-in defaults + .lnkconfig patterns + .lnkignore patterns + CLI --ignore patterns
```

This ordering means CLI `--ignore` patterns are processed last and can negate
earlier patterns using `!pattern` syntax.

---

## 3. Config File Discovery

`loadConfigFile(sourceDir)` searches the following paths in order and uses the
**first file found**:

| Priority | Path                          | Description                               |
| -------- | ----------------------------- | ----------------------------------------- |
| 1        | `<source-dir>/.lnkconfig`     | Repo-specific config (checked in to repo) |
| 2        | `$XDG_CONFIG_HOME/lnk/config` | XDG user config dir                       |
| 3        | `~/.config/lnk/config`        | Fallback if `$XDG_CONFIG_HOME` not set    |
| 4        | `~/.lnkconfig`                | Legacy home directory config              |

`$XDG_CONFIG_HOME` defaults to `~/.config` when the environment variable is not set.

The `.lnkignore` file is always loaded from `<source-dir>/.lnkignore` only; it does
not participate in the multi-location discovery.

---

## 4. .lnkconfig Format

The `.lnkconfig` file uses stow-style flag syntax: one flag per line.

### Rules

- Empty lines and lines beginning with `#` are ignored (comments)
- Every non-comment, non-empty line must begin with `--`
- Values use `--flag=value` or `--flag value` forms
- Unknown flags are silently ignored (forward compatibility)
- Parsing errors (malformed flag lines) return an error

### Supported Keys

| Key        | Example           | Description                        |
| ---------- | ----------------- | ---------------------------------- |
| `--target` | `--target=~`      | Set target directory               |
| `--ignore` | `--ignore=local/` | Add an ignore pattern (repeatable) |

### Example

```
# lnk configuration for this dotfiles repo
--target=~
--ignore=local/
--ignore=*.secret
```

---

## 5. .lnkignore Format

The `.lnkignore` file uses gitignore syntax.

### Rules

- Empty lines and lines beginning with `#` are ignored
- Each non-comment line is a pattern
- Patterns are appended to the ignore list after `.lnkconfig` patterns
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

## 6. Built-in Ignore Patterns

These patterns are always active and cannot be removed (they appear first in the
pattern list, so they can be negated by a later `!pattern` if needed):

```
.git
.gitignore
.DS_Store
*.swp
*.tmp
README*
LICENSE*
CHANGELOG*
.lnkconfig
.lnkignore
```

---

## 7. Configuration Types

```go
// FileConfig holds values loaded from a .lnkconfig file
type FileConfig struct {
    Target         string   // target directory from config file
    IgnorePatterns []string // ignore patterns from config file
}

// Config is the final merged configuration used by all operations
type Config struct {
    SourceDir      string   // source directory (from CLI positional arg)
    TargetDir      string   // resolved target directory
    IgnorePatterns []string // combined ignore patterns from all sources
}
```

---

## 8. LoadConfig Algorithm

```go
func LoadConfig(sourceDir, cliTarget string, cliIgnorePatterns []string) (*Config, error)
```

1. Call `loadConfigFile(sourceDir)` to find and parse the first `.lnkconfig` found
2. Call `LoadIgnoreFile(sourceDir)` to parse `<sourceDir>/.lnkignore` (if it exists)
3. Resolve target directory using precedence rules:
   - If `cliTarget != ""`, use `cliTarget`
   - Else if `fileConfig.Target != ""`, use `fileConfig.Target`
   - Else use `"~"` (default)
4. Build combined ignore patterns:
   ```
   patterns = getBuiltInIgnorePatterns()
            + fileConfig.IgnorePatterns
            + ignoreFilePatterns
            + cliIgnorePatterns
   ```
5. Return `Config{SourceDir: sourceDir, TargetDir: targetDir, IgnorePatterns: patterns}`

---

## 9. Path Handling

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

## 10. Verbose Logging

When `--verbose` is active, `LoadConfig` logs:

- Each config file path checked during discovery
- Which config file was loaded (or that none was found)
- Which source provided the target directory
- Count of patterns from each source and total

---

## 11. Examples

### Minimal (no config files)

```sh
lnk create .
# Uses: target=~, built-in ignores only
```

### Config file sets target

```
# ~/git/dotfiles/.lnkconfig
--target=~
--ignore=local/
```

```sh
lnk create ~/git/dotfiles
# Uses: target=~, built-in + local/ ignores
```

### CLI overrides config file target

```sh
lnk create ~/git/dotfiles /tmp
# Uses: target=/tmp (CLI positional wins), built-in + local/ ignores
```

### Negating a built-in pattern

```sh
lnk create --ignore '!README*' .
# README files are now included (negates built-in README* pattern)
```

---

## 12. Related Specifications

- [cli.md](cli.md) — Flag definitions and parsing
- [create.md](create.md) — How ignore patterns are applied during link collection
- [output.md](output.md) — Verbose logging conventions
