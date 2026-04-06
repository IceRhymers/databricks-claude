package settings

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Locker abstracts file-based locking.
type Locker interface {
	Lock() error
	Unlock() error
}

// SessionTracker abstracts session registration for multi-process coordination.
type SessionTracker interface {
	Register(pid int, proxyURL string) error
	Unregister(pid int) error
	LiveSessions() ([]LiveSession, error)
	MostRecentLive() (*LiveSession, error)
}

// LiveSession represents a live proxy session for handoff logic.
type LiveSession struct {
	PID      int
	ProxyURL string
}

// KeyConfig parameterizes which env keys the SettingsManager manages.
type KeyConfig struct {
	ManagedKeys   []string // keys to save/restore during SaveAndOverwrite
	ProtectedKeys []string // keys skipped during Restore when persistent mode is on
}

// Manager reads, patches, and restores a JSON settings file.
type Manager struct {
	settingsPath       string
	origValues         map[string]interface{} // saved originals for restore
	protectedPersist   bool                   // when true, Restore skips ProtectedKeys
	mu                 sync.Mutex
	lock               Locker
	registry           SessionTracker
	keyConfig          KeyConfig
}

// NewManager creates a Manager for the given settings.json path.
func NewManager(path string, lock Locker, reg SessionTracker, kc KeyConfig) *Manager {
	return &Manager{
		settingsPath: path,
		origValues:   make(map[string]interface{}),
		lock:         lock,
		registry:     reg,
		keyConfig:    kc,
	}
}

// SetProtectedPersistent controls whether Restore skips ProtectedKeys. When true,
// Restore will not touch any keys in ProtectedKeys, leaving them in settings.json
// as-is.
func (sm *Manager) SetProtectedPersistent(v bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.protectedPersist = v
}

// ReadSettings reads and parses settings.json, returning the full document.
func (sm *Manager) ReadSettings() (map[string]interface{}, error) {
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

// WriteSettings atomically writes doc to settings.json using a temp file in
// the same directory followed by os.Rename.
func (sm *Manager) WriteSettings(doc map[string]interface{}) error {
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
	if err := os.Chmod(tmpName, 0o600); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("chmod temp file: %w", err)
	}
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

// GetEnvBlock returns the "env" sub-map from the document, creating it if needed.
func GetEnvBlock(doc map[string]interface{}) map[string]interface{} {
	if env, ok := doc["env"]; ok {
		if envMap, ok := env.(map[string]interface{}); ok {
			return envMap
		}
	}
	envMap := make(map[string]interface{})
	doc["env"] = envMap
	return envMap
}

// SaveOriginals saves the current values of the given keys from env into
// origValues. Absent keys are stored as nil sentinel.
func (sm *Manager) SaveOriginals(env map[string]interface{}, keys []string) {
	for _, k := range keys {
		if v, exists := env[k]; exists {
			sm.origValues[k] = v
		} else {
			sm.origValues[k] = nil
		}
	}
}

// SaveOriginalsIfPresent saves only keys that exist in env (absent keys are
// left unset in origValues, so Restore deletes them).
func (sm *Manager) SaveOriginalsIfPresent(env map[string]interface{}, keys []string) {
	for _, k := range keys {
		if v, exists := env[k]; exists {
			sm.origValues[k] = v
		}
	}
}

// ClearStaleLocalhost removes stale localhost values from origValues.
func (sm *Manager) ClearStaleLocalhost(keys []string) {
	for _, k := range keys {
		if orig, ok := sm.origValues[k]; ok {
			if s, ok := orig.(string); ok && strings.HasPrefix(s, "http://127.0.0.1") {
				delete(sm.origValues, k)
			}
		}
	}
}

// SaveAndOverwrite reads settings.json, saves original values for keys we will
// overwrite, then writes proxy values. Returns an error if settings.json does
// not exist.
func (sm *Manager) SaveAndOverwrite(proxyURL string, writeValues func(env map[string]interface{}, proxyURL string)) error {
	if err := sm.lock.Lock(); err != nil {
		return fmt.Errorf("acquire file lock: %w", err)
	}
	defer sm.lock.Unlock()

	sm.mu.Lock()
	defer sm.mu.Unlock()

	doc, err := sm.ReadSettings()
	if err != nil {
		return err
	}

	env := GetEnvBlock(doc)

	// Save originals for all managed keys.
	allKeys := append(sm.keyConfig.ManagedKeys, sm.keyConfig.ProtectedKeys...)
	sm.SaveOriginalsIfPresent(env, allKeys)

	// Clear stale localhost values from originals.
	sm.ClearStaleLocalhost(allKeys)

	// Apply the caller-provided write logic.
	writeValues(env, proxyURL)

	if err := sm.WriteSettings(doc); err != nil {
		return err
	}

	// Register this session.
	if err := sm.registry.Register(os.Getpid(), proxyURL); err != nil {
		log.Printf("warning: failed to register session: %v", err)
	}

	return nil
}

// Restore writes the original values back to settings.json. Keys that did not
// exist in the original are removed. When protectedPersist is true, keys in
// ProtectedKeys are skipped entirely (left as-is in settings.json).
//
// Smart handoff: if other live sessions exist, Restore hands off the first
// ManagedKey to the most recent survivor instead of restoring the original
// upstream. Only the last session standing restores originals.
func (sm *Manager) Restore() error {
	if err := sm.lock.Lock(); err != nil {
		return fmt.Errorf("acquire file lock: %w", err)
	}
	defer sm.lock.Unlock()

	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Unregister ourselves from the session registry.
	if err := sm.registry.Unregister(os.Getpid()); err != nil {
		log.Printf("warning: failed to unregister session: %v", err)
	}

	// Check for other live sessions.
	live, err := sm.registry.LiveSessions()
	if err != nil {
		log.Printf("warning: failed to read live sessions: %v", err)
		live = nil
	}

	doc, err := sm.ReadSettings()
	if err != nil {
		return err
	}

	env := GetEnvBlock(doc)

	if len(live) > 0 {
		// Other sessions still alive — hand off to the most recent survivor.
		survivor, sErr := sm.registry.MostRecentLive()
		if sErr == nil && survivor != nil && len(sm.keyConfig.ManagedKeys) > 0 {
			env[sm.keyConfig.ManagedKeys[0]] = survivor.ProxyURL
			log.Printf("handing off %s to surviving session (PID %d, %s)",
				sm.keyConfig.ManagedKeys[0], survivor.PID, survivor.ProxyURL)
		}
	} else {
		// Last session — restore original values.
		skipProtected := map[string]bool{}
		if sm.protectedPersist {
			for _, k := range sm.keyConfig.ProtectedKeys {
				skipProtected[k] = true
			}
		}

		allKeys := append(sm.keyConfig.ManagedKeys, sm.keyConfig.ProtectedKeys...)
		for _, k := range allKeys {
			if skipProtected[k] {
				continue
			}
			if orig, had := sm.origValues[k]; had {
				if orig == nil {
					delete(env, k)
				} else {
					env[k] = orig
				}
			} else {
				delete(env, k)
			}
		}
	}

	return sm.WriteSettings(doc)
}

// RestoreKeys restores a specific set of keys from origValues into env.
// When protectedPersist is true, keys in protectedSkip are skipped.
func (sm *Manager) RestoreKeys(env map[string]interface{}, keys []string, protectedSkip map[string]bool) {
	for _, k := range keys {
		if protectedSkip[k] {
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

// OrigValues returns the saved original values map (for advanced usage).
func (sm *Manager) OrigValues() map[string]interface{} {
	return sm.origValues
}

// SetOrigValue sets a specific original value (for advanced usage / testing).
func (sm *Manager) SetOrigValue(key string, value interface{}) {
	sm.origValues[key] = value
}

// Lock exposes the underlying locker.
func (sm *Manager) Lock() Locker {
	return sm.lock
}

// Registry exposes the underlying session tracker.
func (sm *Manager) Registry() SessionTracker {
	return sm.registry
}

// SettingsPath returns the path to the settings file.
func (sm *Manager) SettingsPath() string {
	return sm.settingsPath
}
