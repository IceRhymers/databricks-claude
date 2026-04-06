package filelock

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestFileLock_LockUnlock(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "test.lock")

	fl := New(lockPath)

	if err := fl.Lock(); err != nil {
		t.Fatalf("Lock() failed: %v", err)
	}

	// Lock file should exist
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Fatal("lock file was not created")
	}

	if err := fl.Unlock(); err != nil {
		t.Fatalf("Unlock() failed: %v", err)
	}

	// Unlock on already-unlocked should be safe
	if err := fl.Unlock(); err != nil {
		t.Fatalf("second Unlock() failed: %v", err)
	}
}

func TestFileLock_Concurrent(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "test.lock")

	var mu sync.Mutex
	var order []int

	var wg sync.WaitGroup
	wg.Add(2)

	// First goroutine acquires lock, holds it briefly
	go func() {
		defer wg.Done()
		fl := New(lockPath)
		if err := fl.Lock(); err != nil {
			t.Errorf("goroutine 1 Lock() failed: %v", err)
			return
		}
		mu.Lock()
		order = append(order, 1)
		mu.Unlock()

		time.Sleep(200 * time.Millisecond)

		if err := fl.Unlock(); err != nil {
			t.Errorf("goroutine 1 Unlock() failed: %v", err)
		}
	}()

	// Small delay so goroutine 1 acquires first
	time.Sleep(50 * time.Millisecond)

	// Second goroutine must wait for first to release
	go func() {
		defer wg.Done()
		fl := New(lockPath)
		if err := fl.Lock(); err != nil {
			t.Errorf("goroutine 2 Lock() failed: %v", err)
			return
		}
		mu.Lock()
		order = append(order, 2)
		mu.Unlock()

		if err := fl.Unlock(); err != nil {
			t.Errorf("goroutine 2 Unlock() failed: %v", err)
		}
	}()

	wg.Wait()

	if len(order) != 2 {
		t.Fatalf("expected 2 entries in order, got %d", len(order))
	}
	if order[0] != 1 || order[1] != 2 {
		t.Errorf("expected order [1, 2], got %v — second goroutine did not wait", order)
	}
}

func TestFileLock_GracefulDegradation(t *testing.T) {
	// Verify that Lock/Unlock on a valid path does not panic.
	// On Linux/macOS syscall.Flock is supported, so we verify the
	// normal path completes without error. The graceful-degradation
	// code path (unsupported flock) prints a warning but returns nil,
	// which is not easily triggered on these platforms without mocking
	// the syscall. This test ensures the fallback logic at minimum
	// does not cause a panic or leave the lock in a broken state.

	lockPath := filepath.Join(t.TempDir(), "degrade.lock")
	fl := New(lockPath)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Lock/Unlock panicked: %v", r)
		}
	}()

	if err := fl.Lock(); err != nil {
		t.Fatalf("Lock() failed: %v", err)
	}
	if err := fl.Unlock(); err != nil {
		t.Fatalf("Unlock() failed: %v", err)
	}

	// Lock again after unlock to verify reusability
	if err := fl.Lock(); err != nil {
		t.Fatalf("re-Lock() failed: %v", err)
	}
	if err := fl.Unlock(); err != nil {
		t.Fatalf("re-Unlock() failed: %v", err)
	}
}
