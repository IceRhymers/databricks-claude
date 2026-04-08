# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

Transparent proxy wrapper for Claude Code that auto-refreshes Databricks OAuth tokens via the Databricks CLI. Zero external Go dependencies — pure stdlib only.

## Build & Test

```bash
make build                       # produces ./databricks-claude
make test                        # go test ./... -v
make lint                        # go vet ./...
make install                     # installs to $GOPATH/bin
make dist                        # cross-compile darwin/linux/windows amd64+arm64
go test -run TestParseArgs -v    # run a single test
go test ./pkg/proxy/... -v       # test a single package
```

## Architecture

### Root package (`main`)

Six `.go` files act as thin facades wiring together `pkg/` sub-packages:

- **main.go** — CLI entry point: flag parsing, config resolution (`~/.claude/settings.json` + `~/.claude/.databricks-claude.json`), token seeding, AI Gateway auto-discovery, proxy startup, settings patching, child launch, headless lifecycle management (`wrapWithLifecycle` for `/shutdown` endpoint and idle timeout), and explicit settings restore before `os.Exit`.
- **proxy.go** — Facade over `pkg/proxy`: defines `ProxyConfig`, wires `NewProxyServer` and `StartProxy`.
- **token.go** — Facade over `pkg/tokencache`: implements `databricksFetcher` (shells out to `databricks auth token`), host discovery, workspace ID resolution via SCIM, AI Gateway URL construction.
- **process.go** — `SettingsManager` wrapping `pkg/settings`: full-setup and save/restore of settings.json env keys, OTEL key management, `RunChild`, `ForwardSignals`.
- **lock.go** — Type alias forwarding `pkg/filelock.FileLock`.
- **registry.go** — Type alias forwarding `pkg/registry.SessionRegistry`.

### Library packages (`pkg/`)

Each package is independently importable with no cross-dependencies:

| Package | Purpose |
|---------|---------|
| `pkg/authcheck` | Pre-flight Databricks auth check + interactive browser login fallback |
| `pkg/childproc` | Child process start, SIGINT/SIGTERM forwarding, exit code propagation |
| `pkg/filelock` | Exclusive file locking via `syscall.Flock` (graceful degradation) |
| `pkg/proxy` | HTTP/WebSocket reverse proxy, OTEL routing, API key auth, TLS, panic recovery, log sanitization |
| `pkg/registry` | JSON-file session registry for multi-process coordination and handoff |
| `pkg/settings` | Settings.json read/write/restore engine with session-aware handoff |
| `pkg/tokencache` | Mutex-guarded token cache with 5-min refresh buffer and fallback-on-error |

### Key data flow

1. `main.go` seeds token cache → starts proxy on `127.0.0.1:0` → patches `settings.json` to point `ANTHROPIC_BASE_URL` at proxy → launches `claude` as child (or enters headless mode)
2. In headless mode, the handler is wrapped with `wrapWithLifecycle` which adds `POST /shutdown` (refcount decrement + conditional exit) and idle timeout (default 30m, resets on each proxied request)
3. Proxy intercepts every request, injects fresh OAuth token via `pkg/tokencache` → forwards to AI Gateway (inference) or Databricks OTEL endpoint
4. On exit, `SettingsManager.Restore()` is called **explicitly before `os.Exit`** (not via defer — `os.Exit` skips defers). Smart handoff points `ANTHROPIC_BASE_URL` to a surviving session if one exists.

## Key Design Constraints

- **Zero external dependencies** — do not add third-party imports. Pure Go stdlib only.
- **No breaking changes to settings.json** — a botched restore leaves the user's Claude config pointing at a dead proxy. Any change to key names, restore logic, or the save/restore lifecycle requires extreme care.
- **Atomic file writes everywhere** — all JSON writes (settings, registry, persistent config) use temp-file + `os.Rename` in the same directory.
- **Session handoff** — when multiple `databricks-claude` instances run concurrently, the exiting one hands `ANTHROPIC_BASE_URL` to the most recent survivor via `pkg/registry`.
- **OTEL key persistence** — when `--otel` is active, OTEL keys survive settings restore (controlled by `otelKeysPersistent` flag).

## Testing Patterns

- **Helper binaries**: `token_test.go` compiles small Go programs at test time (`buildHelperBinary`, `buildSlowBinary`, `buildAuthEnvBinary`) to mock the `databricks` CLI. Don't try to mock `exec.Command` directly.
- **Token pre-warming**: `warmToken()` in `proxy_test.go` seeds the cache to avoid subprocess calls during proxy tests.
- **Test-overridable globals**: `authcheck` uses `var execCommand = exec.Command` pattern; `checks.go` uses `var getuid = os.Getuid`. Override in tests, not via interfaces.
- **Settings isolation**: `process_test.go` creates temp directories with fresh `settings.json` files. `pkg/settings` has no dedicated test file — it's tested through root-level integration tests.

## Pre-Commit Consistency Checks

Before generating any commit, verify that documentation files are consistent with the code:

1. **README.md** — flags table, usage examples, and `--print-env` output must match `parseArgs` in `main.go` and `handlePrintEnv` output format.
2. **CLAUDE.md** — architecture description must reflect the current file and package structure.
3. **AGENTS.md files** — key file tables must list files that actually exist; parent references must resolve.

If any documentation is stale, fix it in the same commit or flag it.
