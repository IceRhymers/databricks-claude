package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// buildHelperBinary compiles a small helper binary that prints a fixed JSON response
// and exits with a given code. Returns the path to the binary.
//
// Kept here (rather than deleted alongside the ported token-provider tests) because
// desktop_helper_test.go still consumes it.
func buildHelperBinary(t *testing.T, jsonPayload string, exitCode int) string {
	t.Helper()
	dir := t.TempDir()

	src := filepath.Join(dir, "main.go")
	bin := filepath.Join(dir, "helper")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}

	// Escape the payload for embedding in Go source
	payloadLiteral, _ := json.Marshal(jsonPayload)

	code := fmt.Sprintf(`package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Print(%s)
	os.Exit(%d)
}
`, string(payloadLiteral), exitCode)

	if err := os.WriteFile(src, []byte(code), 0600); err != nil {
		t.Fatalf("write helper src: %v", err)
	}

	cmd := exec.Command("go", "build", "-o", bin, src)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build helper: %v\n%s", err, out)
	}
	return bin
}

func TestConstructGatewayURL(t *testing.T) {
	tests := []struct {
		host string
		want string
	}{
		{"https://adb-123.azuredatabricks.net", "https://adb-123.azuredatabricks.net/ai-gateway/anthropic"},
		{"https://adb-123.azuredatabricks.net/", "https://adb-123.azuredatabricks.net/ai-gateway/anthropic"},
	}
	for _, tc := range tests {
		got := ConstructGatewayURL(tc.host)
		if got != tc.want {
			t.Errorf("ConstructGatewayURL(%q) = %q, want %q", tc.host, got, tc.want)
		}
	}
}
