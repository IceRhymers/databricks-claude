package state

import (
	"os"
	"path/filepath"
	"testing"
)

type testState struct {
	Profile string `json:"profile,omitempty"`
	Port    int    `json:"port,omitempty"`
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	if err := Save(path, testState{Profile: "aidev", Port: 49154}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load[testState](path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Profile != "aidev" {
		t.Errorf("Profile = %q, want %q", got.Profile, "aidev")
	}
	if got.Port != 49154 {
		t.Errorf("Port = %d, want %d", got.Port, 49154)
	}
}

func TestLoad_Missing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.json")

	got, err := Load[testState](path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Profile != "" || got.Port != 0 {
		t.Errorf("expected zero value from missing file, got %+v", got)
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	os.WriteFile(path, []byte("not json"), 0o644)

	got, err := Load[testState](path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Profile != "" {
		t.Errorf("expected zero value from invalid JSON, got %+v", got)
	}
}

func TestSave_Overwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	Save(path, testState{Profile: "first"})
	Save(path, testState{Profile: "second"})

	got, _ := Load[testState](path)
	if got.Profile != "second" {
		t.Errorf("Profile = %q, want %q", got.Profile, "second")
	}
}

func TestResolvePort(t *testing.T) {
	tests := []struct {
		name        string
		flagPort    int
		savedPort   int
		defaultPort int
		want        int
	}{
		{"flag wins", 9999, 8080, 49153, 9999},
		{"saved wins over default", 0, 8080, 49153, 8080},
		{"default when no flag and no saved", 0, 0, 49153, 49153},
		{"flag wins over default", 5555, 0, 49153, 5555},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolvePort(tc.flagPort, tc.savedPort, tc.defaultPort)
			if got != tc.want {
				t.Errorf("ResolvePort(%d, %d, %d) = %d, want %d",
					tc.flagPort, tc.savedPort, tc.defaultPort, got, tc.want)
			}
		})
	}
}
