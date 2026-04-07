// Package refcount provides cross-process session counting backed by a JSON file.
// on Windows).
package refcount

// Acquire atomically increments the session counter at path.
func Acquire(path string) error {
	return withLock(path, func(c *counter) { c.Count++ })
}

// Release atomically decrements the session counter and returns the remaining count.
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
