# databricks-claude

Transparent proxy wrapper for Claude Code that auto-refreshes Databricks OAuth tokens via the Databricks CLI.

## Architecture

Four source files plus tests, all in `package main`:

- **main.go** — CLI entry point: parses flags, resolves config from `~/.claude/settings.json` and `~/.claude/.databricks-claude.json` (persistent config), seeds the token cache, discovers the AI Gateway URL, starts the proxy, patches settings, launches `claude`, and restores settings on exit.
- **proxy.go** — HTTP reverse proxy with two routes: `/` forwards to the inference upstream (AI Gateway), `/otel/` forwards to the OTEL upstream. Both routes inject fresh OAuth tokens per-request.
- **token.go** — `TokenProvider` fetches and caches Databricks access tokens by shelling out to `databricks auth token`. Also contains `DiscoverHost`, `ResolveWorkspaceID`, and `ConstructGatewayURL` for auto-discovery.
- **process.go** — `SettingsManager` for atomic read/patch/restore of `~/.claude/settings.json`. `RunChild` launches `claude` as a subprocess. `ForwardSignals` relays SIGINT/SIGTERM to the child.

## Key Design Decisions

- **Zero external Go dependencies** — pure stdlib only. No vendor directory, no dependency management beyond `go.mod`.
- **settings.json is patched atomically** — writes go to a temp file in the same directory, then `os.Rename`. Always restored via `defer`, even on crash.
- **Token cache** — mutex-guarded `TokenProvider`. Tokens are cached and refreshed 5 minutes before expiry. On refresh failure, the last good token is returned.
- **Child process signals** — SIGINT/SIGTERM are forwarded to `claude`; its exit code is propagated as our exit code.
- **Two proxy routes** — `/` routes to inference upstream, `/otel/` routes to OTEL upstream. Path algebra prepends the upstream base path.
- **Panic recovery** — both proxy routes are wrapped in `recoveryHandler` that catches panics and returns HTTP 502.
- **Persistent config** — `~/.claude/.databricks-claude.json` stores the Databricks CLI profile across sessions, independent of the settings.json restore cycle. Written on first setup when profile is not DEFAULT.

## Testing

- Tests use **helper binaries compiled at test time** to mock the `databricks` CLI (see `buildHelperBinary`, `buildSlowBinary`, `buildAuthEnvBinary` in `token_test.go`).
- `warmToken()` in `proxy_test.go` pre-loads the token cache to avoid subprocess calls during proxy tests.
- `process_test.go` tests settings.json save/restore, atomic writes, OTEL handling, signal forwarding, and exit code propagation.
- Run: `make test` or `go test ./... -v`

## Build

- `make build` — produces `./databricks-claude`
- `make install` — installs to `$GOPATH/bin`
- `make dist` — cross-compiles for darwin/linux/windows amd64+arm64
- `make lint` — runs `go vet`

## No Breaking Changes Policy

This binary wraps `claude` and patches `~/.claude/settings.json`. Any change to settings.json handling, key names, or restore logic needs careful thought — a botched restore leaves the user's Claude config broken.
