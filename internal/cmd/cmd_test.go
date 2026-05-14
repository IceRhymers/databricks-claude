package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestAllFlagsOrdersPersistentBeforeFlags(t *testing.T) {
	c := Command{
		Persistent: []FlagDef{{Name: "profile"}, {Name: "port"}},
		Flags:      []FlagDef{{Name: "verbose"}},
	}
	got := c.AllFlags()
	want := []string{"profile", "port", "verbose"}
	if len(got) != len(want) {
		t.Fatalf("AllFlags len=%d, want %d", len(got), len(want))
	}
	for i, name := range want {
		if got[i].Name != name {
			t.Errorf("AllFlags[%d].Name = %q, want %q", i, got[i].Name, name)
		}
	}
}

func TestKnownFlagsCoversPersistentAndLocal(t *testing.T) {
	c := Command{
		Persistent: []FlagDef{{Name: "profile"}},
		Flags:      []FlagDef{{Name: "verbose"}, {Name: "port", TakesArg: true}},
	}
	known := c.KnownFlags()
	for _, want := range []string{"--profile", "--verbose", "--port"} {
		if !known[want] {
			t.Errorf("KnownFlags missing %q", want)
		}
	}
	if known["--unknown"] {
		t.Errorf("KnownFlags should not contain --unknown")
	}
}

func TestToCompletionDropsResolutionMetadata(t *testing.T) {
	f := FlagDef{
		Name:        "profile",
		Short:       "p",
		Description: "Databricks profile",
		TakesArg:    true,
		Completer:   "__databricks_profiles",
		StateKey:    "profile",
		EnvVar:      "DATABRICKS_CONFIG_PROFILE",
		MDMKey:      "databricksProfile",
		Default:     "DEFAULT",
	}
	got := f.ToCompletion()
	if got.Name != "profile" || got.Short != "p" || got.Description != "Databricks profile" ||
		!got.TakesArg || got.Completer != "__databricks_profiles" {
		t.Errorf("ToCompletion lost completion-relevant fields: %+v", got)
	}
}

func TestCompletionSubcommandsMapsShortToDescription(t *testing.T) {
	c := Command{Subcommands: []Command{
		{Name: "serve", Short: "Long-lived daemon"},
		{Name: "setup", Short: "Idempotent auth bootstrap"},
	}}
	got := c.CompletionSubcommands()
	if len(got) != 2 || got[0].Name != "serve" || got[0].Description != "Long-lived daemon" ||
		got[1].Name != "setup" || got[1].Description != "Idempotent auth bootstrap" {
		t.Errorf("CompletionSubcommands = %+v", got)
	}
}

func TestRenderUsesLongVerbatimWithVarSubstitution(t *testing.T) {
	c := Command{Long: "version={{Version}}\n"}
	var buf bytes.Buffer
	if err := Render(&buf, c, map[string]string{"Version": "1.2.3"}); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if buf.String() != "version=1.2.3\n" {
		t.Errorf("Render output = %q, want %q", buf.String(), "version=1.2.3\n")
	}
}

func TestRenderFallsBackToProgrammaticWhenLongIsEmpty(t *testing.T) {
	c := Command{
		Name:  "demo",
		Short: "Example",
		Flags: []FlagDef{{Name: "verbose", Description: "Enable debug"}},
		Subcommands: []Command{
			{Name: "child", Short: "A child"},
		},
	}
	var buf bytes.Buffer
	if err := Render(&buf, c, nil); err != nil {
		t.Fatalf("Render: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"demo — Example", "--verbose", "Enable debug", "child", "A child"} {
		if !strings.Contains(out, want) {
			t.Errorf("Render fallback missing %q in:\n%s", want, out)
		}
	}
}
