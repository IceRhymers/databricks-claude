# databricks-claude

> **Disclaimer:** This is an unofficial, community-built workaround to enable Databricks OAuth SSO authentication with this AI coding tool. It is not supported, endorsed, or recognized by Databricks. Use at your own risk.


Transparent proxy wrapper for Claude Code that auto-refreshes Databricks OAuth tokens — so you never manually paste a token again.

## The Problem

Databricks AI Gateway uses short-lived OAuth tokens. Claude Code only supports a static `ANTHROPIC_AUTH_TOKEN` in `~/.claude/settings.json`. Without this tool, you'd need to manually refresh and paste a new token every hour.

## How It Works

`databricks-claude` wraps the `claude` binary. It:

1. Binds a local HTTP proxy on `127.0.0.1:49153` (fixed port — shared across concurrent sessions)
2. Writes `~/.claude/settings.json` once to point `ANTHROPIC_BASE_URL` at the proxy (idempotent — no restore on exit)
3. Launches `claude` with your args — fully transparent
4. Injects fresh Databricks OAuth tokens on every request (auto-refreshed from `databricks auth token`)
5. Tracks concurrent sessions with a ref-count; the last session out closes the listener

You use it exactly like `claude`. Every flag and argument is forwarded.

## Installation

Via Homebrew (recommended):

```
brew tap IceRhymers/tap
brew install databricks-claude
```

Via Go:

### From source

```bash
go install github.com/IceRhymers/databricks-claude@latest
```

### Alias (optional but recommended)

```bash
echo 'alias claude="databricks-claude"' >> ~/.zshrc  # or ~/.bashrc
```

## Prerequisites

- Go 1.22+
- [Databricks CLI](https://docs.databricks.com/dev-tools/cli/databricks-cli.html) installed and authenticated (`databricks auth login`)
- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) installed

## Usage

```bash
# Use exactly like claude:
databricks-claude "explain this codebase"

# With a specific Databricks CLI profile:
databricks-claude --profile my-workspace "write tests for auth.py"

# Verbose logging (debug output to stderr):
databricks-claude --verbose "fix the bug in main.go"

# Log to file:
databricks-claude --log-file /tmp/dc.log "fix the bug in main.go"

# Both stderr and file:
databricks-claude -v --log-file /tmp/dc.log "fix the bug in main.go"

# With OTEL telemetry:
databricks-claude --otel "summarize this PR"

# With custom OTEL tables:
databricks-claude --otel --otel-metrics-table main.catalog.metrics --otel-logs-table main.catalog.logs "summarize this PR"

# Disable OTEL (clears persisted keys):
databricks-claude --no-otel

# With proxy API key authentication:
databricks-claude --proxy-api-key my-secret-key "explain this codebase"

# With TLS:
databricks-claude --tls-cert cert.pem --tls-key key.pem "explain this codebase"
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--profile` | `DEFAULT` | Databricks CLI profile |
| `--verbose`, `-v` | `false` | Enable debug logging to stderr |
| `--log-file` | | Write debug logs to a file (combinable with `--verbose`) |
| `--otel` | `false` | Enable OTEL telemetry proxying |
| `--no-otel` | | Clear persisted OTEL keys and disable OTEL for future sessions |
| `--otel-metrics-table` | `main.claude_telemetry.claude_otel_metrics` | Unity Catalog table for OTEL metrics |
| `--otel-logs-table` | derived from metrics table | Unity Catalog table for OTEL logs |
| `--upstream` | auto-discovered | Override the AI Gateway URL |
| `--proxy-api-key` | | Require Bearer token auth on all proxy requests |
| `--port` | `49153` | Proxy listen port (saved for future sessions) |
| `--tls-cert` | | Path to TLS certificate file (requires `--tls-key`) |
| `--tls-key` | | Path to TLS private key file (requires `--tls-cert`) |
| `--headless` | `false` | Start proxy without launching claude (for IDE extensions) |
| `--idle-timeout` | `30m` | Idle timeout in headless mode (`0` disables) |
| `--version` | | Print version and exit |
| `--print-env` | | Print resolved configuration (token redacted) and exit |
| `--help`, `-h` | | Print wrapper flags and the full `claude --help` output, then exit |

All other flags and args are forwarded to `claude`.

## Auto-Discovery

On first run (when `ANTHROPIC_BASE_URL` is not set), `databricks-claude` auto-discovers:

- Your workspace host from `databricks auth env`
- Your workspace ID via the SCIM API (`x-databricks-org-id` header)
- Constructs the AI Gateway URL: `https://<workspace-id>.ai-gateway.cloud.databricks.com/anthropic`

If workspace ID resolution fails, it falls back to `<host>/serving-endpoints/anthropic`.

## Headless Mode

`--headless` starts the proxy without launching a `claude` child process, for use by IDE extensions and external tooling.

```bash
databricks-claude --headless
# prints: PROXY_URL=http://127.0.0.1:<port>
```

### Lifecycle Management

- **`GET /health`** — liveness check, returns `{"tool":"databricks-claude","version":"...","pid":...}`
- **`POST /shutdown`** — decrements the session refcount; when it reaches 0, the proxy exits. Returns `{"remaining": N, "exiting": true/false}`
- **Idle timeout** — after 30 minutes with no proxied requests, the proxy shuts down automatically. Configure with `--idle-timeout <duration>` (e.g. `10m`, `1h`). Use `--idle-timeout 0` to disable.

## Profile Resolution Order

1. `--profile` CLI flag (writes to state file for future runs)
2. `profile` from `~/.claude/.databricks-claude.json` (state file)
3. `DEFAULT`

> **Note:** `DATABRICKS_CONFIG_PROFILE` is intentionally *not* consulted during
> resolution. Claude's `settings.json` injects env vars into child processes,
> which would override the user's explicit `--profile` choice persisted in the
> state file.

## Persistent Config (`~/.claude/.databricks-claude.json`)

On first setup (when `ANTHROPIC_BASE_URL` is not yet configured), `databricks-claude` saves your resolved profile to `~/.claude/.databricks-claude.json`. This file persists independently of `settings.json` — your profile is never lost when config is rewritten.

```json
{
  "profile": "my-workspace"
}
```

This means you only need to pass `--profile` once — subsequent runs will automatically use the saved profile. To switch profiles, pass `--profile <new-profile>` and the persistent config is updated.

The file is only written when the profile is not `DEFAULT` (the implicit default doesn't need saving).

## Debugging

### Verify your auth setup

Run `--print-env` to see the resolved configuration without starting the proxy. The token is redacted so it's safe to share output for debugging.

```bash
databricks-claude --print-env
```

Example output:

```
databricks-claude configuration:
  Profile:              DEFAULT
  DATABRICKS_HOST:      https://adb-1234567890123456.7.azuredatabricks.net
  ANTHROPIC_BASE_URL:   https://1234567890123456.ai-gateway.cloud.databricks.com/anthropic
  ANTHROPIC_AUTH_TOKEN: dapi-***
  ANTHROPIC_MODEL:
  Upstream binary:      /usr/local/bin/claude
  OTEL enabled:         false
```

If the token shows as empty or the base URL looks wrong, check your Databricks CLI profile with `databricks auth env`.

### View full usage

`databricks-claude --help` (or `-h`) prints the wrapper's own flags followed by the complete `claude --help` output, so you see everything in one place.

## Development

```bash
git clone https://github.com/IceRhymers/databricks-claude
cd databricks-claude
make test
make build
```

## License

MIT