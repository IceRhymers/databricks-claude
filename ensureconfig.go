package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// ensureConfig writes the env block to ~/.claude/settings.json only if
// ANTHROPIC_BASE_URL doesn't already point at proxyURL. Idempotent.
//
// It preserves all non-env keys in settings.json. The otelKeys map is optional;
// non-empty values are written into the env block.
func ensureConfig(proxyURL string, otelEnv map[string]string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	settingsPath := filepath.Join(homeDir, ".claude", "settings.json")

	// Read existing settings.json or start with empty doc.
	doc, err := readSettingsJSON(settingsPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		doc = map[string]interface{}{}
		if err := os.MkdirAll(filepath.Dir(settingsPath), 0o700); err != nil {
			return err
		}
	}

	// Get or create the env block.
	env := getOrCreateEnvBlock(doc)

	// Check if already configured — no-op if ANTHROPIC_BASE_URL matches.
	if v, ok := env["ANTHROPIC_BASE_URL"].(string); ok && v == proxyURL {
		return nil
	}

	// Write proxy values.
	env["ANTHROPIC_BASE_URL"] = proxyURL
	env["ANTHROPIC_AUTH_TOKEN"] = "proxy-managed"

	// Write any non-empty OTEL keys.
	for k, v := range otelEnv {
		if v != "" {
			env[k] = v
		}
	}

	doc["env"] = env
	return writeSettingsJSON(settingsPath, doc)
}

// readSettingsJSON reads and parses a settings.json file.
// If the file does not exist, an empty document is returned.
func readSettingsJSON(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{}, nil
		}
		return nil, err
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// writeSettingsJSON atomically writes a settings.json file.
func writeSettingsJSON(path string, doc map[string]interface{}) error {
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// getOrCreateEnvBlock returns the "env" sub-map from doc, creating it if needed.
func getOrCreateEnvBlock(doc map[string]interface{}) map[string]interface{} {
	if env, ok := doc["env"]; ok {
		if m, ok := env.(map[string]interface{}); ok {
			return m
		}
	}
	env := map[string]interface{}{}
	doc["env"] = env
	return env
}

// clearOTELKeys removes all OTEL-related env keys from ~/.claude/settings.json.
// Used by --no-otel to ensure telemetry config is fully absent from settings.
func clearOTELKeys(settingsPath string) error {
	doc, err := readSettingsJSON(settingsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	env, ok := doc["env"].(map[string]interface{})
	if !ok {
		return nil
	}
	for _, k := range otelEnvKeys {
		delete(env, k)
	}
	return writeSettingsJSON(settingsPath, doc)
}

// otelEnvKeys is the complete list of OTEL env keys managed by databricks-claude.
var otelEnvKeys = []string{
	"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT",
	"OTEL_EXPORTER_OTLP_METRICS_HEADERS",
	"OTEL_EXPORTER_OTLP_METRICS_PROTOCOL",
	"CLAUDE_CODE_ENABLE_TELEMETRY",
	"OTEL_METRICS_EXPORTER",
	"OTEL_METRIC_EXPORT_INTERVAL",
	"OTEL_EXPORTER_OTLP_LOGS_ENDPOINT",
	"OTEL_EXPORTER_OTLP_LOGS_HEADERS",
	"OTEL_EXPORTER_OTLP_LOGS_PROTOCOL",
	"OTEL_LOGS_EXPORTER",
	"OTEL_LOGS_EXPORT_INTERVAL",
	"CLAUDE_OTEL_UC_METRICS_TABLE",
	"CLAUDE_OTEL_UC_LOGS_TABLE",
}
