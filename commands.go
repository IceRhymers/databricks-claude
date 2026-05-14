package main

import (
	"github.com/IceRhymers/databricks-claude/internal/cmd"
)

// rootCommand is the source-of-truth declaration for the databricks-claude
// CLI. It drives:
//   - parseArgs → knownFlags (the set of "--flag" names the binary owns;
//     anything else is forwarded transparently to the wrapped claude binary).
//   - handleHelp → the help body (rendered from rootCommand.Long).
//   - completion <shell> → the bash/zsh/fish completion scripts (fed via
//     pkg/completion using rootCommand.CompletionFlags() and
//     rootCommand.CompletionSubcommands()).
//
// Adding a new root flag requires three edits:
//  1. Append a FlagDef to Flags (or Persistent for inherited flags) here.
//  2. Add a case to the switch in parseArgs (main.go) that wires the flag
//     into the Args struct.
//  3. Add the matching field to the Args struct.
//
// The parity tests in main_test.go (TestRootTreeFlagsAreParseRecognised,
// TestParseArgsCasesAreDeclaredInRootTree) fail loudly if step 1 and 2
// drift apart — the tree is the single source of truth.
//
// #170 migrates only the root command. serve / setup / desktop continue
// to use hand-rolled flag.FlagSet parsers in their own files; their
// migration onto Subcommands is tracked in #171. --profile and --port
// are already declared as Persistent so subcommand inheritance works
// out of the box once those migrations land.
var rootCommand = cmd.Command{
	Name:  "databricks-claude",
	Short: "Databricks AI Gateway proxy for Claude Code",
	Long:  rootHelpTemplate,

	// Persistent flags are inherited by every subcommand once those
	// commands migrate onto the tree (#171). For now, declaring them
	// here is a no-op for serve/setup/desktop (which have their own
	// FlagSets) but ensures the tree is shaped correctly for the
	// follow-up. Both flags also feed the resolution chain in main.go
	// today; the StateKey/EnvVar/Default fields below document that
	// behavior so #171/#172 can derive the chain from this declaration.
	Persistent: []cmd.FlagDef{
		{
			Name:        "profile",
			Description: "Databricks CLI profile (default: DEFAULT)",
			TakesArg:    true,
			Completer:   "__databricks_profiles",
			StateKey:    "profile",
			MDMKey:      "databricksProfile",
			Default:     "DEFAULT",
		},
		{
			Name:        "port",
			Description: "Proxy listen port (default: 49153)",
			TakesArg:    true,
			StateKey:    "port",
			Default:     "49153",
		},
	},

	// Order matches the legacy flagDefs slice so the bash/zsh/fish
	// completion output stays byte-identical with the pre-tree binary.
	// The few flags that previously ordered "profile" first now have
	// it pulled into Persistent (which renders first in AllFlags), so
	// "profile" still leads the completion script.
	Flags: []cmd.FlagDef{
		{Name: "verbose", Short: "v", Description: "Enable debug logging to stderr"},
		{Name: "version", Description: "Print version and exit"},
		{Name: "help", Short: "h", Description: "Show help message"},
		{Name: "print-env", Description: "Print resolved configuration (token redacted) and exit"},
		{Name: "otel", Description: "Enable OpenTelemetry logs/metrics export"},
		{Name: "no-otel", Description: "Disable OpenTelemetry (clears all signal keys + telemetry toggle)"},
		{Name: "no-otel-metrics", Description: "Clear persisted OTel metrics keys from settings.json"},
		{Name: "no-otel-logs", Description: "Clear persisted OTel logs keys from settings.json"},
		{Name: "no-otel-traces", Description: "Clear persisted OTel traces keys from settings.json"},
		{Name: "otel-metrics-table", Description: "Unity Catalog table for OTel metrics (cat.schema.table)", TakesArg: true,
			StateKey: "otel_metrics_table"},
		{Name: "otel-logs-table", Description: "Unity Catalog table for OTel logs (cat.schema.table)", TakesArg: true,
			StateKey: "otel_logs_table"},
		{Name: "otel-traces", Description: "Enable OpenTelemetry traces export (Claude Code beta)"},
		{Name: "otel-traces-table", Description: "Unity Catalog table for OTel traces (cat.schema.table)", TakesArg: true,
			StateKey: "otel_traces_table"},
		{Name: "upstream", Description: "Override upstream claude binary path", TakesArg: true, Completer: "__files"},
		{Name: "log-file", Description: "Write debug logs to file (combinable with --verbose)", TakesArg: true, Completer: "__files"},
		{Name: "proxy-api-key", Description: "Require this API key on all proxy requests", TakesArg: true},
		{Name: "tls-cert", Description: "TLS certificate file for the local proxy (requires --tls-key)", TakesArg: true, Completer: "__files"},
		{Name: "tls-key", Description: "TLS private key file for the local proxy (requires --tls-cert)", TakesArg: true, Completer: "__files"},
		{Name: "headless", Description: "Start proxy without launching claude (for IDE extensions or hooks)"},
		{Name: "write-claude-config", Description: "Write first-run settings.json env block and exit (no proxy, no port bind)"},
		{Name: "idle-timeout", Description: "Idle timeout for headless mode (default: 30m; 0 disables; use e.g. 30s, 5m, 1h)", TakesArg: true,
			Default: "30m"},
		{Name: "install-hooks", Description: "Install SessionStart/Stop hooks into ~/.claude/settings.json"},
		{Name: "uninstall-hooks", Description: "Remove databricks-claude hooks from ~/.claude/settings.json"},
		{Name: "headless-ensure", Description: "Start proxy if not running — called by the SessionStart hook"},
		{Name: "headless-release", Description: "Decrement proxy refcount — called by the Stop hook"},
		{Name: "no-update-check", Description: "Skip the automatic update check on startup",
			EnvVar: "DATABRICKS_NO_UPDATE_CHECK"},
		{Name: "with-websearch", Description: "Locally fulfill Anthropic web_search/web_fetch tools (workaround for FMAPI gap)",
			StateKey: "with_websearch"},
		{Name: "websearch-backend", Description: "Web search backend (duckduckgo|none)", TakesArg: true,
			StateKey: "websearch_backend", Default: "duckduckgo"},
		{Name: "websearch-fetch-budget", Description: "Per-fetch byte budget for --with-websearch (default 102400)", TakesArg: true,
			StateKey: "websearch_fetch_budget", Default: "102400"},
		// NOTE: --daemon and --daemon-fake-key are *desktop generate-config*
		// flags and are deliberately not declared here. They were previously
		// in the root flag set so completion would offer them, but completion
		// is position-1-only today (it can't tell whether the cursor is inside
		// `desktop`), so listing them at the root just polluted the root
		// completion namespace. The desktop subcommand has its own scanner
		// (extractDaemonFlag / extractDaemonFakeKeyFlag) so dropping them
		// here doesn't change parsing for `desktop generate-config --daemon
		// --daemon-fake-key=…`. They will land back as desktop-scoped flags
		// when `desktop` migrates onto the tree (#171).
	},

	// Subcommands are listed for completion at position 1. The Run
	// fields are intentionally nil — main() still owns dispatch in
	// #170. Each entry's Short matches the description in the legacy
	// knownSubcommands slice so completion output stays byte-identical.
	Subcommands: []cmd.Command{
		{Name: "completion", Short: "Generate shell completion scripts (bash, zsh, fish)"},
		{Name: "update", Short: "Check for a newer release and print upgrade instructions"},
		{Name: "desktop", Short: "Claude Desktop integration (generate-config, credential-helper)"},
		{Name: "setup", Short: "Idempotent auth bootstrap for fleet init scripts"},
		{Name: "serve", Short: "Long-lived daemon; sub-subcommands: install, uninstall, status"},
	},
}

// rootHelpTemplate is the verbatim help body rendered by handleHelp(). The
// "{{Version}}" placeholder is substituted by cmd.Render at print time.
//
// This template is hand-formatted (two distinct flag-column widths matching
// the legacy printf) to preserve byte-for-byte equivalence with the pre-tree
// help output. When subcommands migrate onto the tree (#171) they can use
// the programmatic renderer (cmd.Render with empty Long) for a uniform layout
// at the cost of touching every line; today the tree is still the single
// source of truth — every flag in the table also has a FlagDef above, the
// parity tests in main_test.go enforce that, and the completion scripts
// derive from those FlagDefs (not from this template).
const rootHelpTemplate = `databricks-claude v{{Version}} — Databricks AI Gateway proxy for Claude Code

Transparently proxies the Claude Code CLI with Databricks AI Gateway authentication
injected via environment variables.

Usage:
  databricks-claude [databricks-claude flags] [claude flags] [claude args]

Databricks-Claude Flags:
  --profile string      Databricks config profile (saved to ~/.claude/.databricks-claude.json)
  --upstream string     Override the AI Gateway URL (default: auto-discovered)
  --print-env           Print resolved configuration and exit (token redacted)
  --verbose, -v         Enable debug logging to stderr
  --log-file string     Write debug logs to a file (combinable with --verbose)
  --otel                       Enable OpenTelemetry metrics + logs export
  --otel-metrics-table string  Unity Catalog table for OTEL metrics
  --otel-logs-table string     Unity Catalog table for OTEL logs (derived from metrics table if omitted)
  --otel-traces                Enable OpenTelemetry traces export (Claude Code beta;
                               requires --otel-traces-table; can be used standalone)
  --otel-traces-table string   Unity Catalog table for OTEL traces (cat.schema.table)
  --no-otel                    Clear persisted OTEL keys and disable OTEL for future sessions
  --no-otel-metrics            Clear persisted OTEL metrics keys (other signals untouched)
  --no-otel-logs               Clear persisted OTEL logs keys (other signals untouched)
  --no-otel-traces             Clear persisted OTEL traces keys (other signals untouched)
  --proxy-api-key string       Require Bearer token auth on all proxy requests
  --tls-cert string            Path to TLS certificate file (requires --tls-key)
  --tls-key string             Path to TLS private key file (requires --tls-cert)
  --port int                   Fixed proxy port (default: 49153, saved to state)
  --headless                   Start proxy without launching claude (for IDE extensions)
  --write-claude-config        Write first-run settings.json env block (proxy URL, model
                               routing, custom headers) and exit — no proxy startup, no
                               port binding, no child process. Designed for MDM / fleet
                               init scripts and as a cleaner alternative to the
                               '--headless then Ctrl+C' workaround. Idempotent.
                               Accepts --profile, --port, and OTEL table flags.
  --headless-ensure            Start proxy if not running (called by SessionStart hook)
  --headless-release           Decrement proxy refcount (called by Stop hook)
  --idle-timeout duration      Idle timeout for headless mode (default 30m, 0 disables; use e.g. 30s, 5m, 1h)
  --install-hooks              Install SessionStart/SessionEnd hooks AND perform first-run
                               env setup (idempotent). Accepts --profile and --port to
                               persist them; no prior databricks-claude invocation needed.
  --uninstall-hooks            Remove databricks-claude hooks from ~/.claude/settings.json
  --no-update-check            Skip the automatic update check on startup
  --with-websearch             Enable local fulfillment of Anthropic web_search/web_fetch
                               tools (workaround until Databricks FMAPI ships native
                               support; saved to state). Default: disabled.
  --websearch-backend string   Search backend when --with-websearch is enabled.
                               Values: duckduckgo (default, zero config), none
  --websearch-fetch-budget int Max bytes returned per web_fetch call (default 102400)
  --version                    Print version and exit
  --help, -h                   Show this help message

Subcommands:
  completion <shell>           Generate shell completions (bash, zsh, fish)
  update                       Check for a newer release and print upgrade instructions
  desktop <action>             Claude Desktop integration. Run 'databricks-claude desktop'
                               for actions (generate-config, credential-helper) and flags.
  setup [flags]                Idempotent auth bootstrap. Persists the profile to state and
                               runs 'databricks auth login' when not authenticated. Designed
                               for fleet init scripts and per-user login agents.
                               Run 'databricks-claude setup --help' for flags.
  serve [install|uninstall|status|flags]
                               Long-lived daemon serving Claude Code and Claude
                               Desktop with persistent Databricks OAuth. Owns
                               OAuth refresh; exposes inference + OTLP on
                               127.0.0.1. No refcount, no /shutdown, append-only
                               logging. A third deployment mode alongside the
                               per-session CLI wrapper and SessionStart hooks.
                               Sub-subcommands register the daemon as a per-user
                               OS service (LaunchAgent/schtasks/systemd --user).
                               Run 'databricks-claude serve --help' for flags.
                               Run 'databricks-claude serve install --help' etc.

Example Unity Catalog table setup (run in a Databricks SQL warehouse):

  CREATE TABLE main.claude_telemetry.claude_otel_metrics (
    ... -- see https://docs.databricks.com/aws/en/ai-gateway/coding-agent-integration-beta
  ) USING DELTA TBLPROPERTIES ('otel.schemaVersion' = 'v1');

  CREATE TABLE main.claude_telemetry.claude_otel_logs (
    ... -- see https://docs.databricks.com/aws/en/ai-gateway/coding-agent-integration-beta
  ) USING DELTA TBLPROPERTIES ('otel.schemaVersion' = 'v1');

Passthrough to claude:
  Anything after a "--" separator is forwarded to the claude CLI unchanged.
  Examples:
    databricks-claude -- --help                # show claude's own help
    databricks-claude -- --model opus -p "hi"  # run claude with extra flags
`
