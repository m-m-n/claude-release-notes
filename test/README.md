# Test Instructions for AI Agents

This document provides guidelines for AI agents when writing and executing tests.

## Test Framework

Go standard `testing` package (no external test framework).

## Test Execution

### Unit Tests
```bash
go test ./...
```

### Integration Tests (if applicable)

None. External integrations (GitHub API, claude-batch, SMTP) are abstracted
behind interfaces and covered by unit tests with mocks. No real network or
process calls in tests.

### E2E Tests (if applicable)

None.

## Test File Organization

Test files live next to the code they test (`foo.go` → `foo_test.go`), in the
same package.

## Writing Tests

### Test Naming Conventions

- `TestXxx` functions named after the function/behavior under test
  (e.g. `TestCompareVersions`, `TestBuildEmailBody`).
- Table-driven subtests named with `t.Run("case description", ...)`.

### Test Structure

Prefer table-driven tests. Mock external dependencies via the interfaces
defined in the codebase (GitHub client, translator, mail sender); do not
perform real HTTP, SMTP, or subprocess calls.

## Adding New Tests

Add cases to the existing table when extending behavior; create a new
`_test.go` file alongside new source files.

## E2E Test Guidelines (if applicable)

N/A.

## Common Patterns

- Use `t.TempDir()` for state/config file tests instead of touching real XDG
  directories.
- Inject paths and dependencies via constructor arguments so tests never read
  `$HOME`.
