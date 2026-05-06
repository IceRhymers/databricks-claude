// Package cli provides helpers for locating the Databricks CLI binary.
package cli

import (
	"os"
	"os/exec"
	"path/filepath"
)

// FallbackCLIDirs lists install locations to probe when the CLI binary is not
// on PATH. Order matters: most-likely first. GUI-launched subprocesses (e.g.
// Claude Desktop invoking the credential helper) inherit launchd's minimal
// PATH (/usr/bin:/bin:/usr/sbin:/sbin), which omits all of these.
var FallbackCLIDirs = []string{
	"/usr/local/bin",
	"/opt/homebrew/bin",
	"/opt/homebrew/sbin",
	".local/bin", // resolved against $HOME
	"go/bin",     // resolved against $HOME
	"bin",        // resolved against $HOME
}

// ResolveDatabricksCLI returns an executable path for cmdName.
// Lookup order:
//  1. Absolute or path-qualified cmdName → returned unchanged (back-compat for tests).
//  2. $DATABRICKS_CLI env override, if set and executable.
//  3. exec.LookPath(cmdName), which honors the inherited PATH.
//  4. A scan of common install dirs (FallbackCLIDirs).
//
// If none match, cmdName is returned unchanged so the eventual exec error
// surfaces with its original message.
func ResolveDatabricksCLI(cmdName string) string {
	if cmdName == "" {
		cmdName = "databricks"
	}
	if filepath.IsAbs(cmdName) || filepath.Base(cmdName) != cmdName {
		return cmdName
	}
	if override := os.Getenv("DATABRICKS_CLI"); override != "" {
		if IsExecutableFile(override) {
			return override
		}
	}
	if p, err := exec.LookPath(cmdName); err == nil {
		return p
	}
	home, _ := os.UserHomeDir()
	for _, dir := range FallbackCLIDirs {
		if !filepath.IsAbs(dir) {
			if home == "" {
				continue
			}
			dir = filepath.Join(home, dir)
		}
		candidate := filepath.Join(dir, cmdName)
		if IsExecutableFile(candidate) {
			return candidate
		}
	}
	return cmdName
}

// IsExecutableFile reports whether path refers to an executable file.
func IsExecutableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode()&0o111 != 0
}
