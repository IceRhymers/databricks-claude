package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/IceRhymers/databricks-claude/pkg/authcheck"
	"github.com/IceRhymers/databricks-claude/pkg/portbind"
	"github.com/IceRhymers/databricks-claude/pkg/proxy"
	"github.com/IceRhymers/databricks-claude/pkg/refcount"
)

// Version is set at build time via -ldflags.
var Version = "dev"

func main() {
	// Parse databricks-claude flags, passing everything else through to claude.
	// Usage: databricks-claude [databricks-claude-flags] [--] [claude-args...]
	// Unknown flags are forwarded to claude automatically.
	profile, verbose, version, showHelp, printEnv, otel, otelMetricsTable, otelMetricsTableSet, otelLogsTable, otelLogsTableSet, upstream, logFile, noOtel, proxyAPIKey, tlsCert, tlsKey, portFlag, claudeArgs := parseArgs(os.Args[1:])

	if showHelp {
		handleHelp(upstream)
		os.Exit(0)
	}

	if version {
		fmt.Printf("databricks-claude %s\n", Version)
		os.Exit(0)
	}

	// --no-otel: clear persisted OTEL keys and proceed without OTEL this session.
	if noOtel {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("databricks-claude: cannot determine home dir: %v", err)
		}
		settingsPathForClear := filepath.Join(homeDir, ".claude", "settings.json")
		if err := clearOTELKeys(settingsPathForClear); err != nil {
			log.Fatalf("databricks-claude: failed to clear OTEL keys: %v", err)
		}
		fmt.Fprintln(os.Stderr, "databricks-claude: OTEL keys cleared — OTEL disabled for future sessions")
		// Continue without OTEL — otel remains false for the rest of this run.
	}

	// --- Resolve config from settings.json ---
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("databricks-claude: cannot determine home dir: %v", err)
	}

	// Default: discard all logs (silent wrapper — identical to vanilla claude).
	log.SetOutput(io.Discard)

	if verbose {
		log.SetOutput(os.Stderr)
	}
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			log.SetOutput(os.Stderr) // ensure this fatal is visible
			log.Fatalf("databricks-claude: cannot open log file %q: %v", logFile, err)
		}
		defer f.Close()
		if verbose {
			// Both stderr and file.
			log.SetOutput(io.MultiWriter(os.Stderr, f))
		} else {
			log.SetOutput(f)
		}
	}
	settingsPath := filepath.Join(homeDir, ".claude", "settings.json")

	settingsDoc, err := readSettingsDoc(settingsPath)
	if err != nil {
		log.Fatalf("databricks-claude: cannot read settings.json: %v", err)
	}

	env := envBlock(settingsDoc)

	// Resolve profile: --profile flag (saved to state file) > state file > "DEFAULT"
	// The env var DATABRICKS_CONFIG_PROFILE is NOT consulted here because
	// Claude's settings.json injects env vars into child processes, which
	// would override the user's explicit --profile choice persisted in the
	// state file.
	resolvedProfile := profile
	if resolvedProfile == "" {
		if saved := loadState(); saved.Profile != "" {
			resolvedProfile = saved.Profile
			log.Printf("databricks-claude: using profile %q from state file", saved.Profile)
		}
	}
	if resolvedProfile == "" {
		resolvedProfile = "DEFAULT"
	}

	// --- Ensure the user is authenticated before proceeding ---
	if err := authcheck.EnsureAuthenticated(resolvedProfile); err != nil {
		log.Fatalf("databricks-claude: auth failed: %v", err)
	}

	// --- Startup security checks ---
	for _, w := range proxy.SecurityChecks() {
		fmt.Fprintln(os.Stderr, w)
	}

	// Extract upstream values from settings.json.
	// ANTHROPIC_BASE_URL in settings.json is the real upstream at this point
	// (unless it's stale localhost, which we ignore).
	inferenceUpstream := ""
	if v, ok := env["ANTHROPIC_BASE_URL"].(string); ok {
		inferenceUpstream = v
	}
	// If it's already localhost (stale crash), treat as absent.
	if inferenceUpstream != "" && len(inferenceUpstream) >= 16 &&
		inferenceUpstream[:16] == "http://127.0.0.1" {
		inferenceUpstream = ""
	}
	databricksHost := ""
	if v, ok := env["DATABRICKS_HOST"].(string); ok {
		databricksHost = v
	}

	// Detect existing OTEL config.
	otelConfigured := false
	otelEndpoint := ""
	if v, ok := env["OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"].(string); ok && v != "" {
		otelConfigured = true
		otelEndpoint = v
	}
	// If OTEL endpoint is stale localhost (crash leftover), clear the URL but
	// keep otelConfigured=true so OTEL re-enables with the new proxy URL.
	if otelEndpoint != "" && strings.HasPrefix(otelEndpoint, "http://127.0.0.1") {
		otelEndpoint = ""
	}

	ucMetricsTable := ""
	if v, ok := env["CLAUDE_OTEL_UC_METRICS_TABLE"].(string); ok {
		ucMetricsTable = v
	}
	ucLogsTable := ""
	if v, ok := env["CLAUDE_OTEL_UC_LOGS_TABLE"].(string); ok {
		ucLogsTable = v
	}

	// --- Seed token cache ---
	tp := NewTokenProvider(resolvedProfile, "")
	initialToken, err := tp.Token(context.Background())
	if err != nil {
		log.Fatalf("databricks-claude: failed to fetch initial token for profile %q: %v", resolvedProfile, err)
	}

	// --- Self-setup: discover host + gateway URL if ANTHROPIC_BASE_URL is absent ---
	needsFullSetup := false

	// --upstream flag takes highest priority for the inference endpoint.
	if upstream != "" {
		inferenceUpstream = upstream
		log.Printf("databricks-claude: using explicit upstream: %s", inferenceUpstream)
		if databricksHost == "" {
			// Try to discover host for OTEL even when upstream is explicit.
			if h, err := DiscoverHost(resolvedProfile, ""); err == nil {
				databricksHost = h
			}
		}
		needsFullSetup = true
	} else if inferenceUpstream == "" {
		log.Printf("databricks-claude: no ANTHROPIC_BASE_URL configured — discovering from profile %q", resolvedProfile)

		host, err := DiscoverHost(resolvedProfile, "")
		if err != nil {
			log.Fatalf("databricks-claude: failed to discover host for profile %q: %v\nRun 'databricks auth login --profile %s' first",
				resolvedProfile, err, resolvedProfile)
		}
		log.Printf("databricks-claude: discovered host: %s", host)
		databricksHost = host

		inferenceUpstream = ConstructGatewayURL(host, initialToken)
		log.Printf("databricks-claude: upstream: %s", inferenceUpstream)

		needsFullSetup = true
	}

	// OTEL upstream: use host/api/2.0/otel when OTEL is enabled, else fall back.
	otelUpstream := otelEndpoint
	if otelUpstream == "" && databricksHost != "" {
		otelUpstream = databricksHost + "/api/2.0/otel"
	}
	if otelUpstream == "" {
		// Safe fallback so proxy.go doesn't crash.
		otelUpstream = inferenceUpstream
	}

	// OTEL metrics table: --otel-metrics-table flag overrides settings.json value.
	if otelMetricsTableSet {
		ucMetricsTable = otelMetricsTable
	} else if ucMetricsTable == "" {
		ucMetricsTable = otelMetricsTable
	}

	// OTEL logs table: --otel-logs-table flag overrides settings.json value.
	// Falls back to deriveLogsTable() when neither flag nor persisted value exists.
	if otelLogsTableSet {
		ucLogsTable = otelLogsTable
	} else if ucLogsTable == "" {
		ucLogsTable = deriveLogsTable(ucMetricsTable)
	}

	// --- Load persistent state and resolve port ---
	state := loadState()

	port := resolvePort(portFlag, state)
	if portFlag > 0 {
		state.Port = port
		if err := saveState(state); err != nil {
			log.Printf("databricks-claude: warning: failed to save port: %v", err)
		}
	}

	// Persist profile so future runs don't need --profile.
	if resolvedProfile != "DEFAULT" {
		state.Profile = resolvedProfile
		if err := saveState(state); err != nil {
			log.Printf("databricks-claude: warning: failed to persist profile: %v", err)
		} else {
			log.Printf("databricks-claude: persisted profile %q", resolvedProfile)
		}
	}

	// --- Print env and exit if requested ---
	if printEnv {
		handlePrintEnv(resolvedProfile, databricksHost, inferenceUpstream, initialToken, upstream, otel || otelConfigured, ucMetricsTable, ucLogsTable)
		os.Exit(0)
	}

	// --- Validate TLS config ---
	if err := proxy.ValidateTLSConfig(tlsCert, tlsKey); err != nil {
		log.Fatalf("databricks-claude: %v", err)
	}

	// --- Bind proxy port ---
	ln, isOwner, err := portbind.Bind("databricks-claude", port)
	if err != nil {
		log.Fatalf("databricks-claude: %v", err)
	}

	scheme := "http"
	if tlsCert != "" && tlsKey != "" {
		scheme = "https"
		fmt.Fprintln(os.Stderr, "databricks-claude: TLS enabled")
	}
	proxyURL := fmt.Sprintf("%s://127.0.0.1:%d", scheme, listenerPort(ln, port))

	// --- Start proxy if we own the port ---
	if isOwner {
		proxyConfig := &ProxyConfig{
			InferenceUpstream: inferenceUpstream,
			OTELUpstream:      otelUpstream,
			UCMetricsTable:    ucMetricsTable,
			UCLogsTable:       ucLogsTable,
			TokenProvider:     tp,
			Verbose:           verbose,
			APIKey:            proxyAPIKey,
			TLSCertFile:       tlsCert,
			TLSKeyFile:        tlsKey,
			ToolName:          "databricks-claude",
			Version:           Version,
		}
		if proxyAPIKey != "" {
			fmt.Fprintln(os.Stderr, "databricks-claude: proxy API key authentication enabled")
		}
		handler := NewProxyServer(proxyConfig)
		go func() {
			srv := &http.Server{Handler: handler}
			if tlsCert != "" && tlsKey != "" {
				if err := srv.ServeTLS(ln, tlsCert, tlsKey); err != nil && err != http.ErrServerClosed {
					log.Printf("databricks-claude: proxy serve error: %v", err)
				}
			} else {
				if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
					log.Printf("databricks-claude: proxy serve error: %v", err)
				}
			}
		}()
	}

	// --- Reference counting ---
	refcountPath := filepath.Join(os.TempDir(), fmt.Sprintf(".databricks-claude-sessions-%d", port))
	refcount.Acquire(refcountPath)
	defer func() {
		n, _ := refcount.Release(refcountPath)
		if n == 0 && isOwner {
			ln.Close()
		}
	}()

	// --- Write config once (idempotent) ---
	otelEnabled := otel || otelConfigured
	otelEnv := map[string]string{}
	if otelEnabled {
		otelEnv["OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"] = proxyURL + "/otel/v1/metrics"
		otelEnv["OTEL_EXPORTER_OTLP_METRICS_HEADERS"] = "content-type=application/x-protobuf"
		otelEnv["CLAUDE_CODE_ENABLE_TELEMETRY"] = "1"
		otelEnv["OTEL_METRICS_EXPORTER"] = "otlp"
		otelEnv["OTEL_EXPORTER_OTLP_METRICS_PROTOCOL"] = "http/protobuf"
		otelEnv["OTEL_METRIC_EXPORT_INTERVAL"] = "10000"
		otelEnv["OTEL_EXPORTER_OTLP_LOGS_ENDPOINT"] = proxyURL + "/otel/v1/logs"
		otelEnv["OTEL_EXPORTER_OTLP_LOGS_HEADERS"] = "content-type=application/x-protobuf"
		otelEnv["OTEL_EXPORTER_OTLP_LOGS_PROTOCOL"] = "http/protobuf"
		otelEnv["OTEL_LOGS_EXPORTER"] = "otlp"
		otelEnv["OTEL_LOGS_EXPORT_INTERVAL"] = "5000"
		otelEnv["CLAUDE_OTEL_UC_METRICS_TABLE"] = ucMetricsTable
		otelEnv["CLAUDE_OTEL_UC_LOGS_TABLE"] = ucLogsTable
	}
	if needsFullSetup {
		// Also write Databricks-specific keys for full setup.
		otelEnv["DATABRICKS_HOST"] = databricksHost
		otelEnv["DATABRICKS_CONFIG_PROFILE"] = resolvedProfile
		otelEnv["ANTHROPIC_DEFAULT_OPUS_MODEL"] = "databricks-claude-opus-4-6"
		otelEnv["ANTHROPIC_DEFAULT_SONNET_MODEL"] = "databricks-claude-sonnet-4-6"
		otelEnv["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = "databricks-claude-haiku-4-5"
		otelEnv["ANTHROPIC_CUSTOM_HEADERS"] = "x-databricks-use-coding-agent-mode: true"
		otelEnv["CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS"] = "1"
	}

	if err := ensureConfig(proxyURL, otelEnv); err != nil {
		log.Fatalf("databricks-claude: %v", err)
	}

	// --- Log startup info ---
	log.Printf("databricks-claude: proxy on %s (owner=%v), profile=%s, upstream=%s",
		proxyURL, isOwner, resolvedProfile, inferenceUpstream)

	// --- Run child ---
	exitCode, err := RunChild(context.Background(), claudeArgs)
	if err != nil {
		log.Printf("databricks-claude: child error: %v", err)
	}

	os.Exit(exitCode)
}

// listenerPort extracts the port from a net.Listener, falling back to the
// configured port if the listener is nil (e.g., non-owner case).
func listenerPort(ln net.Listener, fallback int) int {
	if ln == nil {
		return fallback
	}
	if addr, ok := ln.Addr().(*net.TCPAddr); ok {
		return addr.Port
	}
	return fallback
}

// readSettingsDoc reads and parses settings.json, returning the full document.
func readSettingsDoc(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// envBlock returns the "env" sub-map from a settings document, or an empty map.
func envBlock(doc map[string]interface{}) map[string]interface{} {
	if env, ok := doc["env"]; ok {
		if m, ok := env.(map[string]interface{}); ok {
			return m
		}
	}
	return map[string]interface{}{}
}

// parseArgs separates databricks-claude flags from claude flags.
// databricks-claude owns: --profile, --verbose/-v, --log-file, --version, --otel, --otel-metrics-table, --otel-logs-table, --no-otel, --proxy-api-key, --tls-cert, --tls-key.
// Everything else (including unknown flags like --debug) passes through to claude.
// An explicit "--" separator is supported but not required.
func parseArgs(args []string) (profile string, verbose bool, version bool, showHelp bool, printEnv bool, otel bool, otelMetricsTable string, otelMetricsTableSet bool, otelLogsTable string, otelLogsTableSet bool, upstream string, logFile string, noOtel bool, proxyAPIKey string, tlsCert string, tlsKey string, portFlag int, claudeArgs []string) {
	otelMetricsTable = "main.claude_telemetry.claude_otel_metrics" // default

	knownFlags := map[string]bool{
		"--profile":            true,
		"--verbose":            true,
		"--version":            true,
		"--help":               true,
		"--print-env":          true,
		"--otel":               true,
		"--no-otel":            true,
		"--otel-metrics-table": true,
		"--otel-logs-table":    true,
		"--upstream":           true,
		"--log-file":           true,
		"--proxy-api-key":     true,
		"--tls-cert":          true,
		"--tls-key":           true,
		"--port":              true,
	}

	i := 0
	for i < len(args) {
		arg := args[i]

		// Explicit separator: everything after "--" goes to claude.
		if arg == "--" {
			claudeArgs = append(claudeArgs, args[i+1:]...)
			return
		}

		// Special case: -h is a short flag for --help, -v for --verbose.
		if arg == "-h" {
			showHelp = true
			i++
			continue
		}
		if arg == "-v" {
			verbose = true
			i++
			continue
		}

		// Check if it's a known databricks-claude flag.
		if strings.HasPrefix(arg, "--") {
			// Handle --flag=value form.
			name := arg
			value := ""
			if eqIdx := strings.Index(arg, "="); eqIdx >= 0 {
				name = arg[:eqIdx]
				value = arg[eqIdx+1:]
			}

			if knownFlags[name] {
				switch name {
				case "--profile":
					if value != "" {
						profile = value
					} else if i+1 < len(args) {
						i++
						profile = args[i]
					}
				case "--otel-metrics-table":
					if value != "" {
						otelMetricsTable = value
						otelMetricsTableSet = true
					} else if i+1 < len(args) {
						i++
						otelMetricsTable = args[i]
						otelMetricsTableSet = true
					}
				case "--otel-logs-table":
					if value != "" {
						otelLogsTable = value
						otelLogsTableSet = true
					} else if i+1 < len(args) {
						i++
						otelLogsTable = args[i]
						otelLogsTableSet = true
					}
				case "--upstream":
					if value != "" {
						upstream = value
					} else if i+1 < len(args) {
						i++
						upstream = args[i]
					}
				case "--log-file":
					if value != "" {
						logFile = value
					} else if i+1 < len(args) {
						i++
						logFile = args[i]
					}
				case "--verbose":
					verbose = true
				case "--version":
					version = true
				case "--help":
					showHelp = true
				case "--print-env":
					printEnv = true
				case "--otel":
					otel = true
				case "--no-otel":
					noOtel = true
				case "--proxy-api-key":
					if value != "" {
						proxyAPIKey = value
					} else if i+1 < len(args) {
						i++
						proxyAPIKey = args[i]
					}
				case "--tls-cert":
					if value != "" {
						tlsCert = value
					} else if i+1 < len(args) {
						i++
						tlsCert = args[i]
					}
				case "--tls-key":
					if value != "" {
						tlsKey = value
					} else if i+1 < len(args) {
						i++
						tlsKey = args[i]
					}
				case "--port":
					if value != "" {
						portFlag, _ = strconv.Atoi(value)
					} else if i+1 < len(args) {
						i++
						portFlag, _ = strconv.Atoi(args[i])
					}
				}
				i++
				continue
			}
		}

		// Not a known databricks-claude flag — pass through to claude.
		claudeArgs = append(claudeArgs, arg)
		i++
	}
	return
}

// handleHelp prints the databricks-claude help message and appends claude's own --help output.
func handleHelp(upstreamBinary string) {
	fmt.Printf(`databricks-claude v%s — Databricks AI Gateway proxy for Claude Code

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
  --otel                       Enable OpenTelemetry tracing
  --no-otel                    Clear persisted OTEL keys and disable OTEL for future sessions
  --otel-metrics-table string  Unity Catalog table for OTEL metrics
  --otel-logs-table string     Unity Catalog table for OTEL logs (derived from metrics table if omitted)
  --proxy-api-key string       Require Bearer token auth on all proxy requests
  --tls-cert string            Path to TLS certificate file (requires --tls-key)
  --tls-key string             Path to TLS private key file (requires --tls-cert)
  --port int                   Fixed proxy port (default: 49153, saved to state)
  --version                    Print version and exit
  --help, -h                   Show this help message

Example Unity Catalog table setup (run in a Databricks SQL warehouse):

  CREATE TABLE main.claude_telemetry.claude_otel_metrics (
    ... -- see https://docs.databricks.com/aws/en/ai-gateway/coding-agent-integration-beta
  ) USING DELTA TBLPROPERTIES ('otel.schemaVersion' = 'v1');

  CREATE TABLE main.claude_telemetry.claude_otel_logs (
    ... -- see https://docs.databricks.com/aws/en/ai-gateway/coding-agent-integration-beta
  ) USING DELTA TBLPROPERTIES ('otel.schemaVersion' = 'v1');

────────────────────────────────────────────────────────────────────────────────
Claude CLI Options:
`, Version)

	// Determine which binary to run for --help passthrough.
	claudeBin := upstreamBinary
	if claudeBin == "" {
		if p, err := exec.LookPath("claude"); err == nil {
			claudeBin = p
		}
	}

	if claudeBin == "" {
		fmt.Println("(claude binary not found on PATH — install from https://claude.ai/code)")
		return
	}

	var buf bytes.Buffer
	cmd := exec.Command(claudeBin, "--help")
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	_ = cmd.Run()
	fmt.Print(buf.String())
}

// handlePrintEnv prints resolved configuration with the token redacted.
func handlePrintEnv(profile, databricksHost, anthropicBaseURL, token, upstreamBinary string, otelEnabled bool, otelMetricsTable, otelLogsTable string) {
	// Redact token.
	redacted := "**** (redacted)"
	if strings.HasPrefix(token, "dapi-") {
		redacted = "dapi-***"
	}

	// Resolve upstream binary path for display.
	binaryPath := upstreamBinary
	if binaryPath == "" {
		if p, err := exec.LookPath("claude"); err == nil {
			binaryPath = p
		} else {
			binaryPath = "(not found)"
		}
	}

	fmt.Printf(`databricks-claude configuration:
  Profile:              %s
  DATABRICKS_HOST:      %s
  ANTHROPIC_BASE_URL:   %s
  ANTHROPIC_AUTH_TOKEN: %s
  Upstream binary:      %s
  OTEL enabled:         %v
`, profile, databricksHost, anthropicBaseURL, redacted, binaryPath, otelEnabled)

	if otelEnabled {
		fmt.Printf(`  OTEL metrics table:   %s
  OTEL logs table:      %s
  OTEL metric interval: 10000ms
  OTEL logs interval:   5000ms
`, otelMetricsTable, otelLogsTable)
	}
}

// persistentConfigPath returns the path to the persistent config file.
func persistentConfigPath(homeDir string) string {
	return filepath.Join(homeDir, ".claude", ".databricks-claude.json")
}

// readPersistentConfig reads the persistent config file. Returns an empty map
// if the file does not exist.
func readPersistentConfig(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{}, nil
		}
		return nil, err
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// writePersistentConfig atomically writes the persistent config file.
func writePersistentConfig(path string, cfg map[string]interface{}) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// deriveLogsTable derives the OTEL logs table name from the metrics table name.
// If the metrics table ends with "_otel_metrics", replace that suffix with "_otel_logs".
// Otherwise, append "_otel_logs" as a sibling.
func deriveLogsTable(metricsTable string) string {
	if strings.HasSuffix(metricsTable, "_otel_metrics") {
		return strings.TrimSuffix(metricsTable, "_otel_metrics") + "_otel_logs"
	}
	return metricsTable + "_otel_logs"
}
