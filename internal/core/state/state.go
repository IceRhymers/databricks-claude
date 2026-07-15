// Package state provides generic JSON state persistence and port resolution
// shared across databricks-claude, databricks-codex, and databricks-opencode.
package state

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

// Save atomically writes v as indented JSON to path.
// Parent directories are created as needed.
func Save[T any](path string, v T) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".state-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}

// Load reads and unmarshals JSON from path into a value of type T.
// Returns the zero value if the file does not exist or cannot be parsed.
func Load[T any](path string) (T, error) {
	var zero T
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return zero, nil
		}
		return zero, err
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		log.Printf("state: invalid state file %s, ignoring: %v", path, err)
		return zero, nil
	}
	return v, nil
}

// ResolvePort returns the port to use, following the resolution chain:
//  1. flagPort (if > 0)
//  2. savedPort (if > 0)
//  3. defaultPort
func ResolvePort(flagPort, savedPort, defaultPort int) int {
	if flagPort > 0 {
		return flagPort
	}
	if savedPort > 0 {
		return savedPort
	}
	return defaultPort
}
