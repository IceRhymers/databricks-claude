package main

import (
	"fmt"
	"os"
	"syscall"
)

// FileLock provides exclusive file-based locking using syscall.Flock.
// Works on Linux and macOS.
type FileLock struct {
	path string
	file *os.File
}

// NewFileLock creates a new FileLock for the given path.
func NewFileLock(path string) *FileLock {
	return &FileLock{path: path}
}

// Lock acquires an exclusive lock on the file. If syscall.Flock is not
// supported, it warns but returns nil (graceful degradation).
func (fl *FileLock) Lock() error {
	f, err := os.OpenFile(fl.path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("failed to open lock file %s: %w", fl.path, err)
	}
	fl.file = f

	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
	if err != nil {
		// Graceful degradation: warn but don't fail
		fmt.Fprintf(os.Stderr, "warning: flock not supported on this system, proceeding without lock: %v\n", err)
		return nil
	}
	return nil
}

// Unlock releases the lock and closes the file descriptor.
func (fl *FileLock) Unlock() error {
	if fl.file == nil {
		return nil
	}
	defer func() {
		fl.file.Close()
		fl.file = nil
	}()

	err := syscall.Flock(int(fl.file.Fd()), syscall.LOCK_UN)
	if err != nil {
		return fmt.Errorf("failed to unlock %s: %w", fl.path, err)
	}
	return nil
}
