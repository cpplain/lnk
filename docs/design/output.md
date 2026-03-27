# Output System Specification

---

## 1. Overview

### Purpose

The `lnk` output system provides a consistent, context-aware set of print functions
used by all commands. Output adapts based on verbosity level, terminal detection,
and color settings.

### Goals

- **Consistency**: all commands use the same print functions; visual language is uniform
- **Terminal-aware**: icons and colors when connected to a terminal; plain prefixes when piped
- **Verbosity-aware**: verbose mode adds debug info beyond standard informational output
- **Stderr for errors and warnings**: informational output to stdout; errors to stderr

### Non-Goals

- Structured (JSON) output format
- Localization
- Progress bars for long operations (beyond the 1-second delay threshold)

---

## 2. Verbosity Levels

Two levels, controlled by `SetVerbosity(level VerbosityLevel)`:

| Level | Constant           | Flag               | Description                       |
| ----- | ------------------ | ------------------ | --------------------------------- |
| 0     | `VerbosityNormal`  | (default)          | Standard informational output     |
| 1     | `VerbosityVerbose` | `-v` / `--verbose` | Standard output plus debug detail |

Global state: `verbosity` defaults to `VerbosityNormal`. Set once at startup by
`main` before any operations run.

---

## 3. Terminal Detection

### isTerminal()

Returns `true` if stdout is a character device (TTY):

```go
func isTerminal() bool {
    fi, err := os.Stdout.Stat()
    if err != nil {
        return false
    }
    return (fi.Mode() & os.ModeCharDevice) != 0
}
```

### ShouldSimplifyOutput()

Returns `true` when stdout is **not** a terminal (i.e., piped or redirected).
When true, all output uses plain text prefixes instead of icons and colors.

```go
func ShouldSimplifyOutput() bool {
    return !isTerminal()
}
```

---

## 4. Color Support

Color output is enabled when all of the following are true:

1. `--no-color` flag was not passed
2. `NO_COLOR` environment variable is not set (any non-empty value disables color;
   see [no-color.org](https://no-color.org/))
3. stdout is a terminal (`isTerminal()` returns true)

`SetNoColor(true)` disables colors globally. It must be called before any colorized
output is produced (i.e., as the first thing after flag parsing).

Color is computed lazily via `sync.Once` and cached. Calling `SetNoColor` resets the
cache.

### Color Functions

| Function    | ANSI         | Use                            |
| ----------- | ------------ | ------------------------------ |
| `Red(s)`    | `\033[0;31m` | Errors, broken links           |
| `Green(s)`  | `\033[0;32m` | Success, active links          |
| `Yellow(s)` | `\033[0;33m` | Warnings, skip, dry-run prefix |
| `Cyan(s)`   | `\033[0;36m` | `Try:` hint label              |
| `Bold(s)`   | `\033[1m`    | Command headers                |

When color is disabled, all functions return the input string unchanged.

---

## 5. Output Functions

### Terminal vs. Piped Formats

Each function has two output modes:

| Function             | Terminal                                | Piped                          |
| -------------------- | --------------------------------------- | ------------------------------ |
| `PrintSuccess`       | `✓ <message>` (green icon)              | `success <message>`            |
| `PrintError`         | `✗ Error: <message>` (red icon, stderr) | `error: <message>` (stderr)    |
| `PrintWarning`       | `! <message>` (yellow icon, stderr)     | `warning: <message>` (stderr)  |
| `PrintSkip`          | `○ <message>` (yellow icon)             | `skip <message>`               |
| `PrintDryRun`        | `[DRY RUN] <message>` (yellow prefix)   | `dry-run: <message>`           |
| `PrintInfo`          | `<message>` (no prefix)                 | `<message>` (no prefix)        |
| `PrintDetail`        | `  <message>` (2-space indent)          | `  <message>` (2-space indent) |
| `PrintVerbose`       | `[VERBOSE] <message>`                   | `[VERBOSE] <message>`          |
| `PrintCommandHeader` | bold `<text>` + blank line              | nothing (no output)            |

`PrintWarningWithHint` is documented below in Specialized Functions; it does not appear
in this table because its output is composed from `PrintWarning` formatting plus a
dedicated hint line, both on stderr.

### Verbosity Gating

| Function             | Normal     | Verbose |
| -------------------- | ---------- | ------- |
| `PrintSuccess`       | shown      | shown   |
| `PrintInfo`          | shown      | shown   |
| `PrintDetail`        | shown      | shown   |
| `PrintSkip`          | shown      | shown   |
| `PrintDryRun`        | shown      | shown   |
| `PrintVerbose`       | suppressed | shown   |
| `PrintError`         | shown      | shown   |
| `PrintWarning`       | shown      | shown   |
| `PrintCommandHeader` | shown      | shown   |

### Specialized Functions

#### PrintErrorWithHint(err error)

Extracts a hint from the error (via `GetErrorHint`) and displays:

- Terminal: `"✗ Error: <err>"` on stderr; if hint present: `"  Try: <hint>"` (cyan `Try:`)
- Piped: `"error: <err>"` on stderr; if hint present: `"hint: <hint>"` on stderr

Always writes to stderr. Not gated by verbosity (errors are always shown).

#### PrintWarningWithHint(err error)

Prints a warning with an optional hint extracted from the error. Used by
continue-on-failure commands (`create`, `remove`, `prune`) for per-item failures.

Extracts a hint from the error (via `GetErrorHint`). Always writes to stderr.
Not gated by verbosity (warnings are always shown).

- Terminal: `"! <err>"` (yellow icon, stderr); if hint present: `"  Try: <hint>"`
  (cyan `Try:` label, stderr)
- Piped: `"warning: <err>"` (stderr); if hint present: `"hint: <hint>"` (stderr)

Each command formats the error message before passing it (e.g.,
`fmt.Errorf("Failed to create %s: %w", ContractPath(path), err)`), then calls
`PrintWarningWithHint(formattedErr)`. This gives each command control over the
prefix and path formatting while centralizing hint extraction and display.

#### PrintCommandHeader(text string)

```go
func PrintCommandHeader(text string) {
    if !ShouldSimplifyOutput() {
        fmt.Println(Bold(text))
        fmt.Println() // blank line after header, terminal only
    }
}
```

#### PrintSummary(format string, args ...interface{})

Prints a blank line followed by a `PrintSuccess` call. Provides visual separation
between operation output and the summary line.

#### PrintEmptyResult(itemType string)

Prints `"No <itemType> found."` via `PrintInfo`. This is a convenience helper for
generic cases. Commands that need more specific phrasing (e.g., `"No files to link
found."`, `"No broken symlinks found."`) should call `PrintInfo` directly with a
custom message.

#### PrintNextStep(command, sourceDir, description string)

Contracts `sourceDir` via `ContractPath`, then prints
`"Next: Run 'lnk <command> <contractedSourceDir>' to <description>"` via `PrintInfo`.
Callers pass the expanded absolute path; this function handles display formatting.

#### PrintDryRunSummary()

Prints `"No changes made in dry-run mode"` via `PrintInfo`.

---

## 6. Standard Output Flow

Every command follows this output structure:

```
1. PrintCommandHeader("Command Name")   ← bold header + blank line

2. [per-item output]
   PrintSuccess / PrintError / PrintSkip / PrintDryRun

3. PrintSummary(...)                    ← blank line + success icon + count

4. PrintNextStep(...) [optional]        ← "Next: Run 'lnk status <source-dir>' to ..."
```

Empty result:

```
1. PrintCommandHeader("Command Name")
2. PrintEmptyResult("items")            ← "No items found."
```

Dry-run:

```
1. PrintCommandHeader("Command Name")
2. PrintDryRun("Would do X ...")
3. blank line
4. PrintDryRunSummary()
```

---

## 7. Stream Assignment

| Output                                          | Stream |
| ----------------------------------------------- | ------ |
| Normal output (success, info, dry-run, verbose) | stdout |
| Errors                                          | stderr |
| Warnings                                        | stderr |

This allows stdout to be piped (e.g., `lnk status . | grep broken`) without error
messages corrupting the stream.

---

## 8. Related Specifications

- [cli.md](cli.md) — Verbosity flag definitions (`--verbose`, `--no-color`)
- [error-handling.md](error-handling.md) — `PrintErrorWithHint` and error display
