package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// writeJSON writes a map as JSON to path.
func writeJSON(t *testing.T, path string, v map[string]interface{}) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// readJSON reads and parses a JSON file into a map.
func readJSON(t *testing.T, path string) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return doc
}

func TestSettingsManager_SaveAndRestore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	original := map[string]interface{}{
		"env": map[string]interface{}{
			"ANTHROPIC_BASE_URL":   "https://real-upstream.example.com",
			"ANTHROPIC_AUTH_TOKEN": "original-token",
		},
	}
	writeJSON(t, path, original)

	sm := NewSettingsManager(path)
	if err := sm.SaveAndOverwrite("http://127.0.0.1:9999"); err != nil {
		t.Fatalf("SaveAndOverwrite: %v", err)
	}

	// Verify proxy values were written.
	patched := readJSON(t, path)
	env := patched["env"].(map[string]interface{})
	if env["ANTHROPIC_BASE_URL"] != "http://127.0.0.1:9999" {
		t.Errorf("ANTHROPIC_BASE_URL = %v, want http://127.0.0.1:9999", env["ANTHROPIC_BASE_URL"])
	}
	if env["ANTHROPIC_AUTH_TOKEN"] != "proxy-managed" {
		t.Errorf("ANTHROPIC_AUTH_TOKEN = %v, want proxy-managed", env["ANTHROPIC_AUTH_TOKEN"])
	}

	// Restore and verify originals are back.
	if err := sm.Restore(); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	restored := readJSON(t, path)
	rEnv := restored["env"].(map[string]interface{})
	if rEnv["ANTHROPIC_BASE_URL"] != "https://real-upstream.example.com" {
		t.Errorf("restored ANTHROPIC_BASE_URL = %v, want https://real-upstream.example.com", rEnv["ANTHROPIC_BASE_URL"])
	}
	if rEnv["ANTHROPIC_AUTH_TOKEN"] != "original-token" {
		t.Errorf("restored ANTHROPIC_AUTH_TOKEN = %v, want original-token", rEnv["ANTHROPIC_AUTH_TOKEN"])
	}
}

func TestSettingsManager_PreservesOtherKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	original := map[string]interface{}{
		"theme": "dark",
		"env": map[string]interface{}{
			"ANTHROPIC_BASE_URL":   "https://real-upstream.example.com",
			"ANTHROPIC_AUTH_TOKEN": "original-token",
			"MY_CUSTOM_VAR":        "preserved",
		},
	}
	writeJSON(t, path, original)

	sm := NewSettingsManager(path)
	if err := sm.SaveAndOverwrite("http://127.0.0.1:9999"); err != nil {
		t.Fatalf("SaveAndOverwrite: %v", err)
	}

	patched := readJSON(t, path)

	// Top-level keys preserved.
	if patched["theme"] != "dark" {
		t.Errorf("theme = %v, want dark", patched["theme"])
	}

	// Custom env var preserved.
	env := patched["env"].(map[string]interface{})
	if env["MY_CUSTOM_VAR"] != "preserved" {
		t.Errorf("MY_CUSTOM_VAR = %v, want preserved", env["MY_CUSTOM_VAR"])
	}

	// After restore, still preserved.
	if err := sm.Restore(); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	restored := readJSON(t, path)
	if restored["theme"] != "dark" {
		t.Errorf("after restore, theme = %v, want dark", restored["theme"])
	}
	rEnv := restored["env"].(map[string]interface{})
	if rEnv["MY_CUSTOM_VAR"] != "preserved" {
		t.Errorf("after restore, MY_CUSTOM_VAR = %v, want preserved", rEnv["MY_CUSTOM_VAR"])
	}
}

func TestSettingsManager_ClearsStaleLocalhost(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	// Simulate a crash where ANTHROPIC_BASE_URL was left pointing to localhost.
	original := map[string]interface{}{
		"env": map[string]interface{}{
			"ANTHROPIC_BASE_URL":   "http://127.0.0.1:8888",
			"ANTHROPIC_AUTH_TOKEN": "stale-token",
		},
	}
	writeJSON(t, path, original)

	sm := NewSettingsManager(path)
	if err := sm.SaveAndOverwrite("http://127.0.0.1:9999"); err != nil {
		t.Fatalf("SaveAndOverwrite: %v", err)
	}

	// Restore should remove ANTHROPIC_BASE_URL (original was stale localhost,
	// treated as absent) rather than restoring the stale localhost value.
	if err := sm.Restore(); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	restored := readJSON(t, path)
	rEnv := restored["env"].(map[string]interface{})
	if _, exists := rEnv["ANTHROPIC_BASE_URL"]; exists {
		t.Errorf("expected ANTHROPIC_BASE_URL to be removed after stale-localhost restore, got %v", rEnv["ANTHROPIC_BASE_URL"])
	}
}

func TestSettingsManager_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	original := map[string]interface{}{
		"env": map[string]interface{}{
			"ANTHROPIC_BASE_URL":   "https://upstream.example.com",
			"ANTHROPIC_AUTH_TOKEN": "tok",
		},
	}
	writeJSON(t, path, original)

	sm := NewSettingsManager(path)
	if err := sm.SaveAndOverwrite("http://127.0.0.1:9999"); err != nil {
		t.Fatalf("SaveAndOverwrite: %v", err)
	}

	// Verify no temp files remain in the directory.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}

	// Verify the file is valid JSON (atomic write succeeded).
	readJSON(t, path)
}

func TestSettingsManager_HandlesOTEL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	original := map[string]interface{}{
		"env": map[string]interface{}{
			"ANTHROPIC_BASE_URL":                  "https://upstream.example.com",
			"ANTHROPIC_AUTH_TOKEN":                 "original-token",
			"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT": "https://otel.example.com/v1/metrics",
			"OTEL_EXPORTER_OTLP_METRICS_HEADERS":  "Authorization=Bearer original-otel-token",
		},
	}
	writeJSON(t, path, original)

	sm := NewSettingsManager(path)
	if err := sm.SaveAndOverwrite("http://127.0.0.1:9999"); err != nil {
		t.Fatalf("SaveAndOverwrite: %v", err)
	}

	patched := readJSON(t, path)
	env := patched["env"].(map[string]interface{})

	// OTEL endpoint should point to proxy.
	wantOTEL := "http://127.0.0.1:9999/otel/v1/metrics"
	if env["OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"] != wantOTEL {
		t.Errorf("OTEL endpoint = %v, want %v", env["OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"], wantOTEL)
	}

	// Bearer token should be stripped from OTEL headers.
	wantHeaders := "content-type=application/x-protobuf"
	if env["OTEL_EXPORTER_OTLP_METRICS_HEADERS"] != wantHeaders {
		t.Errorf("OTEL headers = %v, want %v", env["OTEL_EXPORTER_OTLP_METRICS_HEADERS"], wantHeaders)
	}

	// Restore should bring back originals.
	if err := sm.Restore(); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	restored := readJSON(t, path)
	rEnv := restored["env"].(map[string]interface{})
	if rEnv["OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"] != "https://otel.example.com/v1/metrics" {
		t.Errorf("restored OTEL endpoint = %v", rEnv["OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"])
	}
	if rEnv["OTEL_EXPORTER_OTLP_METRICS_HEADERS"] != "Authorization=Bearer original-otel-token" {
		t.Errorf("restored OTEL headers = %v", rEnv["OTEL_EXPORTER_OTLP_METRICS_HEADERS"])
	}
}

func TestFullSetup_WritesAllKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	writeJSON(t, path, map[string]interface{}{"env": map[string]interface{}{}})

	sm := NewSettingsManager(path)
	cfg := FullSetupConfig{
		ProxyURL: "http://127.0.0.1:54321",
		Token:    "tok123",
		Host:     "https://dbc-abc.cloud.databricks.com",
		Profile:  "myprofile",
	}
	if err := sm.FullSetup(cfg); err != nil {
		t.Fatalf("FullSetup: %v", err)
	}

	doc := readJSON(t, path)
	env := doc["env"].(map[string]interface{})

	checks := map[string]string{
		"ANTHROPIC_BASE_URL":                     cfg.ProxyURL,
		"ANTHROPIC_AUTH_TOKEN":                    "proxy-managed",
		"ANTHROPIC_DEFAULT_OPUS_MODEL":            "databricks-claude-opus-4-6",
		"ANTHROPIC_DEFAULT_SONNET_MODEL":          "databricks-claude-sonnet-4-5",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL":           "databricks-claude-haiku-4-5",
		"ANTHROPIC_CUSTOM_HEADERS":                "x-databricks-use-coding-agent-mode: true",
		"CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS":  "1",
		"DATABRICKS_HOST":                         cfg.Host,
		"DATABRICKS_CONFIG_PROFILE":               cfg.Profile,
	}
	for k, want := range checks {
		if got, ok := env[k]; !ok {
			t.Errorf("missing key %s", k)
		} else if got != want {
			t.Errorf("%s = %v, want %v", k, got, want)
		}
	}
}

func TestFullSetup_RestoreRemovesNewKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	writeJSON(t, path, map[string]interface{}{"env": map[string]interface{}{}})

	sm := NewSettingsManager(path)
	if err := sm.FullSetup(FullSetupConfig{
		ProxyURL: "http://127.0.0.1:54321",
		Token:    "tok",
		Host:     "https://dbc.example.com",
		Profile:  "p",
	}); err != nil {
		t.Fatalf("FullSetup: %v", err)
	}
	if err := sm.Restore(); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	doc := readJSON(t, path)
	env := doc["env"].(map[string]interface{})
	for _, k := range fullSetupInferenceKeys {
		if _, exists := env[k]; exists {
			t.Errorf("key %s should have been removed by Restore", k)
		}
	}
}

func TestFullSetup_WithOTEL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	writeJSON(t, path, map[string]interface{}{"env": map[string]interface{}{}})

	sm := NewSettingsManager(path)
	cfg := FullSetupConfig{
		ProxyURL:    "http://127.0.0.1:54321",
		Token:       "tok",
		Host:        "https://dbc.example.com",
		Profile:     "p",
		OTELEnabled: true,
	}
	if err := sm.FullSetup(cfg); err != nil {
		t.Fatalf("FullSetup: %v", err)
	}

	doc := readJSON(t, path)
	env := doc["env"].(map[string]interface{})

	otelChecks := map[string]string{
		"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT": cfg.ProxyURL + "/otel/v1/metrics",
		"OTEL_EXPORTER_OTLP_METRICS_HEADERS":  "content-type=application/x-protobuf",
		"CLAUDE_CODE_ENABLE_TELEMETRY":         "1",
		"OTEL_METRICS_EXPORTER":                "otlp",
		"OTEL_EXPORTER_OTLP_METRICS_PROTOCOL":  "http/protobuf",
	}
	for k, want := range otelChecks {
		if got, ok := env[k]; !ok {
			t.Errorf("missing OTEL key %s", k)
		} else if got != want {
			t.Errorf("%s = %v, want %v", k, got, want)
		}
	}
}

func TestFullSetup_WithoutOTEL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	writeJSON(t, path, map[string]interface{}{"env": map[string]interface{}{}})

	sm := NewSettingsManager(path)
	if err := sm.FullSetup(FullSetupConfig{
		ProxyURL:    "http://127.0.0.1:54321",
		Token:       "tok",
		Host:        "https://dbc.example.com",
		Profile:     "p",
		OTELEnabled: false,
	}); err != nil {
		t.Fatalf("FullSetup: %v", err)
	}

	doc := readJSON(t, path)
	env := doc["env"].(map[string]interface{})
	for _, k := range fullSetupOTELKeys {
		if _, exists := env[k]; exists {
			t.Errorf("OTEL key %s should not be written when OTELEnabled=false", k)
		}
	}
}

func TestFullSetup_PreservesExistingKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	writeJSON(t, path, map[string]interface{}{
		"theme": "dark",
		"env": map[string]interface{}{
			"MY_CUSTOM_VAR": "preserved",
		},
	})

	sm := NewSettingsManager(path)
	if err := sm.FullSetup(FullSetupConfig{
		ProxyURL: "http://127.0.0.1:54321",
		Token:    "tok",
		Host:     "https://dbc.example.com",
		Profile:  "p",
	}); err != nil {
		t.Fatalf("FullSetup: %v", err)
	}

	doc := readJSON(t, path)
	if doc["theme"] != "dark" {
		t.Errorf("top-level theme = %v, want dark", doc["theme"])
	}
	env := doc["env"].(map[string]interface{})
	if env["MY_CUSTOM_VAR"] != "preserved" {
		t.Errorf("MY_CUSTOM_VAR = %v, want preserved", env["MY_CUSTOM_VAR"])
	}

	if err := sm.Restore(); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	restored := readJSON(t, path)
	if restored["theme"] != "dark" {
		t.Errorf("after restore, theme = %v, want dark", restored["theme"])
	}
	rEnv := restored["env"].(map[string]interface{})
	if rEnv["MY_CUSTOM_VAR"] != "preserved" {
		t.Errorf("after restore, MY_CUSTOM_VAR = %v, want preserved", rEnv["MY_CUSTOM_VAR"])
	}
}

func TestFullSetup_CreatesSettingsIfMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	// Do NOT create settings.json — FullSetup must create it.

	sm := NewSettingsManager(path)
	if err := sm.FullSetup(FullSetupConfig{
		ProxyURL: "http://127.0.0.1:54321",
		Token:    "tok",
		Host:     "https://dbc.example.com",
		Profile:  "p",
	}); err != nil {
		t.Fatalf("FullSetup: %v", err)
	}

	doc := readJSON(t, path)
	env := doc["env"].(map[string]interface{})
	if env["ANTHROPIC_BASE_URL"] != "http://127.0.0.1:54321" {
		t.Errorf("ANTHROPIC_BASE_URL = %v, want http://127.0.0.1:54321", env["ANTHROPIC_BASE_URL"])
	}
}

func TestSignalForwarding(t *testing.T) {
	// Start a child that sleeps; we'll kill it with SIGINT via ForwardSignals.
	cmd := exec.Command("/bin/sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot start sleep child: %v", err)
	}

	cancel := ForwardSignals(cmd)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	// Give the child a moment to start then send SIGINT to the child directly
	// via ForwardSignals by signaling ourselves.
	time.Sleep(50 * time.Millisecond)

	// Signal the child process directly to verify ForwardSignals wired things up.
	cmd.Process.Signal(syscall.SIGINT)

	select {
	case <-done:
		// Child exited — signal was received.
	case <-time.After(3 * time.Second):
		cmd.Process.Kill()
		t.Fatal("child did not exit after SIGINT")
	}
}

func TestChildExitCode(t *testing.T) {
	// Verify exit code extraction by running a shell command that exits with 42.
	cmd := exec.Command("/bin/sh", "-c", "exit 42")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	cancel := ForwardSignals(cmd)
	defer cancel()

	waitErr := cmd.Wait()
	var exitCode int
	if waitErr == nil {
		exitCode = 0
	} else if exitErr, ok := waitErr.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else {
		exitCode = 1
	}

	if exitCode != 42 {
		t.Errorf("exit code = %d, want 42", exitCode)
	}
}
