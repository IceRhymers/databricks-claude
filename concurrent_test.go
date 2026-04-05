package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestConcurrentSessions_NoStomp verifies that when two sessions are active and
// the first one restores, settings.json retains the second session's proxyURL
// as ANTHROPIC_BASE_URL (smart handoff) instead of clobbering it with the
// original upstream.
func TestConcurrentSessions_NoStomp(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	registryPath := filepath.Join(dir, ".sessions.json")
	lockPath := filepath.Join(dir, ".settings.lock")

	// Seed settings.json with an original upstream value.
	writeJSON(t, settingsPath, map[string]interface{}{
		"env": map[string]interface{}{
			"ANTHROPIC_BASE_URL":   "https://real-upstream.example.com",
			"ANTHROPIC_AUTH_TOKEN": "original-token",
		},
	})

	// Create two SettingsManagers sharing the same lock and registry files.
	sm1 := &SettingsManager{
		settingsPath: settingsPath,
		origValues:   make(map[string]interface{}),
		lock:         NewFileLock(lockPath),
		registry:     NewSessionRegistry(registryPath),
	}
	sm2 := &SettingsManager{
		settingsPath: settingsPath,
		origValues:   make(map[string]interface{}),
		lock:         NewFileLock(lockPath),
		registry:     NewSessionRegistry(registryPath),
	}

	// Simulate two sessions starting up with different proxy URLs.
	proxy1 := "http://127.0.0.1:11111"
	proxy2 := "http://127.0.0.1:22222"

	if err := sm1.FullSetup(FullSetupConfig{
		ProxyURL: proxy1,
		Token:    "tok1",
		Host:     "https://dbc.example.com",
		Profile:  "p",
	}); err != nil {
		t.Fatalf("sm1.FullSetup: %v", err)
	}

	if err := sm2.FullSetup(FullSetupConfig{
		ProxyURL: proxy2,
		Token:    "tok2",
		Host:     "https://dbc.example.com",
		Profile:  "p",
	}); err != nil {
		t.Fatalf("sm2.FullSetup: %v", err)
	}

	// Verify both sessions are registered.
	live, err := NewSessionRegistry(registryPath).LiveSessions()
	if err != nil {
		t.Fatalf("LiveSessions: %v", err)
	}
	// Both registrations use os.Getpid() so they share the same PID in test.
	// That's fine — the key behavior is the handoff logic in Restore.
	if len(live) < 1 {
		t.Fatalf("expected at least 1 live session, got %d", len(live))
	}

	// Session 1 exits — calls Restore. Since session 2 is still "alive" (same PID
	// in test, so it's still registered), Restore should hand off to session 2's URL.
	// We need to manually unregister sm1's entry and keep sm2's entry to simulate
	// two different processes. Since both use os.Getpid(), we'll manipulate the
	// registry directly to simulate different PIDs.

	// Re-seed registry with two distinct fake PIDs.
	pid1 := os.Getpid()       // "session 1" — will be the one that restores
	pid2 := os.Getpid() + 100 // "session 2" — fake PID that we'll mark as alive

	regData := []Session{
		{PID: pid1, ProxyURL: proxy1},
		{PID: pid2, ProxyURL: proxy2},
	}
	data, _ := json.MarshalIndent(regData, "", "  ")
	os.WriteFile(registryPath, data, 0o644)

	// Override sm1's Restore to unregister pid1. Since Restore calls
	// registry.Unregister(os.Getpid()), and os.Getpid() == pid1, this works.
	// But pid2 is a fake PID that won't be alive. We need pid2 to appear alive.
	// Use a real PID (PID 1 is always alive on Linux).
	pid2 = 1 // init process — always alive
	regData = []Session{
		{PID: pid1, ProxyURL: proxy1},
		{PID: pid2, ProxyURL: proxy2},
	}
	data, _ = json.MarshalIndent(regData, "", "  ")
	os.WriteFile(registryPath, data, 0o644)

	// Session 1 restores.
	if err := sm1.Restore(); err != nil {
		t.Fatalf("sm1.Restore: %v", err)
	}

	// Verify settings.json has session 2's proxy URL (handoff), not the original.
	doc := readJSON(t, settingsPath)
	env := doc["env"].(map[string]interface{})
	if env["ANTHROPIC_BASE_URL"] != proxy2 {
		t.Errorf("ANTHROPIC_BASE_URL = %v, want %v (should hand off to surviving session)", env["ANTHROPIC_BASE_URL"], proxy2)
	}
}

// TestConcurrentSessions_LastSessionRestores verifies that when both sessions
// exit, the last one fully restores the original upstream values.
func TestConcurrentSessions_LastSessionRestores(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	registryPath := filepath.Join(dir, ".sessions.json")
	lockPath := filepath.Join(dir, ".settings.lock")

	originalUpstream := "https://real-upstream.example.com"

	writeJSON(t, settingsPath, map[string]interface{}{
		"env": map[string]interface{}{
			"ANTHROPIC_BASE_URL":   originalUpstream,
			"ANTHROPIC_AUTH_TOKEN": "original-token",
		},
	})

	// Both SettingsManagers read the file BEFORE either writes, so each
	// captures the real original upstream in origValues. In production each
	// process does this independently at startup.
	sm1 := &SettingsManager{
		settingsPath: settingsPath,
		origValues:   make(map[string]interface{}),
		lock:         NewFileLock(lockPath),
		registry:     NewSessionRegistry(registryPath),
	}
	sm2 := &SettingsManager{
		settingsPath: settingsPath,
		origValues:   make(map[string]interface{}),
		lock:         NewFileLock(lockPath),
		registry:     NewSessionRegistry(registryPath),
	}

	proxy1 := "http://127.0.0.1:11111"
	proxy2 := "http://127.0.0.1:22222"

	if err := sm1.FullSetup(FullSetupConfig{
		ProxyURL: proxy1, Token: "tok1",
		Host: "https://dbc.example.com", Profile: "p",
	}); err != nil {
		t.Fatalf("sm1.FullSetup: %v", err)
	}
	if err := sm2.FullSetup(FullSetupConfig{
		ProxyURL: proxy2, Token: "tok2",
		Host: "https://dbc.example.com", Profile: "p",
	}); err != nil {
		t.Fatalf("sm2.FullSetup: %v", err)
	}

	// Override sm2's origValues to simulate a real second process that captured
	// the original upstream at its own startup (before sm1 modified the file).
	// In production each process reads originals independently.
	sm2.origValues["ANTHROPIC_BASE_URL"] = originalUpstream
	sm2.origValues["ANTHROPIC_AUTH_TOKEN"] = "original-token"
	for _, k := range []string{
		"ANTHROPIC_DEFAULT_OPUS_MODEL", "ANTHROPIC_DEFAULT_SONNET_MODEL",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL", "ANTHROPIC_CUSTOM_HEADERS",
		"CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "DATABRICKS_HOST",
		"DATABRICKS_CONFIG_PROFILE",
	} {
		sm2.origValues[k] = nil
	}

	// Seed registry with two PIDs — use PID 1 (always alive) for session 2
	// so session 1's restore does a handoff.
	pid1 := os.Getpid()
	pid2 := 1 // init — always alive
	regData := []Session{
		{PID: pid1, ProxyURL: proxy1},
		{PID: pid2, ProxyURL: proxy2},
	}
	data, _ := json.MarshalIndent(regData, "", "  ")
	os.WriteFile(registryPath, data, 0o644)

	// Session 1 restores (handoff to session 2).
	if err := sm1.Restore(); err != nil {
		t.Fatalf("sm1.Restore: %v", err)
	}

	// Now simulate session 2 exiting: clear registry so it's the last one.
	os.WriteFile(registryPath, []byte("[]"), 0o644)

	// Session 2 restores — should be last session, full restore.
	if err := sm2.Restore(); err != nil {
		t.Fatalf("sm2.Restore: %v", err)
	}

	doc := readJSON(t, settingsPath)
	env := doc["env"].(map[string]interface{})

	if env["ANTHROPIC_BASE_URL"] != originalUpstream {
		t.Errorf("ANTHROPIC_BASE_URL = %v, want %v (last session should restore original)", env["ANTHROPIC_BASE_URL"], originalUpstream)
	}
	if env["ANTHROPIC_AUTH_TOKEN"] != "original-token" {
		t.Errorf("ANTHROPIC_AUTH_TOKEN = %v, want original-token", env["ANTHROPIC_AUTH_TOKEN"])
	}

	// FullSetup-only keys that didn't exist originally should be deleted.
	for _, k := range []string{"ANTHROPIC_DEFAULT_OPUS_MODEL", "DATABRICKS_HOST"} {
		if _, exists := env[k]; exists {
			t.Errorf("key %s should have been removed by last-session Restore", k)
		}
	}
}

// TestRegistry_StaleCleanup verifies that a dead PID is pruned from the
// registry when LiveSessions is called.
func TestRegistry_StaleCleanup(t *testing.T) {
	dir := t.TempDir()
	registryPath := filepath.Join(dir, ".sessions.json")

	reg := NewSessionRegistry(registryPath)

	livePID := os.Getpid()
	deadPID := 2147483 // Very unlikely to be a real PID on any system.

	if err := reg.Register(livePID, "http://127.0.0.1:11111"); err != nil {
		t.Fatalf("Register live: %v", err)
	}
	if err := reg.Register(deadPID, "http://127.0.0.1:99999"); err != nil {
		t.Fatalf("Register dead: %v", err)
	}

	// Verify both are in the file before pruning.
	allBefore, err := reg.readLocked()
	if err != nil {
		t.Fatalf("readLocked before prune: %v", err)
	}
	if len(allBefore) != 2 {
		t.Fatalf("expected 2 sessions before prune, got %d", len(allBefore))
	}

	// LiveSessions should prune the dead PID.
	live, err := reg.LiveSessions()
	if err != nil {
		t.Fatalf("LiveSessions: %v", err)
	}
	if len(live) != 1 {
		t.Fatalf("expected 1 live session after prune, got %d", len(live))
	}
	if live[0].PID != livePID {
		t.Errorf("expected live PID %d, got %d", livePID, live[0].PID)
	}

	// Verify the dead PID was persisted away from the file.
	allAfter, err := reg.readLocked()
	if err != nil {
		t.Fatalf("readLocked after prune: %v", err)
	}
	if len(allAfter) != 1 {
		t.Fatalf("expected 1 persisted session after prune, got %d", len(allAfter))
	}
	if allAfter[0].PID != livePID {
		t.Errorf("expected persisted PID %d, got %d", livePID, allAfter[0].PID)
	}
}
