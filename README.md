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

### Via Scoop (Windows)

```powershell
scoop bucket add icerhymers https://github.com/IceRhymers/scoop-bucket
scoop install databricks-claude
```

### Direct binary (Windows)

Download the latest release from the [releases page](https://github.com/IceRhymers/databricks-claude/releases), pick `databricks-claude-windows-amd64.exe` (or `arm64`), rename it to `databricks-claude.exe`, and place it somewhere on your `PATH`.

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
- A Databricks Model Serving endpoint with [AI Gateway](https://docs.databricks.com/aws/en/ai-gateway/) enabled (currently in public Beta)

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
| `--credential-helper` | | Print a fresh Databricks token to stdout (used as Claude Desktop's `inferenceCredentialHelper`). Also dispatched automatically when invoked under the `databricks-claude-credential-helper` symlink. |
| `--generate-desktop-config` | | Write Claude Desktop MDM configs. Without `--output`, writes both `databricks-claude-desktop.mobileconfig` and `databricks-claude-desktop.reg` so one invocation covers macOS and Windows. |
| `--output` | | Single output path for `--generate-desktop-config`; format inferred from `.mobileconfig` / `.reg` extension or host OS. |
| `--binary-path` | derived from running binary | Credential-helper path baked into the generated config. Set this for MDM rollouts so one config works on every endpoint. |
| `--databricks-cli-path` | | Pin the absolute path of the `databricks` CLI used by the credential helper subprocess. Persisted to `~/.claude/.databricks-claude.json`. Useful when the CLI is installed somewhere the launchd-PATH fallback dir scan can't see. |

All other flags and args are forwarded to `claude`.

## Auto-Discovery

On first run (when `ANTHROPIC_BASE_URL` is not set), `databricks-claude` auto-discovers:

- Your workspace host from `databricks auth env`
- Your workspace ID via the SCIM API (`x-databricks-org-id` header)
- Constructs the AI Gateway URL: `https://<workspace-id>.ai-gateway.cloud.databricks.com/anthropic`

If workspace ID resolution fails, it falls back to `<host>/serving-endpoints/anthropic`.

## Claude Desktop Integration

`databricks-claude` can act as the credential helper for the Claude Desktop app's third-party-inference mode. Desktop calls a single executable (no args allowed) once per token TTL and uses whatever it prints to stdout as the bearer token for AI Gateway requests.

### One-time setup

1. **Install** `databricks-claude` (Homebrew, `make install`, or `go install`). All install methods drop a `databricks-claude-credential-helper` symlink next to the main binary; that symlink is the path Claude Desktop will invoke.
2. **Authenticate** with the workspace you want Desktop to talk to: `databricks auth login --profile <name>`.
3. **Generate the desktop config:**
   ```bash
   databricks-claude --profile <name> --generate-desktop-config
   ```
   This writes both `databricks-claude-desktop.mobileconfig` (macOS) and `databricks-claude-desktop.reg` (Windows) into the current directory.
4. **Install the config:**
   - **macOS**: `open databricks-claude-desktop.mobileconfig`, then approve in System Settings → Privacy & Security → Profiles.
   - **Windows**: double-click the `.reg` file, or `reg import databricks-claude-desktop.reg`.
5. **Restart Claude Desktop.**

After this, Desktop's third-party-inference path runs against your Databricks AI Gateway, with tokens refreshed automatically by the credential helper.

### How dispatch works

The `inferenceCredentialHelper` MDM key in the generated config points at `…/databricks-claude-credential-helper` (the symlink). When invoked under that name, the binary checks `argv[0]` and routes directly to the credential-helper code path — no flags required. The same binary still runs as a Claude Code wrapper when invoked under its primary name.

### MDM / fleet rollout

For rolling this out to a fleet via Jamf, Kandji, Intune, etc., generate the config from a reference workstation with paths that match your endpoint layout:

```bash
# Bake fleet-wide paths into the generated config
databricks-claude \
  --profile <name> \
  --generate-desktop-config \
  --binary-path /usr/local/bin/databricks-claude-credential-helper \
  --databricks-cli-path /usr/local/bin/databricks
```

- `--binary-path` is the absolute path of the credential-helper symlink (or hardlink/copy) on every target endpoint.
- `--databricks-cli-path` pins the `databricks` CLI absolute path. It's persisted to `~/.claude/.databricks-claude.json` on the generating machine; admins should arrange for the same field to be set on every endpoint (either by running this command per-user, or by dropping the state file via the same MDM tooling).

The packaging method (`.pkg` installer, custom `brew` formula, etc.) is responsible for ensuring `databricks-claude` and its `databricks-claude-credential-helper` symlink land at the paths you embed in the config.

### Troubleshooting

The helper logs every invocation (best-effort, silent on failure) to:
- macOS: `~/Library/Logs/databricks-claude/credential-helper.log`
- Linux: `~/.cache/databricks-claude/credential-helper.log`

Each entry records the resolved profile, CLI path, and either the token length on success or the underlying error. If Desktop reports `invalid_config` or 401, check this log first.

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

## Session Hooks (automatic proxy lifecycle)

Install hooks so every Claude Code session auto-starts the proxy on startup and releases it cleanly on exit — no manual `--headless` needed.

> **First-time setup:** Run `databricks-claude` at least once before installing hooks. This writes the correct `ANTHROPIC_BASE_URL` to `~/.claude/settings.json` so the proxy is used for all Claude clients. Once set, the hooks keep the proxy running automatically — including for clients that don't use the `databricks-claude` wrapper directly, such as the [Claude VS Code extension](https://marketplace.visualstudio.com/items?itemName=Anthropic.claude-code) and JetBrains/IntelliJ plugin.

### Install

```bash
databricks-claude --install-hooks
```

This merges two hooks into `~/.claude/settings.json`:

- **SessionStart** — calls `databricks-claude --headless-ensure` on session startup: starts the proxy if it isn't already running
- **Stop** — calls `databricks-claude --headless-release` on session end: decrements the refcount; proxy exits when the last session closes

### Uninstall

```bash
databricks-claude --uninstall-hooks
```

Removes only the databricks-claude hook entries. Other hooks in your settings are untouched.

### Notes

- Idempotent — safe to re-run after upgrades
- The proxy starts on the configured port (default `49153`). If you use a custom port via `--port`, the hooks will respect that setting automatically (port is saved to the state file)
- Unclean exits (force-quit, OOM kill) are covered by the idle timeout — the proxy self-exits after 30 minutes with no inference traffic

### Claude Code Plugin (marketplace install)

Hooks are also distributed as a Claude Code plugin. Add this repo as a marketplace, then install the plugin:

```
/plugin marketplace add IceRhymers/databricks-claude
/plugin install databricks-claude@IceRhymers-databricks-claude
```

The `.claude-plugin/` directory and `hooks/hooks.json` at the repo root define the plugin.

## Shell Tab Completions

`databricks-claude` can generate shell completion scripts for bash, zsh, and fish. Completions are derived from the binary's own flag metadata, so they stay in sync automatically.

### Install (one-time)

**bash** — add to `~/.bashrc`:
```bash
eval "$(databricks-claude completion bash)"
```

**zsh** — add to `~/.zshrc`:
```zsh
eval "$(databricks-claude completion zsh)"
```

**fish** — add to `~/.config/fish/config.fish`:
```fish
databricks-claude completion fish | source
```

### Homebrew

If installed via `brew install IceRhymers/tap/databricks-claude`, completions are installed automatically — no extra setup needed.

### What completes

- `--profile <TAB>` — lists profiles from `~/.databrickscfg` (updated live, no rehash needed)
- `--log-file`, `--tls-cert`, `--tls-key`, `--upstream <TAB>` — file path completion
- All other flags — name completion when you type `-`

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
  ANTHROPIC_MODEL: databricks-claude-opus-4-7
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

## Shell Tab Completions

`databricks-claude` includes a completion engine (`pkg/completion`) that generates shell scripts from the binary's own flag definitions. If you installed via Homebrew, completions are registered automatically — no manual setup required.

### Manual Installation

If you installed from source or want to set completions up yourself, source the output of the `completion` subcommand in your shell rc file:

```bash
# Bash (~/.bashrc)
eval "$(databricks-claude completion bash)"

# Zsh (~/.zshrc)
eval "$(databricks-claude completion zsh)"

# Fish (~/.config/fish/config.fish)
databricks-claude completion fish | source
```

### What Gets Completed

- **Flag names** — `--<Tab>` lists all flags (long and short forms).
- **Flag values** — context-aware completions for flags that accept a value:
  - `--profile` completes from `~/.databrickscfg` section headers.
  - `--upstream`, `--log-file`, `--tls-cert`, `--tls-key` complete with local file paths.
  - Flags like `--port` or `--otel-metrics-table` suppress file completion.
- **Passthrough boundary** — after a bare `--`, completions stop. Everything beyond that is forwarded to the wrapped `claude` binary.

### How the Engine Works

This section documents the `pkg/completion` package for other projects that import it.

The `completion` subcommand is the very first check in `main()`, before any config loading, auth, or state. This makes it safe to call in restricted environments like the Homebrew install sandbox.

```
main.go
  └─ if os.Args[1] == "completion"
       └─ completion.Run(args, flagDefs, binaryName)
            ├─ "bash"  → GenerateBash()
            ├─ "zsh"   → GenerateZsh()
            └─ "fish"  → GenerateFish()
```

**`FlagDef` struct** — each flag is described by a single struct in `completion_flags.go`:

| Field | Type | Purpose |
|-------|------|---------|
| `Name` | `string` | Flag name without `--` (e.g. `"profile"`) |
| `Short` | `string` | Single-char alias without `-` (e.g. `"v"`), or empty |
| `Description` | `string` | Human-readable description shown in completions |
| `TakesArg` | `bool` | `true` if the flag consumes the next token as its value |
| `Completer` | `string` | Named completer function, or empty for no value completion |

**Named completers** — two built-in completer names are supported:

- `"__databricks_profiles"` — reads `[section]` headers from `~/.databrickscfg`.
- `"__files"` — completes with local file paths (uses each shell's native mechanism).

Completers are emitted as shell functions embedded in the generated script — no external dependencies at completion time.

**Adding a new flag** — add an entry to the `flagDefs` slice. The completion script, `knownFlags` map, and flag parsing all derive from this single slice. Consistency tests enforce that every `FlagDef` appears in `knownFlags` and vice-versa.

**Integrating in another binary** — import `pkg/completion`, define your own `[]FlagDef`, and add the early-exit check to `main()`:

```go
import "github.com/IceRhymers/databricks-claude/pkg/completion"

var flagDefs = []completion.FlagDef{ /* ... */ }

func main() {
    if len(os.Args) >= 2 && os.Args[1] == "completion" {
        completion.Run(os.Args[2:], flagDefs, "my-binary")
        os.Exit(0)
    }
    // ... rest of main
}
```

## Automatic Update Check

`databricks-claude` checks for newer releases on startup (once every 24 hours) and prints a one-line notice to stderr when an update is available. The check is synchronous with a 2-second timeout — if GitHub is unreachable it silently skips.

### Update notification

When a newer version exists you'll see:

```
# Direct install
databricks-claude: update available (v0.11.0). Run: databricks-claude update

# Homebrew install
databricks-claude: update available (v0.11.0). Run: brew upgrade databricks-claude
```

### `update` subcommand

```bash
databricks-claude update
```

Force-checks GitHub for the latest release (bypasses the 24-hour cache) and prints upgrade instructions:

| Install method | Output |
|---|---|
| Already latest | `databricks-claude v0.10.1 is already the latest version` |
| Direct install | `Update available: v0.11.0. Download from: https://github.com/...` |
| Homebrew | `Update available: v0.11.0. Run: brew upgrade databricks-claude` |

No binary is replaced — the command prints instructions only. In-place self-update is planned for a future release.

### Opt out

```bash
# Per-invocation flag
databricks-claude --no-update-check

# Per-session or permanent (add to shell profile)
export DATABRICKS_NO_UPDATE_CHECK=1
```

Both suppress the startup check and disable the `update` subcommand.

## License

MIT