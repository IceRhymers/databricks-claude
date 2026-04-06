<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-04-06 | Updated: 2026-04-06 -->

# authcheck

## Purpose
Pre-flight Databricks authentication verification. Checks whether the user has a valid token for a given CLI profile and, if not, triggers an interactive browser-based OAuth login via `databricks auth login`.

## Key Files

| File | Description |
|------|-------------|
| `authcheck.go` | `IsAuthenticated` (non-interactive token check) and `EnsureAuthenticated` (check + interactive login fallback) |
| `authcheck_test.go` | Tests using overridable `execCommand`/`execCommandContext` globals to mock CLI calls |

## For AI Agents

### Working In This Directory
- `execCommand` and `execCommandContext` are package-level vars overridable in tests -- do not refactor these into a struct without updating all test mocks.
- `EnsureAuthenticated` attaches stdin/stdout/stderr to the child process for the interactive browser OAuth flow. It must remain interactive.
- Called early in `main.go` before any proxy setup.

### Testing Requirements
- Tests override `execCommand`/`execCommandContext` with mock functions. No real `databricks` CLI is needed.
- Run: `go test ./pkg/authcheck/...`

### Common Patterns
- 5-second timeout on non-interactive token check to avoid hanging on network issues.

## Dependencies

### Internal
- None

### External
- `os/exec`, `context` (stdlib)

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
