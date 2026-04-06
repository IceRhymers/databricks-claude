package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"github.com/IceRhymers/databricks-claude/pkg/childproc"
	"github.com/IceRhymers/databricks-claude/pkg/settings"
)

// SettingsManager reads, patches, and restores ~/.claude/settings.json.
type SettingsManager struct {
	inner              *settings.Manager
	otelKeysPersistent bool // when true, Restore skips OTEL keys
}

// NewSettingsManager creates a SettingsManager for the given settings.json path.
// It initializes a FileLock at ~/.claude/.settings.lock and a SessionRegistry
// at ~/.claude/.sessions.json (derived from the settings.json directory).
func NewSettingsManager(path string) *SettingsManager {
	dir := filepath.Dir(path)
	lock := NewFileLock(filepath.Join(dir, ".settings.lock"))
	reg := NewSessionRegistry(filepath.Join(dir, ".sessions.json"))

	// Wrap the registry to satisfy settings.SessionTracker interface.
	tracker := &registryAdapter{reg: reg}

	kc := settings.KeyConfig{
		ManagedKeys:   inferenceKeys,
		ProtectedKeys: otelKeys,
	}

	return &SettingsManager{
		inner: settings.NewManager(path, lock, tracker, kc),
	}
}

// registryAdapter wraps SessionRegistry to implement settings.SessionTracker.
type registryAdapter struct {
	reg *SessionRegistry
}

func (a *registryAdapter) Register(pid int, proxyURL string) error {
	return a.reg.Register(pid, proxyURL)
}

func (a *registryAdapter) Unregister(pid int) error {
	return a.reg.Unregister(pid)
}

func (a *registryAdapter) LiveSessions() ([]settings.LiveSession, error) {
	sessions, err := a.reg.LiveSessions()
	if err != nil {
		return nil, err
	}
	result := make([]settings.LiveSession, len(sessions))
	for i, s := range sessions {
		result[i] = settings.LiveSession{PID: s.PID, ProxyURL: s.ProxyURL}
	}
	return result, nil
}

func (a *registryAdapter) MostRecentLive() (*settings.LiveSession, error) {
	s, err := a.reg.MostRecentLive()
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, nil
	}
	return &settings.LiveSession{PID: s.PID, ProxyURL: s.ProxyURL}, nil
}

// SetOTELPersistent controls whether Restore skips OTEL keys. When true,
// Restore will not touch any keys in otelKeys or fullSetupOTELKeys, leaving
// them in settings.json as-is. Use ClearOTELKeys to explicitly remove them.
func (sm *SettingsManager) SetOTELPersistent(v bool) {
	sm.otelKeysPersistent = v
	sm.inner.SetProtectedPersistent(v)
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
	"OTEL_METRIC_EXPORT_INTERVAL",
	"OTEL_EXPORTER_OTLP_LOGS_ENDPOINT",
	"OTEL_EXPORTER_OTLP_LOGS_HEADERS",
	"OTEL_EXPORTER_OTLP_LOGS_PROTOCOL",
	"OTEL_LOGS_EXPORTER",
	"OTEL_LOGS_EXPORT_INTERVAL",
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
	"OTEL_METRIC_EXPORT_INTERVAL",
	"OTEL_EXPORTER_OTLP_LOGS_ENDPOINT",
	"OTEL_EXPORTER_OTLP_LOGS_HEADERS",
	"OTEL_EXPORTER_OTLP_LOGS_PROTOCOL",
	"OTEL_LOGS_EXPORTER",
	"OTEL_LOGS_EXPORT_INTERVAL",
	"CLAUDE_OTEL_UC_METRICS_TABLE",
	"CLAUDE_OTEL_UC_LOGS_TABLE",
}

// FullSetupConfig holds all parameters for FullSetup.
type FullSetupConfig struct {
	ProxyURL    string // e.g., "http://127.0.0.1:54321"
	Token       string // initial OAuth token
	Host        string // workspace URL (e.g., "https://dbc-xxx.cloud.databricks.com")
	Profile     string // CLI profile name
	UpstreamURL string // AI Gateway URL (stored for restore)

	// OTEL (optional)
	OTELEnabled      bool
	OTELMetricsTable string // default: "main.claude_telemetry.claude_otel_metrics"
	OTELLogsTable    string // default: derived from metrics table
}

// FullSetup reads settings.json (creating it if missing), saves originals for
// all keys it will write, then writes all Databricks/Claude proxy values.
// Restore() will undo everything FullSetup wrote.
func (sm *SettingsManager) FullSetup(config FullSetupConfig) error {
	if err := sm.inner.Lock().Lock(); err != nil {
		return fmt.Errorf("acquire file lock: %w", err)
	}
	defer sm.inner.Lock().Unlock()

	// Determine which keys we'll write.
	keysToWrite := append([]string{}, fullSetupInferenceKeys...)
	if config.OTELEnabled {
		keysToWrite = append(keysToWrite, fullSetupOTELKeys...)
	}

	// Read or create settings.json.
	doc, err := sm.inner.ReadSettings()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		doc = map[string]interface{}{"env": map[string]interface{}{}}
		dir := filepath.Dir(sm.inner.SettingsPath())
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create settings dir: %w", err)
		}
	}

	env := settings.GetEnvBlock(doc)

	// Save originals for all keys we will write (absent keys stored as nil sentinel).
	sm.inner.SaveOriginals(env, keysToWrite)

	// Write inference/Databricks keys.
	env["ANTHROPIC_BASE_URL"] = config.ProxyURL
	env["ANTHROPIC_AUTH_TOKEN"] = "proxy-managed" // proxy injects real token per-request

	// Model keys: preserve user-configured values, only write defaults for absent keys.
	modelDefaults := map[string]string{
		"ANTHROPIC_DEFAULT_OPUS_MODEL":   "databricks-claude-opus-4-6",
		"ANTHROPIC_DEFAULT_SONNET_MODEL": "databricks-claude-sonnet-4-6",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL":  "databricks-claude-haiku-4-5",
	}
	for k, def := range modelDefaults {
		if v, exists := env[k]; exists {
			log.Printf("databricks-claude: preserving user-configured %s=%v", k, v)
		} else {
			env[k] = def
		}
	}

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
		env["OTEL_METRIC_EXPORT_INTERVAL"] = "10000"
		env["OTEL_EXPORTER_OTLP_LOGS_ENDPOINT"] = config.ProxyURL + "/otel/v1/logs"
		env["OTEL_EXPORTER_OTLP_LOGS_HEADERS"] = "content-type=application/x-protobuf"
		env["OTEL_EXPORTER_OTLP_LOGS_PROTOCOL"] = "http/protobuf"
		env["OTEL_LOGS_EXPORTER"] = "otlp"
		env["OTEL_LOGS_EXPORT_INTERVAL"] = "5000"
		env["CLAUDE_OTEL_UC_METRICS_TABLE"] = config.OTELMetricsTable
		env["CLAUDE_OTEL_UC_LOGS_TABLE"] = config.OTELLogsTable
	}

	if err := sm.inner.WriteSettings(doc); err != nil {
		return err
	}

	// Register this session in the registry so other sessions know we're alive.
	if err := sm.inner.Registry().Register(os.Getpid(), config.ProxyURL); err != nil {
		log.Printf("databricks-claude: warning: failed to register session: %v", err)
	}

	return nil
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
	sm.inner.RestoreKeys(env, allKeys, skipOTEL)
}

// SaveAndOverwrite reads settings.json, saves original values for keys we will
// overwrite, then writes proxy values. Returns an error if settings.json does
// not exist.
func (sm *SettingsManager) SaveAndOverwrite(proxyURL string) error {
	return sm.inner.SaveAndOverwrite(proxyURL, func(env map[string]interface{}, proxyURL string) {
		// Determine whether OTEL was configured before we touched anything.
		_, otelConfigured := env["OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"]

		// Write inference proxy values.
		env["ANTHROPIC_BASE_URL"] = proxyURL
		env["ANTHROPIC_AUTH_TOKEN"] = "proxy-managed" // proxy injects real token per-request

		// Write OTEL proxy values only when OTEL was already configured.
		if otelConfigured {
			env["OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"] = proxyURL + "/otel/v1/metrics"
			// Strip any Bearer token from headers; the proxy re-adds it.
			env["OTEL_EXPORTER_OTLP_METRICS_HEADERS"] = "content-type=application/x-protobuf"
		}
	})
}

// Restore writes the original values back to settings.json. Keys that did not
// exist in the original are removed. When otelKeysPersistent is true, keys in
// otelKeys are skipped entirely (left as-is in settings.json).
//
// Smart handoff: if other live sessions exist, Restore hands off
// ANTHROPIC_BASE_URL to the most recent survivor instead of restoring the
// original upstream. Only the last session standing restores originals.
func (sm *SettingsManager) Restore() error {
	if err := sm.inner.Lock().Lock(); err != nil {
		return fmt.Errorf("acquire file lock: %w", err)
	}
	defer sm.inner.Lock().Unlock()

	// Unregister ourselves from the session registry.
	if err := sm.inner.Registry().Unregister(os.Getpid()); err != nil {
		log.Printf("databricks-claude: warning: failed to unregister session: %v", err)
	}

	// Check for other live sessions.
	live, err := sm.inner.Registry().LiveSessions()
	if err != nil {
		log.Printf("databricks-claude: warning: failed to read live sessions: %v", err)
		// Fall through to full restore on error.
		live = nil
	}

	doc, err := sm.inner.ReadSettings()
	if err != nil {
		return err
	}

	env := settings.GetEnvBlock(doc)

	if len(live) > 0 {
		// Other sessions still alive — hand off ANTHROPIC_BASE_URL to the
		// most recent survivor so it keeps working.
		survivor, sErr := sm.inner.Registry().MostRecentLive()
		if sErr == nil && survivor != nil {
			env["ANTHROPIC_BASE_URL"] = survivor.ProxyURL
			log.Printf("databricks-claude: handing off ANTHROPIC_BASE_URL to surviving session (PID %d, %s)", survivor.PID, survivor.ProxyURL)
		}
		// Leave all other keys as-is — the surviving session owns them.
	} else {
		// Last session — restore original values.

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
			if orig, had := sm.inner.OrigValues()[k]; had {
				env[k] = orig
			} else {
				delete(env, k)
			}
		}

		// Also restore any keys written by FullSetup (nil sentinel = delete).
		sm.restoreFullSetupKeys(env)
	}

	return sm.inner.WriteSettings(doc)
}

// ClearOTELKeys explicitly removes all OTEL keys from settings.json. This is
// used by --no-otel to ensure OTEL configuration is fully removed.
func (sm *SettingsManager) ClearOTELKeys() error {
	if err := sm.inner.Lock().Lock(); err != nil {
		return fmt.Errorf("acquire file lock: %w", err)
	}
	defer sm.inner.Lock().Unlock()

	doc, err := sm.inner.ReadSettings()
	if err != nil {
		return err
	}

	env := settings.GetEnvBlock(doc)
	allOTEL := append(otelKeys, fullSetupOTELKeys...)
	seen := map[string]bool{}
	for _, k := range allOTEL {
		if seen[k] {
			continue
		}
		seen[k] = true
		delete(env, k)
	}

	return sm.inner.WriteSettings(doc)
}

// readSettings reads and parses settings.json, returning the full document.
func (sm *SettingsManager) readSettings() (map[string]interface{}, error) {
	return sm.inner.ReadSettings()
}

// writeSettings atomically writes doc to settings.json.
func (sm *SettingsManager) writeSettings(doc map[string]interface{}) error {
	return sm.inner.WriteSettings(doc)
}

// getEnvBlock returns the "env" sub-map from the document, creating it if needed.
func getEnvBlock(doc map[string]interface{}) map[string]interface{} {
	return settings.GetEnvBlock(doc)
}

// RunChild starts claude as a child process with the supplied arguments and
// waits for it to exit, returning the exit code.
func RunChild(ctx context.Context, claudeArgs []string) (int, error) {
	return childproc.Run(ctx, childproc.Config{
		BinaryName: "claude",
		Args:       claudeArgs,
	})
}

// ForwardSignals sets up SIGINT/SIGTERM forwarding from the parent to cmd's
// process. The returned cancel function stops the forwarding goroutine.
func ForwardSignals(cmd *exec.Cmd) (cancel func()) {
	return childproc.ForwardSignals(cmd)
}

// readSettingsForSaveAndOverwrite is used by tests that directly access readLocked.
// We keep the registry accessible for concurrent_test.go.
func (sm *SettingsManager) registry() *SessionRegistry {
	return sm.inner.Registry().(*registryAdapter).reg
}

// origValues returns the saved originals map (for test access).
func (sm *SettingsManager) origValues() map[string]interface{} {
	return sm.inner.OrigValues()
}

// setOrigValue sets a specific original value (for test access).
func (sm *SettingsManager) setOrigValue(key string, value interface{}) {
	sm.inner.SetOrigValue(key, value)
}

