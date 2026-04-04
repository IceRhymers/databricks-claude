# databricks-claude

Transparent proxy wrapper for Claude Code that auto-refreshes Databricks OAuth tokens — so you never manually paste a token again.

## The Problem

Databricks AI Gateway uses short-lived OAuth tokens. Claude Code only supports a static `ANTHROPIC_AUTH_TOKEN` in `~/.claude/settings.json`. Without this tool, you'd need to manually refresh and paste a new token every hour.

## How It Works

`databricks-claude` wraps the `claude` binary. It:

1. Spins up a local HTTP proxy on `127.0.0.1` (random port)
2. Patches `~/.claude/settings.json` to point at the proxy
3. Launches `claude` with your args — fully transparent
4. Injects fresh Databricks OAuth tokens on every request (auto-refreshed from `databricks auth token`)
5. Restores `settings.json` when done (even on crash, via `defer`)

You use it exactly like `claude`. Every flag and argument is forwarded.

## Installation

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
databricks-claude --otel --otel-table main.catalog.table "summarize this PR"
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--profile` | `DEFAULT` | Databricks CLI profile |
| `--verbose`, `-v` | `false` | Enable debug logging to stderr |
| `--log-file` | | Write debug logs to a file (combinable with `--verbose`) |
| `--otel` | `false` | Enable OTEL telemetry proxying |
| `--otel-table` | `main.claude_telemetry.claude_otel_metrics` | UC table for OTEL metrics |
| `--upstream` | auto-discovered | Override the AI Gateway URL |
| `--version` | | Print version and exit |
| `--print-env` | | Print resolved configuration (token redacted) and exit. Use to verify auth setup without starting the proxy. |
| `--help`, `-h` | | Print wrapper flags and the full `claude --help` output, then exit. |

All other flags and args are forwarded to `claude`.

## Auto-Discovery

On first run (when `ANTHROPIC_BASE_URL` is not set), `databricks-claude` auto-discovers:

- Your workspace host from `databricks auth env`
- Your workspace ID via the SCIM API (`x-databricks-org-id` header)
- Constructs the AI Gateway URL: `https://<workspace-id>.ai-gateway.cloud.databricks.com/anthropic`

If workspace ID resolution fails, it falls back to `<host>/serving-endpoints/anthropic`.

## Profile Resolution Order

1. `--profile` CLI flag
2. `DATABRICKS_CONFIG_PROFILE` environment variable
3. `DATABRICKS_CONFIG_PROFILE` in `~/.claude/settings.json` env block
4. `DEFAULT`

## Debugging

### Verify your auth setup

Run `--print-env` to see the resolved configuration without starting the proxy. The token is redacted so it's safe to share output for debugging.

```bash
databricks-claude --print-env
```

Example output:

```
profile:   DEFAULT
host:      https://adb-1234567890123456.7.azuredatabricks.net
upstream:  https://1234567890123456.ai-gateway.cloud.databricks.com/anthropic
token:     dapi******************************** (redacted)
```

If the token shows as empty or the upstream URL looks wrong, check your Databricks CLI profile with `databricks auth env`.

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
