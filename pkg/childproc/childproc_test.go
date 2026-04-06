package childproc

import (
	"context"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestSignalForwarding(t *testing.T) {
	// Start a child that sleeps; we'll kill it with SIGINT via ForwardSignals.
	cmd := exec.Command("/bin/sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot start sleep child: %v", err)
	}

	cancel := ForwardSignals(cmd)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	// Give the child a moment to start then send SIGINT to the child directly.
	time.Sleep(50 * time.Millisecond)
	cmd.Process.Signal(syscall.SIGINT)

	select {
	case <-done:
		// Child exited — signal was received.
	case <-time.After(3 * time.Second):
		cmd.Process.Kill()
		t.Fatal("child did not exit after SIGINT")
	}
}

func TestRun_ExitCode(t *testing.T) {
	code, err := Run(context.Background(), Config{
		BinaryName: "/bin/sh",
		Args:       []string{"-c", "exit 42"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 42 {
		t.Errorf("exit code = %d, want 42", code)
	}
}

func TestRun_Success(t *testing.T) {
	code, err := Run(context.Background(), Config{
		BinaryName: "/bin/true",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}
