package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Version is set at build time via -ldflags.
var Version = "dev"

func main() {
	// Parse databricks-claude flags, passing everything else through to claude.
	// Usage: databricks-claude [databricks-claude-flags] [--] [claude-args...]
	// Unknown flags are forwarded to claude automatically.
	profile, verbose, version, showHelp, printEnv, otel, otelTable, upstream, logFile, claudeArgs := parseArgs(os.Args[1:])

	if showHelp {
		handleHelp(upstream)
		os.Exit(0)
	}

	if version {
		fmt.Printf("databricks-claude %s\n", Version)
		os.Exit(0)
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
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
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

	// Resolve profile: CLI flag > env var > settings.json > DEFAULT
	resolvedProfile := profile
	if resolvedProfile == "" {
		resolvedProfile = os.Getenv("DATABRICKS_CONFIG_PROFILE")
	}
	if resolvedProfile == "" {
		if v, ok := env["DATABRICKS_CONFIG_PROFILE"].(string); ok && v != "" {
			resolvedProfile = v
		}
	}
	if resolvedProfile == "" {
		resolvedProfile = "DEFAULT"
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

	ucTable := ""
	if v, ok := env["CLAUDE_OTEL_UC_TABLE"].(string); ok {
		ucTable = v
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

	// OTEL table: --otel-table flag overrides settings.json value.
	if ucTable == "" {
		ucTable = otelTable
	}

	// --- Print env and exit if requested ---
	if printEnv {
		anthropicModel := os.Getenv("ANTHROPIC_MODEL")
		handlePrintEnv(resolvedProfile, databricksHost, inferenceUpstream, initialToken, anthropicModel, upstream, otel || otelConfigured)
		os.Exit(0)
	}

	// --- Start proxy ---
	proxyConfig := &ProxyConfig{
		InferenceUpstream: inferenceUpstream,
		OTELUpstream:      otelUpstream,
		UCTable:           ucTable,
		TokenProvider:     tp,
		Verbose:           verbose,
	}
	handler := NewProxyServer(proxyConfig)
	listener, err := StartProxy(handler)
	if err != nil {
		log.Fatalf("databricks-claude: failed to start proxy: %v", err)
	}
	proxyURL := "http://" + listener.Addr().String()

	// --- Patch settings.json ---
	sm := NewSettingsManager(settingsPath)
	otelEnabled := otel || otelConfigured
	if needsFullSetup {
		if err := sm.FullSetup(FullSetupConfig{
			ProxyURL:    proxyURL,
			Token:       initialToken,
			Host:        databricksHost,
			Profile:     resolvedProfile,
			UpstreamURL: inferenceUpstream,
			OTELEnabled: otelEnabled,
			OTELTable:   otelTable,
		}); err != nil {
			log.Fatalf("databricks-claude: failed to write settings.json: %v", err)
		}
		log.Printf("databricks-claude: self-setup complete — profile=%s, host=%s, gateway=%s",
			resolvedProfile, databricksHost, inferenceUpstream)
	} else {
		if err := sm.SaveAndOverwrite(proxyURL); err != nil {
			log.Fatalf("databricks-claude: failed to patch settings.json: %v", err)
		}
	}
	defer func() {
		if err := sm.Restore(); err != nil {
			log.Printf("databricks-claude: warning: failed to restore settings.json: %v", err)
		}
	}()

	// --- Log startup info ---
	log.Printf("databricks-claude: proxy listening on %s, profile=%s, upstream=%s",
		listener.Addr().String(), resolvedProfile, inferenceUpstream)

	// --- Run child ---
	exitCode, err := RunChild(context.Background(), claudeArgs)
	if err != nil {
		log.Printf("databricks-claude: child error: %v", err)
	}

	// Restore happens via defer above before os.Exit.
	os.Exit(exitCode)
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
// databricks-claude owns: --profile, --verbose/-v, --log-file, --version, --otel, --otel-table.
// Everything else (including unknown flags like --debug) passes through to claude.
// An explicit "--" separator is supported but not required.
func parseArgs(args []string) (profile string, verbose bool, version bool, showHelp bool, printEnv bool, otel bool, otelTable string, upstream string, logFile string, claudeArgs []string) {
	otelTable = "main.claude_telemetry.claude_otel_metrics" // default

	knownFlags := map[string]bool{
		"--profile":    true,
		"--verbose":    true,
		"--version":    true,
		"--help":       true,
		"--print-env":  true,
		"--otel":       true,
		"--otel-table": true,
		"--upstream":   true,
		"--log-file":   true,
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
				case "--otel-table":
					if value != "" {
						otelTable = value
					} else if i+1 < len(args) {
						i++
						otelTable = args[i]
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
  --profile string      Databricks config profile (default "DEFAULT")
  --upstream string     Override the AI Gateway URL (default: auto-discovered)
  --print-env           Print resolved configuration and exit (token redacted)
  --verbose, -v         Enable debug logging to stderr
  --log-file string     Write debug logs to a file (combinable with --verbose)
  --otel                Enable OpenTelemetry tracing
  --otel-table string   Unity Catalog table for OTEL spans
  --version             Print version and exit
  --help, -h            Show this help message

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
func handlePrintEnv(profile, databricksHost, anthropicBaseURL, token, anthropicModel, upstreamBinary string, otelEnabled bool) {
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
  ANTHROPIC_MODEL:      %s
  Upstream binary:      %s
  OTEL enabled:         %v
`, profile, databricksHost, anthropicBaseURL, redacted, anthropicModel, binaryPath, otelEnabled)
}
