package refcount

import (
	"encoding/json"
	"fmt"
	"os"
	"syscall"
)

type counter struct {
	Count int `json:"count"`
}

// Acquire atomically increments the counter at path.
func Acquire(path string) error {
	return withLock(path, func(c *counter) { c.Count++ })
}

// Release atomically decrements the counter and returns the remaining count.
func Release(path string) (int, error) {
	var remaining int
	err := withLock(path, func(c *counter) {
		if c.Count > 0 {
			c.Count--
		}
		remaining = c.Count
	})
	return remaining, err
}

func withLock(path string, fn func(*counter)) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return fmt.Errorf("refcount: open %s: %w", path, err)
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("refcount: lock: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	var c counter
	if err := json.NewDecoder(f).Decode(&c); err != nil {
		c = counter{} // treat as 0 if empty/corrupt
	}

	fn(&c)

	if err := f.Truncate(0); err != nil {
		return err
	}
	if _, err := f.Seek(0, 0); err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(c)
}
