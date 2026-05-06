package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsExecutableFile_Executable(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "exec-*")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if err := os.Chmod(f.Name(), 0o755); err != nil {
		t.Fatal(err)
	}
	if !IsExecutableFile(f.Name()) {
		t.Errorf("expected %q to be reported as executable", f.Name())
	}
}

func TestIsExecutableFile_NonExecutable(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "noexec-*")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if err := os.Chmod(f.Name(), 0o644); err != nil {
		t.Fatal(err)
	}
	if IsExecutableFile(f.Name()) {
		t.Errorf("expected %q to not be executable", f.Name())
	}
}

func TestIsExecutableFile_Missing(t *testing.T) {
	if IsExecutableFile("/nonexistent/path/to/binary") {
		t.Error("expected false for missing file")
	}
}

func TestIsExecutableFile_Directory(t *testing.T) {
	dir := t.TempDir()
	if IsExecutableFile(dir) {
		t.Errorf("expected false for directory %q", dir)
	}
}

func TestResolveDatabricksCLI_EmptyDefaultsToDatabricks(t *testing.T) {
	// When the binary isn't found, it returns the default name unchanged.
	result := ResolveDatabricksCLI("")
	if result == "" {
		t.Error("expected non-empty result for empty cmdName")
	}
	// Should either resolve to an absolute path or fall back to "databricks".
	if filepath.Base(result) != "databricks" && !filepath.IsAbs(result) {
		t.Errorf("unexpected result for empty cmdName: %q", result)
	}
}

func TestResolveDatabricksCLI_AbsolutePassthrough(t *testing.T) {
	abs := "/usr/local/bin/databricks"
	if got := ResolveDatabricksCLI(abs); got != abs {
		t.Errorf("expected absolute path returned unchanged, got %q", got)
	}
}

func TestResolveDatabricksCLI_EnvOverride(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "fake-databricks")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\necho ok\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DATABRICKS_CLI", fake)
	if got := ResolveDatabricksCLI("databricks"); got != fake {
		t.Errorf("expected DATABRICKS_CLI override %q, got %q", fake, got)
	}
}

func TestResolveDatabricksCLI_EnvOverrideNotExecutable(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "not-exec")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DATABRICKS_CLI", fake)
	// Should NOT use the override since it's not executable; falls through to LookPath/fallback.
	got := ResolveDatabricksCLI("databricks")
	if got == fake {
		t.Errorf("should not use non-executable DATABRICKS_CLI override, but got %q", got)
	}
}

func TestResolveDatabricksCLI_FallbackDir(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "mybinary")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	orig := FallbackCLIDirs
	FallbackCLIDirs = []string{dir}
	defer func() { FallbackCLIDirs = orig }()

	// Clear DATABRICKS_CLI so it falls through to fallback dirs.
	t.Setenv("DATABRICKS_CLI", "")

	got := ResolveDatabricksCLI("mybinary")
	if got != fake {
		t.Errorf("expected fallback dir resolution to %q, got %q", fake, got)
	}
}
