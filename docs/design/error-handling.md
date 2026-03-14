# Error Handling Specification

---

## 1. Overview

### Purpose

`lnk` uses a structured error system with typed errors, optional actionable hints,
and consistent exit codes. Every user-visible error should be informative enough for
the user to resolve the issue without consulting documentation.

### Goals

- **Typed errors**: callers can distinguish error categories via `errors.As`
- **Actionable hints**: every error that has a likely fix provides a `Try:` suggestion
- **Consistent display**: all errors are displayed through `PrintErrorWithHint`
- **Standard exit codes**: follow POSIX conventions

### Non-Goals

- Structured (JSON) error output
- Error codes for programmatic error discrimination
- Stack traces

---

## 2. Error Types

### PathError

Represents a failure involving a specific filesystem path.

```go
type PathError struct {
    Op   string // operation being performed (e.g., "create directory")
    Path string // path that caused the error
    Err  error  // underlying OS or library error
    Hint string // optional actionable hint
}

func (e *PathError) Error() string {
    return fmt.Sprintf("%s %s: %v", e.Op, e.Path, e.Err)
}
```

Use for: file not found, permission denied, path expansion failures, symlink removal
failures.

### LinkError

Represents a failure involving a symlink operation with both source and target.

```go
type LinkError struct {
    Op     string // operation being performed (e.g., "create symlink")
    Source string // source file path
    Target string // target (symlink) path; may be empty
    Err    error  // underlying error
    Hint   string // optional actionable hint
}

func (e *LinkError) Error() string {
    if e.Target == "" {
        return fmt.Sprintf("%s %s: %v", e.Op, e.Source, e.Err)
    }
    return fmt.Sprintf("%s %s -> %s: %v", e.Op, e.Source, e.Target, e.Err)
}
```

Use for: symlink creation failures, collision with existing files, already-adopted
detection.

### ValidationError

Represents an invalid configuration or argument.

```go
type ValidationError struct {
    Field   string // field or parameter that failed validation
    Value   string // the invalid value (may be empty)
    Message string // description of the validation failure
    Hint    string // optional actionable hint
}

func (e *ValidationError) Error() string {
    if e.Value != "" {
        return fmt.Sprintf("invalid %s '%s': %s", e.Field, e.Value, e.Message)
    }
    return fmt.Sprintf("invalid %s: %s", e.Field, e.Message)
}
```

Use for: source directory does not exist, source/target overlap, circular symlinks,
path-is-not-a-directory.

### HintedError

Wraps any arbitrary error with a hint. Used when the underlying error is not one of
the typed errors above.

```go
type HintedError struct {
    Err  error
    Hint string
}

func (e *HintedError) Error() string { return e.Err.Error() }
func (e *HintedError) Unwrap() error  { return e.Err }
```

Use `WithHint(err, hint)` to create one.

---

## 3. The HintableError Interface

All four error types implement `HintableError`:

```go
type HintableError interface {
    error
    GetHint() string
}
```

`GetErrorHint(err error) string` uses `errors.As` to find a `HintableError` anywhere
in the error chain and returns its hint. Returns `""` if none found.

---

## 4. Sentinel Errors

```go
var (
    ErrNotSymlink    = errors.New("not a symlink")
    ErrAlreadyAdopted = errors.New("file already adopted")
)
```

These are used as the `Err` field inside `PathError` or `LinkError` so callers can
use `errors.Is` for type-safe checks.

---

## 5. LinkExistsError

A non-fatal signal that a symlink already exists with the correct target. Returned
by `CreateSymlink` when no action is needed.

```go
type LinkExistsError struct {
    target string
}

func (e LinkExistsError) Error() string {
    return fmt.Sprintf("symlink already exists: %s", e.target)
}
```

The caller checks for `LinkExistsError` explicitly and silently skips the link
without printing anything. It is **not** wrapped with a hint because it is not an
error condition.

---

## 6. Constructor Functions

| Function                                                  | Returns                           |
| --------------------------------------------------------- | --------------------------------- |
| `NewPathError(op, path, err)`                             | `*PathError` (no hint)            |
| `NewPathErrorWithHint(op, path, err, hint)`               | `*PathError` with hint            |
| `NewLinkErrorWithHint(op, source, target, err, hint)`     | `*LinkError` with hint            |
| `NewValidationErrorWithHint(field, value, message, hint)` | `*ValidationError` with hint      |
| `WithHint(err, hint)`                                     | `*HintedError` wrapping any error |

---

## 7. Error Display

All user-visible errors are displayed through `PrintErrorWithHint(err error)`.

#### Terminal Output

```
✗ Error: source directory does not exist: /nonexistent
  Try: Ensure the source directory exists or specify a different path
```

- Error line: `"✗ Error: <err.Error()>"`
- Hint line (if present): `"  Try: <hint>"` (indented two spaces, cyan `Try:` label)

#### Piped Output

```
error: source directory does not exist: /nonexistent
hint: Ensure the source directory exists or specify a different path
```

---

## 8. Exit Codes

| Code | Constant    | Meaning                                                             |
| ---- | ----------- | ------------------------------------------------------------------- |
| 0    | —           | Success                                                             |
| 1    | `ExitError` | Runtime error (operation encountered an error)                      |
| 2    | `ExitUsage` | Usage error (bad flags, unknown command, missing required argument) |

### When to Use Each Code

- **Exit 0**: command completed successfully (including "nothing to do" cases)
- **Exit 1**: operation was attempted but failed (e.g., permission denied, symlink creation failed)
- **Exit 2**: command was invoked incorrectly (e.g., unknown flag, `--quiet` + `--verbose`, missing required file argument, unknown command)

---

## 9. Error Propagation

- Library functions (`CreateLinks`, `RemoveLinks`, etc.) return errors to `main`
- `main` calls `PrintErrorWithHint(err)` then `os.Exit(ExitError)` for runtime errors
- Usage errors in `main` call `PrintErrorWithHint(err)` then `os.Exit(ExitUsage)`

Two patterns are used depending on the operation:

**Continue on failure** (`create`, `remove`, `prune`): per-item failures are printed
inline and counted; processing continues for remaining items; an aggregate error is
returned after all items are processed.

**Transactional** (`adopt`, `orphan`): all validations must pass before any filesystem
changes are made; if any execution step fails, all completed operations are rolled back
in reverse order and the error is returned — no partial state is left on disk.

---

## 10. Hint Guidelines

Good hints:

- Start with an imperative verb: `"Ensure..."`, `"Check..."`, `"Use..."`, `"Run..."`
- Are specific and actionable: reference the exact command or flag to use
- Do not repeat the error message

Examples:

```
"Check that the file path is correct and the file exists"
"Use 'lnk adopt <source-dir> <path>' to adopt this file first"
"Run 'lnk status' to see managed files"
"Ensure source and target paths are different"
```

---

## 11. Error Type Mapping by Operation

### Adopt (Phase 1 validation)

| Scenario                                      | Error Type        | Constructor                                           |
| --------------------------------------------- | ----------------- | ----------------------------------------------------- |
| Path does not exist                           | `PathError`       | `NewPathErrorWithHint(op, path, err, hint)`           |
| Already adopted (is a symlink into sourceDir) | `LinkError`       | `NewLinkErrorWithHint` with `ErrAlreadyAdopted`       |
| Path outside home directory                   | `ValidationError` | `NewValidationErrorWithHint(field, value, msg, hint)` |
| Destination already exists                    | `PathError`       | `NewPathErrorWithHint(op, destPath, err, hint)`       |
| Permission denied                             | `PathError`       | `NewPathErrorWithHint(op, path, err, hint)`           |

### Orphan (Phase 1 validation)

| Scenario                      | Error Type    | Constructor                                           |
| ----------------------------- | ------------- | ----------------------------------------------------- |
| Path does not exist           | `PathError`   | `NewPathErrorWithHint(op, path, err, hint)`           |
| Path is not a symlink         | `PathError`   | `NewPathErrorWithHint` with `ErrNotSymlink`           |
| Symlink not managed by source | `LinkError`   | `NewLinkErrorWithHint(op, source, target, err, hint)` |
| Broken symlink                | `PathError`   | `NewPathErrorWithHint(op, path, err, hint)`           |
| No managed links in directory | `HintedError` | `WithHint(err, hint)`                                 |

---

## 12. Related Specifications

- [cli.md](cli.md) — Where exit codes are applied
- [output.md](output.md) — `PrintErrorWithHint` implementation details
- [internals.md](internals.md) — Internal helper functions referenced by operation specs
