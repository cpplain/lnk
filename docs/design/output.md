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
- **Verbosity-aware**: quiet mode suppresses informational output; verbose mode adds debug info
- **Stderr for errors and warnings**: informational output to stdout; errors to stderr

### Non-Goals

- Structured (JSON) output format
- Localization
- Progress bars for long operations (beyond the 1-second delay threshold)

---

## 2. Verbosity Levels

Three levels, controlled by `SetVerbosity(level VerbosityLevel)`:

| Level | Constant           | Flag               | Description                       |
| ----- | ------------------ | ------------------ | --------------------------------- |
| 0     | `VerbosityQuiet`   | `-q` / `--quiet`   | Only errors and warnings          |
| 1     | `VerbosityNormal`  | (default)          | Standard informational output     |
| 2     | `VerbosityVerbose` | `-v` / `--verbose` | Standard output plus debug detail |

`--quiet` and `--verbose` are mutually exclusive; using both is a usage error.

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

| Function             | Terminal                                | Piped                                |
| -------------------- | --------------------------------------- | ------------------------------------ |
| `PrintSuccess`       | `✓ <message>` (green icon)              | `success <message>`                  |
| `PrintError`         | `✗ Error: <message>` (red icon, stderr) | `error: <message>` (stderr)          |
| `PrintWarning`       | `! <message>` (yellow icon, stderr)     | `warning: <message>` (stderr)        |
| `PrintSkip`          | `○ <message>` (yellow icon)             | `skip <message>`                     |
| `PrintDryRun`        | `[DRY RUN] <message>` (yellow prefix)   | `dry-run: <message>`                 |
| `PrintInfo`          | `<message>` (no prefix)                 | `<message>` (no prefix)              |
| `PrintDetail`        | `  <message>` (2-space indent)          | `  <message>` (2-space indent)       |
| `PrintVerbose`       | `[VERBOSE] <message>`                   | `[VERBOSE] <message>`                |
| `PrintCommandHeader` | bold `<text>` + blank line              | blank line only (no header in quiet) |

### Verbosity Gating

| Function             | Quiet                                     | Normal     | Verbose |
| -------------------- | ----------------------------------------- | ---------- | ------- |
| `PrintSuccess`       | suppressed                                | shown      | shown   |
| `PrintInfo`          | suppressed                                | shown      | shown   |
| `PrintDetail`        | suppressed                                | shown      | shown   |
| `PrintSkip`          | suppressed                                | shown      | shown   |
| `PrintDryRun`        | suppressed                                | shown      | shown   |
| `PrintVerbose`       | suppressed                                | suppressed | shown   |
| `PrintError`         | shown                                     | shown      | shown   |
| `PrintWarning`       | shown                                     | shown      | shown   |
| `PrintCommandHeader` | text suppressed, blank line still printed | shown      | shown   |

Note: `PrintCommandHeader` always emits a trailing blank line even in quiet mode to
maintain consistent spacing before operation output.

### Specialized Functions

#### PrintErrorWithHint(err error)

Extracts a hint from the error (via `GetErrorHint`) and displays:

- Terminal: `"✗ Error: <err>"` on stderr; if hint present: `"  Try: <hint>"` (cyan `Try:`)
- Piped: `"error: <err>"` on stderr; if hint present: `"hint: <hint>"` on stderr

Always writes to stderr. Not gated by verbosity (errors are always shown).

#### PrintCommandHeader(text string)

```go
func PrintCommandHeader(text string) {
    if !IsQuiet() && !ShouldSimplifyOutput() {
        fmt.Println(Bold(text))
    }
    fmt.Println() // blank line always printed
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

Prints `"Next: Run 'lnk <command> <sourceDir>' to <description>"` via `PrintInfo`.

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
2. blank line
3. PrintDryRun("Would do X ...")
4. blank line
5. PrintDryRunSummary()
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

- [cli.md](cli.md) — Verbosity flag definitions (`--quiet`, `--verbose`, `--no-color`)
- [error-handling.md](error-handling.md) — `PrintErrorWithHint` and error display
