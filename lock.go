package main

import "github.com/IceRhymers/databricks-claude/pkg/filelock"

// FileLock provides exclusive file-based locking using syscall.Flock.
// Works on Linux and macOS.
type FileLock = filelock.FileLock

// NewFileLock creates a new FileLock for the given path.
func NewFileLock(path string) *FileLock {
	return filelock.New(path)
}
