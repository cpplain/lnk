---
status: test
---

# Task: fix CLI edge cases — bare lnk exit code and extractCommand --ignore parsing

## Behavior

Two fixes in `main.go` to align with `docs/design/cli.md`:

1. **Bare `lnk` exit code** (cli.md §3 step 4): bare `lnk` (no arguments)
   should print usage and exit 0. Currently exits 2.

2. **`extractCommand` --ignore handling** (cli.md §2, §4): when `--ignore`
   appears before the command name, `extractCommand` must skip its value
   argument so it is not mistaken for the command name. Two sub-issues:
   - `--ignore value` form: the current `i++` inside a `for i, arg := range`
     loop is a no-op in Go — the range iterator ignores mutations to `i`.
     The value token is therefore treated as the command name.
   - `--ignore=value` form: currently skipped by accident (starts with `-`),
     but the value-skip logic should explicitly recognize this form so
     behavior is intentional rather than coincidental.

## Acceptance Criteria

- Bare `lnk` (no args) prints usage and exits 0.
- `lnk --ignore pattern create .` correctly identifies `create` as the
  command (not `pattern`).
- `lnk --ignore=pattern create .` correctly identifies `create` as the
  command.
- `lnk --ignore pattern` with no command still returns empty command.
- All existing tests pass (`make check`).

## Context

- `main.go` lines 44–48: bare `lnk` handler — change `ExitUsage` to `0`.
- `main.go` lines 308–332: `extractCommand` — convert `for range` to
  index-based `for` loop so `i++` works, and add `--ignore=` prefix check.
- `main_test.go` contains `extractCommand` tests — will need new cases.
- No changes needed in `lnk/` package.

## Log

### Planning

- Both fixes are small, localized changes to `main.go`.
- `extractCommand` fix: convert to C-style for loop (`for i := 0; i < len(args); i++`)
  so the `i++` skip actually advances past the `--ignore` value. Add an
  explicit check for `strings.HasPrefix(arg, "--ignore=")` to document the
  handling of the `=value` form.
- Bare `lnk` fix: change `os.Exit(lnk.ExitUsage)` to a bare `return` (exit 0).
- Status set to `test` because both fixes have clear testable behavior:
  exit codes and command extraction logic.

### Testing

- ...

### Implementation

- ...
