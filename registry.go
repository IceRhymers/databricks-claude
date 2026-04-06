package main

import "github.com/IceRhymers/databricks-claude/pkg/registry"

// Session represents a live Claude proxy session.
type Session = registry.Session

// SessionRegistry tracks live sessions via a JSON file.
type SessionRegistry = registry.SessionRegistry

// NewSessionRegistry creates a registry backed by the given JSON file path.
func NewSessionRegistry(path string) *SessionRegistry {
	return registry.New(path)
}

// isProcessAlive is no longer needed here — it lives in pkg/registry.
