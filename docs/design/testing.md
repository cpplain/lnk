# Testing Specification

---

## 1. Overview

### Purpose

This spec defines how tests are written for `lnk`. It supports a TDD workflow where
design specs are translated into tests before implementation code is written.

### Goals

- **TDD-first**: design specs drive test cases before implementation
- **Two test levels**: unit tests in `lnk/` and e2e tests in `test/` with clear boundaries
- **Stdlib only**: pure `testing` package, no external test frameworks
- **Reuse over reinvention**: documented helpers prevent duplicate test utilities

### Non-Goals

- Fuzz testing or property-based testing
- Benchmark tests
- Test mocking frameworks or dependency injection

---

## 2. TDD Workflow

### From Spec to Tests

Each design spec contains the information needed to write tests before implementation.
The mapping is:

| Spec element                       | Test case                                       |
| ---------------------------------- | ----------------------------------------------- |
| Numbered behavior step             | Unit test case verifying that step's outcome    |
| Error-type mapping table row       | Unit test case triggering that error condition  |
| Go function signature              | `Test<Function>` with table-driven cases        |
| CLI usage example                  | E2e test running the binary and checking output |
| Output format (terminal and piped) | Output test asserting both formats              |

### Process

1. **Read the design spec** — identify Go function signatures, input/output contracts,
   error types, and output formats
2. **Write unit tests first** — for every Go function in the spec, write table-driven
   tests covering: happy path, each error condition from the error-mapping table, and
   boundary cases
3. **Write e2e tests second** — for every CLI usage example in the spec, write a test
   that runs the compiled binary and asserts exit code, stdout, and stderr
4. **Implement until tests pass**

### Example: Mapping `create.md` to Tests

`create.md` specifies three phases:

- **Phase 1 (Collect)**: walk source dir, apply `PatternMatcher` → test that ignored
  files are excluded, non-ignored files are collected
- **Phase 2 (Validate)**: call `ValidateSymlinkCreation` for each file, all-or-nothing
  → test that any validation failure aborts before filesystem changes
- **Phase 3 (Execute)**: call `CreateSymlink`, continue-on-failure → test that one
  failure does not prevent remaining symlinks from being created, failure count is
  correct, aggregate error is returned

Each phase becomes a group of test cases in `TestCreateLinks`.

---

## 3. Test Levels and Boundaries

### Unit Tests (`lnk/*_test.go`)

- Same package (`package lnk`) — access to exported and unexported functions
- Direct function calls — no binary compilation
- `t.TempDir()` for filesystem isolation — no shared state between tests
- **Decision rule**: if a design spec defines a Go function, that function gets a unit test

### E2E Tests (`test/*_test.go`)

- Separate package (`package test`) — compiled binary execution via `exec.Command`
- Tests CLI behavior from a user's perspective — flag parsing, exit codes, output content
- Fixture-based environment via `setupTestEnv(t)`
- **Decision rule**: if a design spec defines CLI usage or examples, those get e2e tests

### Boundary Table

| Spec                | Unit tests                                                           | E2E tests                                            |
| ------------------- | -------------------------------------------------------------------- | ---------------------------------------------------- |
| `create.md`         | `CreateLinks` phases, ignore filtering, idempotency                  | `lnk create .`, dry-run output, error exit codes     |
| `remove.md`         | `RemoveLinks`, symlink verification, empty dir cleanup               | `lnk remove .`, dry-run, already-removed cases       |
| `status.md`         | `Status` categorization of active/broken links                       | `lnk status .`, piped output format                  |
| `prune.md`          | `PruneLinks`, broken-only filtering                                  | `lnk prune .`, dry-run, nothing-to-prune cases       |
| `adopt.md`          | `Adopt` validation, move, symlink creation, rollback                 | `lnk adopt . ~/.bashrc`, error cases                 |
| `orphan.md`         | `Orphan` validation, symlink removal, file move, rollback            | `lnk orphan . ~/.bashrc`, error cases                |
| `config.md`         | `LoadConfig`, `LoadIgnoreFile`, pattern merging                      | `--ignore` flag integration                          |
| `error-handling.md` | Error constructors, `Error()` format, `GetErrorHint`, `errors.As/Is` | Error display format, hint presence, exit codes      |
| `output.md`         | Print functions, color toggle, verbosity gating                      | Piped output prefixes, stderr separation             |
| `internals.md`      | `FindManagedLinks`, `CreateSymlink`, `MoveFile`, `CleanEmptyDirs`    | Covered indirectly through command e2e tests         |
| `cli.md`            | `suggestCommand`, `levenshteinDistance`                              | Flag parsing, help output, version, unknown commands |

---

## 4. Conventions

### Test Naming

- Test functions: `Test<FunctionName>` (e.g., `TestCreateLinks`, `TestPathError`)
- Subtests: lowercase descriptive phrases via `t.Run` (e.g., `"single source directory"`,
  `"path error with hint"`)

### Table-Driven Tests

All tests with two or more cases use the table-driven pattern:

```go
tests := []struct {
    name        string
    setup       func(t *testing.T, tmpDir string)
    wantErr     bool
    checkResult func(t *testing.T, tmpDir string)
}{
    {
        name: "descriptive case name",
        setup: func(t *testing.T, tmpDir string) {
            // arrange
        },
        checkResult: func(t *testing.T, tmpDir string) {
            // assert
        },
    },
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        tmpDir := t.TempDir()
        tt.setup(t, tmpDir)
        // act
        tt.checkResult(t, tmpDir)
    })
}
```

- `name string` is always the first field
- Range variable is `tt`
- Each case runs in a subtest via `t.Run(tt.name, ...)`

### Assertions

- All custom assertion functions call `t.Helper()` as the first line
- `t.Errorf` for non-fatal assertions (test continues to check remaining conditions)
- `t.Fatalf` only for setup failures that make the rest of the test meaningless
- Never use `t.Fatal` inside a goroutine

---

## 5. Test Helpers

Reuse these helpers. Do not create duplicates.

### Unit Test Helpers (`lnk/testutil_test.go`)

| Helper              | Signature                     | Use when                                      |
| ------------------- | ----------------------------- | --------------------------------------------- |
| `CaptureOutput`     | `(t, fn) string`              | Testing stdout content from print functions   |
| `ContainsOutput`    | `(t, output, expected...)`    | Asserting output includes specific strings    |
| `NotContainsOutput` | `(t, output, notExpected...)` | Asserting output excludes specific strings    |
| `createTestFile`    | `(t, path, content)`          | Creating source files with parent directories |
| `assertSymlink`     | `(t, link, expectedTarget)`   | Verifying symlink exists and points correctly |
| `assertNotExists`   | `(t, path)`                   | Verifying file or directory was removed       |
| `assertDirExists`   | `(t, path)`                   | Verifying directory was created               |

### E2E Test Helpers (`test/helpers_test.go`)

| Helper              | Signature                       | Use when                                  |
| ------------------- | ------------------------------- | ----------------------------------------- |
| `buildBinary`       | `(t) string`                    | Called automatically by `runCommand`      |
| `runCommand`        | `(t, args...) commandResult`    | Running any `lnk` CLI invocation          |
| `setupTestEnv`      | `(t) func()`                    | Setting up fixture-based test environment |
| `assertContains`    | `(t, output, expected...)`      | Checking CLI output strings               |
| `assertNotContains` | `(t, output, notExpected...)`   | Checking CLI output excludes strings      |
| `assertExitCode`    | `(t, result, expected)`         | Verifying exit code                       |
| `assertSymlink`     | `(t, linkPath, expectedTarget)` | Verifying symlink after CLI operation     |
| `assertNoSymlink`   | `(t, path)`                     | Verifying symlink was removed             |

`commandResult` has three fields: `Stdout`, `Stderr`, `ExitCode`.

---

## 6. Filesystem Setup

### Unit Tests

- Always use `t.TempDir()` — never `os.MkdirTemp` with manual cleanup
- Create source and target directories inside the temp dir
- Use `createTestFile(t, path, content)` for source files
- Use `os.Symlink` directly when testing against pre-existing symlinks
- Each test case gets its own temp dir (no shared mutable state)

### E2E Tests

- Call `setupTestEnv(t)` which runs `scripts/setup-testdata.sh` once per test session
- Fixture structure: `test/testdata/dotfiles/home/` (source),
  `test/testdata/target/` (target, cleaned between tests)
- Always `defer cleanup()` — the cleanup function removes test-created content
  from the target directory while preserving source fixtures

---

## 7. Output Testing

### Unit Tests

- Use `CaptureOutput(t, fn)` to capture stdout from print functions
- Test both terminal and piped formats by toggling `ShouldSimplifyOutput` state
- Test color by toggling `SetNoColor`
- Test verbosity gating by toggling `SetVerbosity`

### E2E Tests

- E2E tests always run in piped mode (binary stdout goes to `bytes.Buffer`, not a TTY)
- Assert piped-format prefixes: `"success "`, `"error: "`, `"warning: "`, `"skip "`,
  `"dry-run: "`
- Use `result.Stdout` for normal output, `result.Stderr` for errors and warnings
- Command headers are suppressed in piped mode — do not assert them in e2e tests

---

## 8. Error Testing

### Unit Tests

- Test `Error()` string format for each error type (`PathError`, `LinkError`,
  `ValidationError`, `HintedError`)
- Test `Unwrap()` chain with `errors.Is` and `errors.As`
- Test `GetErrorHint()` extraction from each error type
- Test constructor functions produce correct field values
- Test sentinel errors (`ErrNotSymlink`, `ErrAlreadyAdopted`) via `errors.Is`
- Each row in the error-type mapping tables ([error-handling.md](error-handling.md)
  Section 11) maps to a test case in the corresponding operation's test file

### E2E Tests

- Assert `result.ExitCode` matches expected code (0, 1, or 2)
- Assert `result.Stderr` contains the error message
- Assert `result.Stderr` contains `"hint: "` when a hint is expected
- Assert `result.Stdout` is empty on error (piped mode suppresses command headers)

---

## 9. Coverage

### Targets

| Component                                  | Target               |
| ------------------------------------------ | -------------------- |
| Overall                                    | 80%                  |
| Error types and constructors               | 100%                 |
| Core operations (`Create`, `Remove`, etc.) | 90%                  |
| Output functions                           | 80%                  |
| CLI parsing (`main.go`)                    | Covered by e2e tests |

### Running Coverage

```bash
make test-coverage    # generates coverage.html
```

Review uncovered lines before considering a feature complete.

---

## 10. Related Specifications

- [error-handling.md](error-handling.md) — Error type mapping tables that drive error test cases
- [output.md](output.md) — Output function contracts that drive output test cases
- [cli.md](cli.md) — CLI usage and exit codes that drive e2e test cases
- [internals.md](internals.md) — Internal function signatures that drive unit test cases
- [stdlib.md](stdlib.md) — Stdlib-only constraint (no external test frameworks)
