<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-04-06 | Updated: 2026-04-06 -->

# pkg

## Purpose
Reusable library packages extracted from the root `main` package. Each sub-package is self-contained, has its own tests, and exposes clean interfaces. The root `.go` files in the project are thin facades that wire these packages together.

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `authcheck/` | Pre-flight Databricks authentication verification and interactive login (see `authcheck/AGENTS.md`) |
| `childproc/` | Child process lifecycle: start, signal forwarding, exit code propagation (see `childproc/AGENTS.md`) |
| `filelock/` | File-based exclusive locking via `syscall.Flock` (see `filelock/AGENTS.md`) |
| `proxy/` | HTTP/WebSocket reverse proxy, security checks, log sanitization (see `proxy/AGENTS.md`) |
| `registry/` | JSON-file-backed session registry for multi-process coordination (see `registry/AGENTS.md`) |
| `settings/` | Generic settings.json read/write/restore engine with session-aware handoff (see `settings/AGENTS.md`) |
| `tokencache/` | Generic mutex-guarded token cache with configurable fetcher (see `tokencache/AGENTS.md`) |

## For AI Agents

### Working In This Directory
- Each package must remain **zero external dependencies** -- Go stdlib only.
- Packages expose interfaces (`TokenFetcher`, `TokenSource`, `Locker`, `SessionTracker`) so the root `main` package can plug in concrete implementations.
- Do not add cross-dependencies between sibling packages. Each package should be independently importable.

### Testing Requirements
- Each package has co-located `*_test.go` files. Run `go test ./pkg/...` to test all packages.
- `proxy/` has the most extensive tests; `settings/` has no separate test file (tested via `process_test.go` in root).

### Common Patterns
- Packages that do I/O use **atomic temp-file + rename** for safe writes.
- Test-overridable globals (e.g., `execCommand` in `authcheck`) enable unit testing without real CLI calls.

## Dependencies

### Internal
- No cross-package dependencies within `pkg/`.

### External
- None (pure Go stdlib)

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
