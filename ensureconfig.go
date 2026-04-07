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
func readSettingsJSON(path string) (map[string]interface{}, error) {
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
