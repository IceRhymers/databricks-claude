<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-04-06 | Updated: 2026-04-06 -->

# registry

## Purpose
JSON-file-backed session registry for multi-process coordination. Tracks live `databricks-claude` proxy sessions so that when one exits, it can hand off `ANTHROPIC_BASE_URL` to a surviving session instead of restoring the original (which would break the survivor).

## Key Files

| File | Description |
|------|-------------|
| `registry.go` | `Session` struct, `SessionRegistry` with `Register`, `Unregister`, `LiveSessions` (with stale PID pruning), `MostRecentLive` |
| `registry_test.go` | Tests for register/unregister, stale PID pruning, most-recent-live selection |

## For AI Agents

### Working In This Directory
- Registry file is `~/.claude/.sessions.json` (path set by `process.go`).
- `LiveSessions` prunes dead PIDs using `syscall.Kill(pid, 0)` -- ESRCH means dead, EPERM means alive but different user.
- All reads/writes are mutex-guarded and use atomic temp-file + `os.Rename`.
- `MostRecentLive` sorts by `StartedAt` descending and returns the newest live session.
- The `ReadLocked` export exists for testing only.

### Testing Requirements
- Run: `go test ./pkg/registry/...`
- Tests create temp files and use real PIDs for liveness checks.

### Common Patterns
- Atomic writes: `os.CreateTemp` in same dir + `os.Chmod(0600)` + write + close + `os.Rename`.
- PID liveness: `syscall.Kill(pid, 0)` with EPERM tolerance.

## Dependencies

### Internal
- None

### External
- `syscall`, `encoding/json` (stdlib)

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
