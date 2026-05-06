package authcheck

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/IceRhymers/databricks-claude/pkg/cli"
)

// Overridable for testing.
var execCommand = exec.Command
var execCommandContext = exec.CommandContext

// IsAuthenticated returns true if a valid token can be fetched for the given
// Databricks profile without triggering an interactive login.
// cmdName is the Databricks CLI binary name or path; pass "" to use the default ("databricks").
func IsAuthenticated(profile, cmdName string) bool {
	resolved := cli.ResolveDatabricksCLI(cmdName)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := execCommandContext(ctx, resolved, "auth", "token", "--profile", profile).Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "access_token")
}

// EnsureAuthenticated verifies the user has a valid token for the profile.
// If not authenticated, it runs "<cmdName> auth login --profile <profile>"
// interactively (attaches stdin/stdout/stderr so the browser OAuth flow works).
// Returns nil if authentication succeeds, error if it fails even after login.
// cmdName is the Databricks CLI binary name or path; pass "" to use the default ("databricks").
func EnsureAuthenticated(profile, cmdName string) error {
	if IsAuthenticated(profile, cmdName) {
		return nil
	}
	resolved := cli.ResolveDatabricksCLI(cmdName)
	fmt.Fprintf(os.Stderr, "databricks: not authenticated for profile %q, opening browser login...\n", profile)
	cmd := execCommand(resolved, "auth", "login", "--profile", profile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("databricks auth login failed: %w", err)
	}
	if !IsAuthenticated(profile, cmdName) {
		return fmt.Errorf("still not authenticated for profile %q after login attempt", profile)
	}
	return nil
}
