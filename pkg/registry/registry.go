package registry

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"syscall"
	"time"
)

// Session represents a live proxy session.
type Session struct {
	PID       int       `json:"pid"`
	ProxyURL  string    `json:"proxy_url"`
	StartedAt time.Time `json:"started_at"`
}

// SessionRegistry tracks live sessions via a JSON file.
type SessionRegistry struct {
	path string
	mu   sync.Mutex
}

// New creates a registry backed by the given JSON file path.
func New(path string) *SessionRegistry {
	return &SessionRegistry{path: path}
}

// Register adds a session entry to the registry file atomically.
func (r *SessionRegistry) Register(pid int, proxyURL string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	sessions, err := r.readLocked()
	if err != nil {
		return err
	}

	sessions = append(sessions, Session{
		PID:       pid,
		ProxyURL:  proxyURL,
		StartedAt: time.Now(),
	})

	return r.writeLocked(sessions)
}

// Unregister removes a session by PID atomically.
func (r *SessionRegistry) Unregister(pid int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	sessions, err := r.readLocked()
	if err != nil {
		return err
	}

	filtered := sessions[:0]
	for _, s := range sessions {
		if s.PID != pid {
			filtered = append(filtered, s)
		}
	}

	return r.writeLocked(filtered)
}

// LiveSessions reads the registry, prunes stale PIDs (where the process no
// longer exists), and returns the remaining live sessions.
func (r *SessionRegistry) LiveSessions() ([]Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	sessions, err := r.readLocked()
	if err != nil {
		return nil, err
	}

	var live []Session
	for _, s := range sessions {
		if isProcessAlive(s.PID) {
			live = append(live, s)
		}
	}

	// Persist the pruned list so stale entries don't accumulate.
	if len(live) != len(sessions) {
		if err := r.writeLocked(live); err != nil {
			return nil, err
		}
	}

	return live, nil
}

// MostRecentLive returns the live session with the latest StartedAt, or nil if
// there are no live sessions.
func (r *SessionRegistry) MostRecentLive() (*Session, error) {
	live, err := r.LiveSessions()
	if err != nil {
		return nil, err
	}
	if len(live) == 0 {
		return nil, nil
	}

	sort.Slice(live, func(i, j int) bool {
		return live[i].StartedAt.After(live[j].StartedAt)
	})

	result := live[0]
	return &result, nil
}

// ReadLocked reads sessions from the JSON file. Exported for testing.
func (r *SessionRegistry) ReadLocked() ([]Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.readLocked()
}

// readLocked reads sessions from the JSON file. Caller must hold r.mu.
func (r *SessionRegistry) readLocked() ([]Session, error) {
	data, err := os.ReadFile(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}

	var sessions []Session
	if err := json.Unmarshal(data, &sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

// writeLocked writes sessions to the JSON file atomically via temp+rename.
// Caller must hold r.mu.
func (r *SessionRegistry) writeLocked(sessions []Session) error {
	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".sessions-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}

	return os.Rename(tmpName, r.path)
}

// isProcessAlive checks whether a process with the given PID exists.
// EPERM means the process exists but we lack permission to signal it — alive.
// ESRCH means no such process — dead.
func isProcessAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
