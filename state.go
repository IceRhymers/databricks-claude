package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// persistentState holds cross-session state persisted to ~/.claude/.databricks-claude.json.
// This is separate from the persistent config (profile, etc.) — it tracks runtime
// state that survives across invocations.
type persistentState struct {
	Profile string `json:"profile,omitempty"`
	Port    int    `json:"port,omitempty"`
}

const defaultPort = 49153

// statePath returns the path to the persistent state file.
func statePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".claude", ".databricks-claude.json"), nil
}

// loadState reads the persistent state from disk. Returns a zero-value state
// if the file does not exist.
func loadState() (*persistentState, error) {
	p, err := statePath()
	if err != nil {
		return &persistentState{}, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &persistentState{}, nil
		}
		return nil, err
	}
	var s persistentState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// saveState writes the persistent state to disk atomically.
func saveState(s *persistentState) error {
	p, err := statePath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// resolvePort returns the port to use, following the resolution chain:
// 1. --port flag (portFlag > 0)
// 2. Saved state value (non-zero)
// 3. Default 49153
func resolvePort(portFlag int, state *persistentState) int {
	if portFlag > 0 {
		return portFlag
	}
	if state != nil && state.Port > 0 {
		return state.Port
	}
	return defaultPort
}
