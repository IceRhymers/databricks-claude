// Package headless provides the headless-ensure logic shared across
// databricks-claude, databricks-codex, and databricks-opencode.
package headless

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/IceRhymers/databricks-claude/pkg/health"
	"github.com/IceRhymers/databricks-claude/pkg/refcount"
)

// Config holds the parameters for Ensure.
type Config struct {
	// Port is the proxy port to check/start.
	Port int

	// Scheme is "http" or "https" for health checks. Typically "http" for
	// the headless-ensure path since TLS is not yet negotiated.
	Scheme string

	// ManagedEnvVar is the environment variable that, when set to "1",
	// causes Ensure to skip (e.g. "DATABRICKS_CLAUDE_MANAGED").
	ManagedEnvVar string

	// BinaryPath is the path to the binary to launch in headless mode.
	// If empty, os.Executable() is used.
	BinaryPath string

	// LogPrefix is the string used in log messages (e.g. "databricks-claude").
	LogPrefix string

	// RefcountPath, when non-empty, causes Ensure to acquire/release the
	// refcount around the operation (claude behavior). When empty, refcount
	// is skipped (codex/opencode behavior).
	RefcountPath string
}

// Ensure checks whether the proxy is healthy on the given port.
// If not, it starts a detached headless proxy and polls until ready (max 10s).
func Ensure(cfg Config) {
	if cfg.ManagedEnvVar != "" && os.Getenv(cfg.ManagedEnvVar) == "1" {
		log.Printf("%s: --headless-ensure: skipped (managed session)", cfg.LogPrefix)
		return
	}

	// Acquire refcount FIRST so every ensure/release pair is symmetric.
	if cfg.RefcountPath != "" {
		if err := refcount.Acquire(cfg.RefcountPath); err != nil {
			log.Printf("%s: --headless-ensure: refcount acquire warning: %v", cfg.LogPrefix, err)
		}
	}

	if health.IsProxyHealthy(cfg.Port) {
		return // already running, refcount incremented
	}

	self := cfg.BinaryPath
	if self == "" {
		var err error
		self, err = os.Executable()
		if err != nil {
			if cfg.RefcountPath != "" {
				refcount.Release(cfg.RefcountPath) // undo acquire on failure
			}
			log.Fatalf("%s: --headless-ensure: cannot find self: %v", cfg.LogPrefix, err)
		}
	}

	cmd := exec.Command(self, "--headless", fmt.Sprintf("--port=%d", cfg.Port))
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		if cfg.RefcountPath != "" {
			refcount.Release(cfg.RefcountPath) // undo acquire on failure
		}
		log.Fatalf("%s: --headless-ensure: failed to start proxy: %v", cfg.LogPrefix, err)
	}
	if err := cmd.Process.Release(); err != nil {
		log.Printf("%s: --headless-ensure: release warning: %v", cfg.LogPrefix, err)
	}

	// Poll until healthy or timeout.
	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		if health.IsProxyHealthy(cfg.Port) {
			return
		}
	}
	if cfg.RefcountPath != "" {
		refcount.Release(cfg.RefcountPath) // undo acquire on failure
	}
	log.Fatalf("%s: --headless-ensure: proxy did not become healthy within 10s", cfg.LogPrefix)
}
