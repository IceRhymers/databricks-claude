<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-04-06 | Updated: 2026-04-06 -->

# settings

## Purpose
Generic settings.json read/write/restore engine. Manages the lifecycle of `~/.claude/settings.json` env keys: saves originals before patching, atomically writes changes, and restores originals on exit with session-aware handoff to surviving processes.

## Key Files

| File | Description |
|------|-------------|
| `settings.go` | `Manager` struct, `Locker`/`SessionTracker`/`LiveSession` interfaces, `KeyConfig` (managed vs protected keys), `ReadSettings`, `WriteSettings` (atomic), `GetEnvBlock`, `SaveOriginals`, `SaveOriginalsIfPresent`, `ClearStaleLocalhost`, `SaveAndOverwrite`, `Restore` (with smart handoff), `RestoreKeys` |

## For AI Agents

### Working In This Directory
- **This is the most safety-critical package.** A bug in Restore leaves `settings.json` pointing at a dead proxy, breaking the user's Claude setup.
- `Locker` and `SessionTracker` are interfaces -- the root package injects `filelock.FileLock` and `registry.SessionRegistry`.
- **Key categories**: `ManagedKeys` (always save/restore, e.g., `ANTHROPIC_BASE_URL`) and `ProtectedKeys` (skipped during Restore when `protectedPersist` is true, e.g., OTEL keys).
- `SaveOriginals` stores absent keys as `nil` sentinel so Restore can `delete` them.
- `ClearStaleLocalhost` removes stale `http://127.0.0.1` values from originals to prevent restoring a dead proxy URL from a previous crash.
- **Smart handoff in Restore**: when other live sessions exist, the first `ManagedKey` (`ANTHROPIC_BASE_URL`) is pointed at the most recent survivor's proxy URL. Only the last session standing restores original values.
- `WriteSettings` uses atomic temp-file + `os.Chmod(0600)` + `os.Rename`.

### Testing Requirements
- This package has **no dedicated test file**. It is tested through `process_test.go` and `concurrent_test.go` in the root package.
- When modifying this package, run `go test ./... -v` (full suite) to ensure root-level integration tests pass.

### Common Patterns
- `SaveAndOverwrite` accepts a callback `writeValues(env, proxyURL)` so the caller controls what env keys are written.
- Double-locking: file lock (cross-process) + mutex (in-process).

## Dependencies

### Internal
- None (depends on injected `Locker` and `SessionTracker` interfaces)

### External
- `encoding/json`, `os`, `sync` (stdlib)

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
