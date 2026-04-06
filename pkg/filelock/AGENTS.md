<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-04-06 | Updated: 2026-04-06 -->

# filelock

## Purpose
File-based exclusive locking using `syscall.Flock`. Provides mutual exclusion for concurrent access to `settings.json` across multiple `databricks-claude` processes. Works on Linux and macOS with graceful degradation on unsupported systems.

## Key Files

| File | Description |
|------|-------------|
| `filelock.go` | `FileLock` struct with `Lock` (exclusive `LOCK_EX`) and `Unlock` (`LOCK_UN` + close) methods |
| `filelock_test.go` | Tests for acquire/release and lock contention scenarios |

## For AI Agents

### Working In This Directory
- `Lock()` uses `syscall.Flock` with `LOCK_EX` (blocking exclusive). If flock is unsupported, it prints a warning to stderr and returns nil (graceful degradation).
- `Unlock()` is safe to call when `file` is nil (no-op).
- The lock file is created with mode `0600` for security.
- Lock file path is `~/.claude/.settings.lock` (set by `process.go`).

### Testing Requirements
- Run: `go test ./pkg/filelock/...`
- Tests may be platform-sensitive (flock behavior differs on some filesystems).

## Dependencies

### Internal
- None

### External
- `syscall` (stdlib)

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
