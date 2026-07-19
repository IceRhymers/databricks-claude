package proxy

import "testing"

func TestSecurityChecks_Root(t *testing.T) {
	orig := getuid
	t.Cleanup(func() { getuid = orig })

	getuid = func() int { return 0 }

	warnings := SecurityChecks()

	found := false
	for _, w := range warnings {
		if w == "WARNING: running as root is not recommended for proxy operation" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected root warning, got %v", warnings)
	}
}

func TestSecurityChecks_NonRoot(t *testing.T) {
	orig := getuid
	t.Cleanup(func() { getuid = orig })

	getuid = func() int { return 1000 }

	warnings := SecurityChecks()

	for _, w := range warnings {
		if w == "WARNING: running as root is not recommended for proxy operation" {
			t.Errorf("unexpected root warning for uid 1000, got %v", warnings)
		}
	}
}
