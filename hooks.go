package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/IceRhymers/databricks-claude/pkg/refcount"
)

// headlessEnsure checks whether the proxy is healthy on the given port.
// If not, it starts a detached headless proxy and polls until ready (max 10s).
// Called by the SessionStart hook via: databricks-claude --headless-ensure
func headlessEnsure(port int) {
	if os.Getenv("DATABRICKS_CLAUDE_MANAGED") == "1" {
		log.Printf("databricks-claude: --headless-ensure: skipped (managed session)")
		return
	}

	// Acquire refcount FIRST so every ensure/release pair is symmetric.
	refcountPath := refcountPathForPort(port)
	if err := refcount.Acquire(refcountPath); err != nil {
		log.Printf("databricks-claude: --headless-ensure: refcount acquire warning: %v", err)
	}

	if isProxyHealthy(port) {
		return // already running, refcount incremented
	}

	self, err := os.Executable()
	if err != nil {
		refcount.Release(refcountPath) // undo acquire on failure
		log.Fatalf("databricks-claude: --headless-ensure: cannot find self: %v", err)
	}

	cmd := exec.Command(self, "--headless", fmt.Sprintf("--port=%d", port))
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		refcount.Release(refcountPath) // undo acquire on failure
		log.Fatalf("databricks-claude: --headless-ensure: failed to start proxy: %v", err)
	}
	if err := cmd.Process.Release(); err != nil {
		log.Printf("databricks-claude: --headless-ensure: release warning: %v", err)
	}

	// Poll until healthy or timeout.
	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		if isProxyHealthy(port) {
			return
		}
	}
	refcount.Release(refcountPath) // undo acquire on failure
	log.Fatalf("databricks-claude: --headless-ensure: proxy did not become healthy within 10s")
}

// headlessRelease calls POST /shutdown on the proxy to decrement the refcount.
// Called by the Stop hook via: databricks-claude --headless-release
// Errors are logged but not fatal — proxy may already be stopped.
func headlessRelease(port int) {
	if os.Getenv("DATABRICKS_CLAUDE_MANAGED") == "1" {
		log.Printf("databricks-claude: --headless-release: skipped (managed session)")
		return
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Post(fmt.Sprintf("http://127.0.0.1:%d/shutdown", port), "application/json", nil)
	if err != nil {
		log.Printf("databricks-claude: --headless-release: %v (proxy may already be stopped)", err)
		return
	}
	resp.Body.Close()
}

// isProxyHealthy returns true if the proxy on port responds to GET /health.
func isProxyHealthy(port int) bool {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// installHooks merges the databricks-claude SessionStart and Stop hooks into
// ~/.claude/settings.json. Idempotent — safe to run after upgrades.
func installHooks(settingsPath string) error {
	doc, err := readSettingsDoc(settingsPath)
	if err != nil {
		// File may not exist yet — start with an empty document.
		doc = map[string]interface{}{}
	}

	hooks, _ := doc["hooks"].(map[string]interface{})
	if hooks == nil {
		hooks = map[string]interface{}{}
	}

	// Remove any existing databricks-claude hooks before re-adding (idempotent).
	removeDBXHooks(hooks)

	// SessionStart hook — starts proxy if not already running.
	sessionStart, _ := hooks["SessionStart"].([]interface{})
	sessionStart = append(sessionStart, map[string]interface{}{
		"matcher": "startup",
		"hooks": []interface{}{
			map[string]interface{}{
				"type":    "command",
				"command": "databricks-claude --headless-ensure",
				"timeout": 15,
			},
		},
	})
	hooks["SessionStart"] = sessionStart

	// Stop hook — decrements proxy refcount; proxy exits when last session ends.
	stop, _ := hooks["Stop"].([]interface{})
	stop = append(stop, map[string]interface{}{
		"hooks": []interface{}{
			map[string]interface{}{
				"type":    "command",
				"command": "databricks-claude --headless-release",
				"timeout": 5,
			},
		},
	})
	hooks["Stop"] = stop

	doc["hooks"] = hooks
	return writeSettingsDoc(settingsPath, doc)
}

// uninstallHooks removes the databricks-claude hooks from ~/.claude/settings.json.
func uninstallHooks(settingsPath string) error {
	doc, err := readSettingsDoc(settingsPath)
	if err != nil {
		return nil // nothing to remove
	}

	hooks, _ := doc["hooks"].(map[string]interface{})
	if hooks == nil {
		return nil
	}

	removeDBXHooks(hooks)

	// Clean up empty hook event keys.
	for k, v := range hooks {
		if arr, ok := v.([]interface{}); ok && len(arr) == 0 {
			delete(hooks, k)
		}
	}
	if len(hooks) == 0 {
		delete(doc, "hooks")
	} else {
		doc["hooks"] = hooks
	}

	return writeSettingsDoc(settingsPath, doc)
}

// removeDBXHooks removes any hook entries whose command contains "databricks-claude --headless".
func removeDBXHooks(hooks map[string]interface{}) {
	for event, val := range hooks {
		arr, _ := val.([]interface{})
		filtered := arr[:0]
		for _, entry := range arr {
			if !isDBXHookEntry(entry) {
				filtered = append(filtered, entry)
			}
		}
		hooks[event] = filtered
	}
}

// isDBXHookEntry returns true if any nested hook command references databricks-claude --headless.
func isDBXHookEntry(entry interface{}) bool {
	m, ok := entry.(map[string]interface{})
	if !ok {
		return false
	}
	inner, _ := m["hooks"].([]interface{})
	for _, h := range inner {
		hm, _ := h.(map[string]interface{})
		if cmd, _ := hm["command"].(string); len(cmd) > 0 {
			if len(cmd) >= len("databricks-claude --headless") &&
				cmd[:len("databricks-claude --headless")] == "databricks-claude --headless" {
				return true
			}
		}
	}
	return false
}

// writeSettingsDoc writes a settings document back to disk as indented JSON.
func writeSettingsDoc(path string, doc map[string]interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating settings dir: %w", err)
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling settings: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}
