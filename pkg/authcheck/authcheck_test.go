package authcheck

import (
	"context"
	"fmt"
	"os/exec"
	"testing"
)

// fakeCommandContext returns a *exec.Cmd that does nothing successfully
// and whose Output() returns the given data.
func fakeCommandContext(output string, fail bool) func(ctx context.Context, name string, args ...string) *exec.Cmd {
	return func(ctx context.Context, name string, args ...string) *exec.Cmd {
		if fail {
			return exec.CommandContext(ctx, "false")
		}
		return exec.CommandContext(ctx, "echo", output)
	}
}

func fakeCommand(output string, fail bool) func(name string, args ...string) *exec.Cmd {
	return func(name string, args ...string) *exec.Cmd {
		if fail {
			return exec.Command("false")
		}
		return exec.Command("echo", output)
	}
}

func TestIsAuthenticated_Success(t *testing.T) {
	origCtx := execCommandContext
	defer func() { execCommandContext = origCtx }()

	execCommandContext = fakeCommandContext(`{"access_token":"dapi-xxx","token_type":"Bearer"}`, false)

	if !IsAuthenticated("DEFAULT") {
		t.Error("expected IsAuthenticated to return true when access_token is present")
	}
}

func TestIsAuthenticated_NoToken(t *testing.T) {
	origCtx := execCommandContext
	defer func() { execCommandContext = origCtx }()

	execCommandContext = fakeCommandContext(`{"error":"no token"}`, false)

	if IsAuthenticated("DEFAULT") {
		t.Error("expected IsAuthenticated to return false when access_token is absent")
	}
}

func TestIsAuthenticated_CommandFails(t *testing.T) {
	origCtx := execCommandContext
	defer func() { execCommandContext = origCtx }()

	execCommandContext = fakeCommandContext("", true)

	if IsAuthenticated("DEFAULT") {
		t.Error("expected IsAuthenticated to return false when command fails")
	}
}

func TestEnsureAuthenticated_AlreadyAuthed(t *testing.T) {
	origCtx := execCommandContext
	defer func() { execCommandContext = origCtx }()

	execCommandContext = fakeCommandContext(`{"access_token":"dapi-xxx"}`, false)

	if err := EnsureAuthenticated("DEFAULT"); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestEnsureAuthenticated_LoginFails(t *testing.T) {
	origCtx := execCommandContext
	origCmd := execCommand
	defer func() {
		execCommandContext = origCtx
		execCommand = origCmd
	}()

	// IsAuthenticated returns false
	execCommandContext = fakeCommandContext("", true)
	// login command fails
	execCommand = fakeCommand("", true)

	err := EnsureAuthenticated("DEFAULT")
	if err == nil {
		t.Error("expected error when login fails")
	}
}

func TestEnsureAuthenticated_LoginSucceeds(t *testing.T) {
	origCtx := execCommandContext
	origCmd := execCommand
	defer func() {
		execCommandContext = origCtx
		execCommand = origCmd
	}()

	callCount := 0
	// First call: not authenticated. Second call (after login): authenticated.
	execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		callCount++
		if callCount == 1 {
			return exec.CommandContext(ctx, "false")
		}
		return exec.CommandContext(ctx, "echo", fmt.Sprintf(`{"access_token":"dapi-xxx"}`))
	}
	// login succeeds
	execCommand = fakeCommand("login ok", false)

	if err := EnsureAuthenticated("DEFAULT"); err != nil {
		t.Errorf("expected no error after successful login, got: %v", err)
	}
}
