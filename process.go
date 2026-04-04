package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

// SettingsManager reads, patches, and restores ~/.claude/settings.json.
type SettingsManager struct {
	settingsPath    string
	origValues      map[string]interface{} // saved originals for restore
	otelKeysPersistent bool                // when true, Restore skips OTEL keys
	mu              sync.Mutex
}

// NewSettingsManager creates a SettingsManager for the given settings.json path.
func NewSettingsManager(path string) *SettingsManager {
	return &SettingsManager{
		settingsPath: path,
		origValues:   make(map[string]interface{}),
	}
}

// SetOTELPersistent controls whether Restore skips OTEL keys. When true,
// Restore will not touch any keys in otelKeys or fullSetupOTELKeys, leaving
// them in settings.json as-is. Use ClearOTELKeys to explicitly remove them.
func (sm *SettingsManager) SetOTELPersistent(v bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.otelKeysPersistent = v
}

// readSettings reads and parses settings.json, returning the full document.
func (sm *SettingsManager) readSettings() (map[string]interface{}, error) {
	data, err := os.ReadFile(sm.settingsPath)
	if err != nil {
		return nil, fmt.Errorf("read settings.json: %w", err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse settings.json: %w", err)
	}
	return doc, nil
}

// writeSettings atomically writes doc to settings.json using a temp file in
// the same directory followed by os.Rename.
func (sm *SettingsManager) writeSettings(doc map[string]interface{}) error {
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings.json: %w", err)
	}
	dir := filepath.Dir(sm.settingsPath)
	tmp, err := os.CreateTemp(dir, ".settings-*.json.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpName, sm.settingsPath); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}

// getEnvBlock returns the "env" sub-map from the document, creating it if needed.
func getEnvBlock(doc map[string]interface{}) map[string]interface{} {
	if env, ok := doc["env"]; ok {
		if envMap, ok := env.(map[string]interface{}); ok {
			return envMap
		}
	}
	envMap := make(map[string]interface{})
	doc["env"] = envMap
	return envMap
}

// keysToManage lists the env keys we read/restore. OTEL keys are only touched
// when they already exist in the original settings.
var inferenceKeys = []string{
	"ANTHROPIC_BASE_URL",
	"ANTHROPIC_AUTH_TOKEN", // set to "proxy-managed" — proxy injects real token per-request
}

var otelKeys = []string{
	"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT",
	"OTEL_EXPORTER_OTLP_METRICS_HEADERS",
}

// fullSetupInferenceKeys lists the env keys written by FullSetup (non-OTEL).
var fullSetupInferenceKeys = []string{
	"ANTHROPIC_BASE_URL",
	"ANTHROPIC_AUTH_TOKEN",
	"ANTHROPIC_DEFAULT_OPUS_MODEL",
	"ANTHROPIC_DEFAULT_SONNET_MODEL",
	"ANTHROPIC_DEFAULT_HAIKU_MODEL",
	"ANTHROPIC_CUSTOM_HEADERS",
	"CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS",
	"DATABRICKS_HOST",
	"DATABRICKS_CONFIG_PROFILE",
	// Note: DATABRICKS_TOKEN is intentionally omitted — budget-checker and MCP
	// get tokens dynamically from the CLI, and the proxy handles inference auth.
}

// fullSetupOTELKeys lists the OTEL env keys written by FullSetup when OTELEnabled.
var fullSetupOTELKeys = []string{
	"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT",
	"OTEL_EXPORTER_OTLP_METRICS_HEADERS",
	"CLAUDE_CODE_ENABLE_TELEMETRY",
	"OTEL_METRICS_EXPORTER",
	"OTEL_EXPORTER_OTLP_METRICS_PROTOCOL",
}

// FullSetupConfig holds all parameters for FullSetup.
type FullSetupConfig struct {
	ProxyURL    string // e.g., "http://127.0.0.1:54321"
	Token       string // initial OAuth token
	Host        string // workspace URL (e.g., "https://dbc-xxx.cloud.databricks.com")
	Profile     string // CLI profile name
	UpstreamURL string // AI Gateway URL (stored for restore)

	// OTEL (optional)
	OTELEnabled bool
	OTELTable   string // default: "main.claude_telemetry.claude_otel_metrics"
}

// FullSetup reads settings.json (creating it if missing), saves originals for
// all keys it will write, then writes all Databricks/Claude proxy values.
// Restore() will undo everything FullSetup wrote.
func (sm *SettingsManager) FullSetup(config FullSetupConfig) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Determine which keys we'll write.
	keysToWrite := append([]string{}, fullSetupInferenceKeys...)
	if config.OTELEnabled {
		keysToWrite = append(keysToWrite, fullSetupOTELKeys...)
	}

	// Read or create settings.json.
	doc, err := sm.readSettings()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		doc = map[string]interface{}{"env": map[string]interface{}{}}
		dir := filepath.Dir(sm.settingsPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create settings dir: %w", err)
		}
	}

	env := getEnvBlock(doc)

	// Save originals for all keys we will write (absent keys stored as nil sentinel).
	for _, k := range keysToWrite {
		if v, exists := env[k]; exists {
			sm.origValues[k] = v
		} else {
			// Explicit nil signals Restore to delete this key.
			sm.origValues[k] = nil
		}
	}

	// Write inference/Databricks keys.
	env["ANTHROPIC_BASE_URL"] = config.ProxyURL
	env["ANTHROPIC_AUTH_TOKEN"] = "proxy-managed" // proxy injects real token per-request
	env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = "databricks-claude-opus-4-6"
	env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = "databricks-claude-sonnet-4-5"
	env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = "databricks-claude-haiku-4-5"
	env["ANTHROPIC_CUSTOM_HEADERS"] = "x-databricks-use-coding-agent-mode: true"
	env["CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS"] = "1"
	env["DATABRICKS_HOST"] = config.Host
	env["DATABRICKS_CONFIG_PROFILE"] = config.Profile

	// Write OTEL keys only when enabled.
	if config.OTELEnabled {
		env["OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"] = config.ProxyURL + "/otel/v1/metrics"
		env["OTEL_EXPORTER_OTLP_METRICS_HEADERS"] = "content-type=application/x-protobuf"
		env["CLAUDE_CODE_ENABLE_TELEMETRY"] = "1"
		env["OTEL_METRICS_EXPORTER"] = "otlp"
		env["OTEL_EXPORTER_OTLP_METRICS_PROTOCOL"] = "http/protobuf"
	}

	return sm.writeSettings(doc)
}

// restoreFullSetupKeys restores keys written by FullSetup. It handles the nil
// sentinel (absent before FullSetup) by deleting the key. When
// otelKeysPersistent is true, keys in fullSetupOTELKeys are skipped.
func (sm *SettingsManager) restoreFullSetupKeys(env map[string]interface{}) {
	// Build set of OTEL keys to skip when persistent mode is on.
	skipOTEL := map[string]bool{}
	if sm.otelKeysPersistent {
		for _, k := range fullSetupOTELKeys {
			skipOTEL[k] = true
		}
	}

	allKeys := append(fullSetupInferenceKeys, fullSetupOTELKeys...)
	for _, k := range allKeys {
		if skipOTEL[k] {
			continue
		}
		orig, tracked := sm.origValues[k]
		if !tracked {
			continue
		}
		if orig == nil {
			delete(env, k)
		} else {
			env[k] = orig
		}
	}
}

// SaveAndOverwrite reads settings.json, saves original values for keys we will
// overwrite, then writes proxy values. Returns an error if settings.json does
// not exist.
func (sm *SettingsManager) SaveAndOverwrite(proxyURL string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	doc, err := sm.readSettings()
	if err != nil {
		return err
	}

	env := getEnvBlock(doc)

	// Determine whether OTEL was configured before we touched anything.
	_, otelConfigured := env["OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"]

	// Save originals for all keys we may touch.
	allKeys := append(inferenceKeys, otelKeys...)
	for _, k := range allKeys {
		if v, exists := env[k]; exists {
			sm.origValues[k] = v
		}
		// If it didn't exist, we leave origValues[k] unset — Restore will delete the key.
	}

	// If ANTHROPIC_BASE_URL already points to localhost (stale from a crash), clear it
	// before setting it to the new proxy URL so we don't keep a stale value as the "original".
	if orig, ok := sm.origValues["ANTHROPIC_BASE_URL"]; ok {
		if s, ok := orig.(string); ok && strings.HasPrefix(s, "http://127.0.0.1") {
			// Stale localhost value — treat it as if it was absent.
			delete(sm.origValues, "ANTHROPIC_BASE_URL")
		}
	}

	// Write inference proxy values.
	env["ANTHROPIC_BASE_URL"] = proxyURL
	env["ANTHROPIC_AUTH_TOKEN"] = "proxy-managed" // proxy injects real token per-request

	// Write OTEL proxy values only when OTEL was already configured.
	if otelConfigured {
		env["OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"] = proxyURL + "/otel/v1/metrics"
		// Strip any Bearer token from headers; the proxy re-adds it.
		env["OTEL_EXPORTER_OTLP_METRICS_HEADERS"] = "content-type=application/x-protobuf"
	}

	return sm.writeSettings(doc)
}

// Restore writes the original values back to settings.json. Keys that did not
// exist in the original are removed. When otelKeysPersistent is true, keys in
// otelKeys are skipped entirely (left as-is in settings.json).
func (sm *SettingsManager) Restore() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	doc, err := sm.readSettings()
	if err != nil {
		return err
	}

	env := getEnvBlock(doc)

	// Build set of OTEL keys to skip when persistent mode is on.
	skipOTEL := map[string]bool{}
	if sm.otelKeysPersistent {
		for _, k := range otelKeys {
			skipOTEL[k] = true
		}
	}

	allKeys := append(inferenceKeys, otelKeys...)
	for _, k := range allKeys {
		if skipOTEL[k] {
			continue
		}
		if orig, had := sm.origValues[k]; had {
			env[k] = orig
		} else {
			delete(env, k)
		}
	}

	// Also restore any keys written by FullSetup (nil sentinel = delete).
	sm.restoreFullSetupKeys(env)

	return sm.writeSettings(doc)
}

// ClearOTELKeys explicitly removes all OTEL keys from settings.json. This is
// used by --no-otel to ensure OTEL configuration is fully removed.
func (sm *SettingsManager) ClearOTELKeys() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	doc, err := sm.readSettings()
	if err != nil {
		return err
	}

	env := getEnvBlock(doc)
	allOTEL := append(otelKeys, fullSetupOTELKeys...)
	seen := map[string]bool{}
	for _, k := range allOTEL {
		if seen[k] {
			continue
		}
		seen[k] = true
		delete(env, k)
	}

	return sm.writeSettings(doc)
}

// RunChild starts claude as a child process with the supplied arguments and
// waits for it to exit, returning the exit code.
func RunChild(ctx context.Context, claudeArgs []string) (int, error) {
	cmd := exec.CommandContext(ctx, "claude", claudeArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return 1, fmt.Errorf("start claude: %w", err)
	}

	cancel := ForwardSignals(cmd)
	defer cancel()

	err := cmd.Wait()
	if err == nil {
		return 0, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode(), nil
	}
	return 1, fmt.Errorf("wait claude: %w", err)
}

// ForwardSignals sets up SIGINT/SIGTERM forwarding from the parent to cmd's
// process. The returned cancel function stops the forwarding goroutine.
func ForwardSignals(cmd *exec.Cmd) (cancel func()) {
	ch := make(chan os.Signal, 4)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	done := make(chan struct{})
	go func() {
		defer signal.Stop(ch)
		for {
			select {
			case sig := <-ch:
				if cmd.Process != nil {
					cmd.Process.Signal(sig)
				}
			case <-done:
				return
			}
		}
	}()

	return func() {
		close(done)
	}
}
