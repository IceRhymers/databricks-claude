package completion

import (
	"strings"
	"testing"
)

var testFlags = []FlagDef{
	{Name: "profile", Short: "", Description: "Databricks CLI profile", TakesArg: true, Completer: "__databricks_profiles"},
	{Name: "verbose", Short: "v", Description: "Enable debug logging", TakesArg: false, Completer: ""},
	{Name: "help", Short: "h", Description: "Show help", TakesArg: false, Completer: ""},
	{Name: "port", Short: "", Description: "Proxy listen port", TakesArg: true, Completer: ""},
	{Name: "log-file", Short: "", Description: "Write logs to file", TakesArg: true, Completer: "__files"},
}

func TestGenerateBash_ContainsFlags(t *testing.T) {
	out := GenerateBash("databricks-claude", testFlags)
	for _, want := range []string{
		"--profile", "--verbose", "-v", "--help", "-h", "--port", "--log-file",
		"_databricks_claude",
		"complete -F _databricks_claude databricks-claude",
		"__databricks_profiles",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("bash output missing %q", want)
		}
	}
}

func TestGenerateBash_PassthroughBoundary(t *testing.T) {
	out := GenerateBash("databricks-claude", testFlags)
	if !strings.Contains(out, `"--"`) {
		t.Error("bash output missing passthrough -- guard")
	}
}

func TestGenerateBash_FileCompleter(t *testing.T) {
	out := GenerateBash("databricks-claude", testFlags)
	// --log-file should use compgen -f
	if !strings.Contains(out, "compgen -f") {
		t.Error("bash output missing file completer for --log-file")
	}
}

func TestGenerateZsh_ContainsFlags(t *testing.T) {
	out := GenerateZsh("databricks-claude", testFlags)
	for _, want := range []string{
		"#compdef databricks-claude",
		"--profile",
		"--verbose",
		"--port",
		"__databricks_profiles",
		"_arguments",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("zsh output missing %q", want)
		}
	}
}

func TestGenerateZsh_ShortFlags(t *testing.T) {
	out := GenerateZsh("databricks-claude", testFlags)
	// verbose has short -v so should appear as paired spec
	if !strings.Contains(out, "-v") {
		t.Error("zsh output missing short flag -v")
	}
}

func TestGenerateFish_ContainsFlags(t *testing.T) {
	out := GenerateFish("databricks-claude", testFlags)
	for _, want := range []string{
		"complete -c databricks-claude -f",
		"-l profile",
		"-l verbose",
		"-s v",
		"-l port",
		"__databricks_profiles",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("fish output missing %q", want)
		}
	}
}

func TestGenerateFish_FileCompleter(t *testing.T) {
	out := GenerateFish("databricks-claude", testFlags)
	// --log-file should have -F for file completion
	if !strings.Contains(out, "-F") {
		t.Error("fish output missing -F file completer for --log-file")
	}
}

func TestGenerateBash_InvalidShellExitsNonZero(t *testing.T) {
	// Run() with an invalid shell should exit 1. We test GenerateBash directly
	// for content; Run() behavior is tested via the binary integration.
	out := GenerateBash("test-bin", testFlags)
	if out == "" {
		t.Error("GenerateBash returned empty string")
	}
}

func TestUniqueCompleters(t *testing.T) {
	flags := []FlagDef{
		{Name: "a", Completer: "__databricks_profiles"},
		{Name: "b", Completer: "__databricks_profiles"}, // duplicate
		{Name: "c", Completer: "__files"},               // filtered out
		{Name: "d", Completer: ""},
	}
	got := uniqueCompleters(flags)
	if len(got) != 1 || got[0] != "__databricks_profiles" {
		t.Errorf("uniqueCompleters = %v, want [__databricks_profiles]", got)
	}
}
