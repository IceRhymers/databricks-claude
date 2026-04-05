package main

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func tempRegistryPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "sessions.json")
}

func TestSessionRegistry_RegisterUnregister(t *testing.T) {
	path := tempRegistryPath(t)
	reg := NewSessionRegistry(path)

	pid := os.Getpid()

	if err := reg.Register(pid, "http://localhost:8080"); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := reg.Register(pid+999999, "http://localhost:8081"); err != nil {
		t.Fatalf("Register second: %v", err)
	}

	// Unregister the second entry.
	if err := reg.Unregister(pid + 999999); err != nil {
		t.Fatalf("Unregister: %v", err)
	}

	// Read back — only the first entry should remain.
	sessions, err := reg.readLocked()
	if err != nil {
		t.Fatalf("readLocked: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].PID != pid {
		t.Fatalf("expected PID %d, got %d", pid, sessions[0].PID)
	}
	if sessions[0].ProxyURL != "http://localhost:8080" {
		t.Fatalf("expected proxy URL http://localhost:8080, got %s", sessions[0].ProxyURL)
	}
}

func TestSessionRegistry_LiveSessions_PrunesStale(t *testing.T) {
	path := tempRegistryPath(t)
	reg := NewSessionRegistry(path)

	livePID := os.Getpid()
	deadPID := 2147483 // Very unlikely to be a real PID.

	if err := reg.Register(livePID, "http://localhost:8080"); err != nil {
		t.Fatalf("Register live: %v", err)
	}
	if err := reg.Register(deadPID, "http://localhost:9999"); err != nil {
		t.Fatalf("Register dead: %v", err)
	}

	live, err := reg.LiveSessions()
	if err != nil {
		t.Fatalf("LiveSessions: %v", err)
	}

	if len(live) != 1 {
		t.Fatalf("expected 1 live session after pruning, got %d", len(live))
	}
	if live[0].PID != livePID {
		t.Fatalf("expected live PID %d, got %d", livePID, live[0].PID)
	}

	// Verify the stale entry was persisted away.
	all, err := reg.readLocked()
	if err != nil {
		t.Fatalf("readLocked after prune: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 persisted session after prune, got %d", len(all))
	}
}

func TestSessionRegistry_MostRecentLive(t *testing.T) {
	path := tempRegistryPath(t)
	reg := NewSessionRegistry(path)

	pid := os.Getpid()

	// Register twice with the same live PID but different proxy URLs.
	if err := reg.Register(pid, "http://localhost:8080"); err != nil {
		t.Fatalf("Register first: %v", err)
	}
	if err := reg.Register(pid, "http://localhost:8081"); err != nil {
		t.Fatalf("Register second: %v", err)
	}

	most, err := reg.MostRecentLive()
	if err != nil {
		t.Fatalf("MostRecentLive: %v", err)
	}
	if most == nil {
		t.Fatal("expected a session, got nil")
	}
	// The second registration should be more recent.
	if most.ProxyURL != "http://localhost:8081" {
		t.Fatalf("expected most recent proxy URL http://localhost:8081, got %s", most.ProxyURL)
	}

	// Empty registry should return nil.
	emptyReg := NewSessionRegistry(tempRegistryPath(t))
	most, err = emptyReg.MostRecentLive()
	if err != nil {
		t.Fatalf("MostRecentLive empty: %v", err)
	}
	if most != nil {
		t.Fatalf("expected nil for empty registry, got %+v", most)
	}
}

func TestSessionRegistry_ConcurrentAccess(t *testing.T) {
	path := tempRegistryPath(t)
	reg := NewSessionRegistry(path)

	pid := os.Getpid()
	const goroutines = 20

	var wg sync.WaitGroup
	errs := make(chan error, goroutines*2)

	// Half register, half unregister concurrently.
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			fakePID := pid + i + 1
			if err := reg.Register(fakePID, "http://localhost:8080"); err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()

	// Now unregister half of them concurrently.
	for i := 0; i < goroutines/2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			fakePID := pid + i + 1
			if err := reg.Unregister(fakePID); err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("concurrent error: %v", err)
	}

	// Verify the file is valid JSON and has the expected count.
	sessions, err := reg.readLocked()
	if err != nil {
		t.Fatalf("readLocked after concurrent: %v", err)
	}
	if len(sessions) != goroutines/2 {
		t.Fatalf("expected %d sessions, got %d", goroutines/2, len(sessions))
	}
}
