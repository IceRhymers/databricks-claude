package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// helperDebugLog appends a single diagnostic line to
// ~/Library/Logs/databricks-claude/credential-helper.log (or the platform
// equivalent). Best-effort: failures are silent. Used only to diagnose how
// Claude Desktop spawns the helper.
func helperDebugLog(format string, args ...any) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return
	}
	dir := filepath.Join(home, "Library", "Logs", "databricks-claude")
	if runtime.GOOS != "darwin" {
		dir = filepath.Join(home, ".cache", "databricks-claude")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, "credential-helper.log"),
		os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s pid=%d ppid=%d ", time.Now().Format(time.RFC3339Nano), os.Getpid(), os.Getppid())
	fmt.Fprintf(f, format, args...)
	fmt.Fprintln(f)
}

// inferenceModelsJSON is the JSON-encoded model list embedded in the generated
// Claude Desktop configuration. Kept as a single source of truth so the macOS
// and Windows generators stay aligned.
const inferenceModelsJSON = `[{"name":"databricks-claude-opus-4-7","supports1m":true},{"name":"databricks-claude-opus-4-6","supports1m":true},{"name":"databricks-claude-sonnet-4-6","supports1m":true},{"name":"databricks-claude-sonnet-4-5","supports1m":true},{"name":"databricks-claude-haiku-4-5"}]`

// credentialHelperBinaryName is the basename used to dispatch into
// runCredentialHelper via argv[0]. Each install method is expected to install
// a symlink (or hard copy) at this name pointing at the main binary so that
// Claude Desktop's mobileconfig — which can only specify a path, not args —
// can target it directly.
const credentialHelperBinaryName = "databricks-claude-credential-helper"

// isCredentialHelperBinaryName returns true if the program was invoked under
// the credential-helper alias. Pass os.Args[0] (or any argv[0]-like value).
func isCredentialHelperBinaryName(arg0 string) bool {
	base := filepath.Base(arg0)
	// On Windows the symlink name will carry an .exe suffix.
	base = strings.TrimSuffix(base, ".exe")
	return base == credentialHelperBinaryName
}

// runCredentialHelper fetches a fresh Databricks OAuth token and writes only
// the raw token to stdout. Intended to be called by Claude Desktop via the
// inferenceCredentialHelper MDM key. Stays silent on stderr on success.
//
// Profile resolution mirrors the main flow: explicit --profile flag > saved
// state file > "DEFAULT".
func runCredentialHelper(profile string) {
	// Suppress all stdlib logging so the upstream tokencache cannot leak
	// anything onto stderr while Claude Desktop is watching.
	log.SetOutput(io.Discard)

	helperDebugLog("invoked args=%q HOME=%q PATH=%q USER=%q",
		os.Args, os.Getenv("HOME"), os.Getenv("PATH"), os.Getenv("USER"))

	resolved := profile
	if resolved == "" {
		if saved := loadState(); saved.Profile != "" {
			resolved = saved.Profile
		}
	}
	if resolved == "" {
		resolved = "DEFAULT"
	}
	helperDebugLog("profile resolved=%q (input=%q)", resolved, profile)

	tp := NewTokenProvider(resolved, "")
	tok, err := tp.Token(context.Background())
	if err != nil {
		helperDebugLog("FAIL profile=%q err=%v", resolved, err)
		fmt.Fprintf(os.Stderr, "databricks-claude: credential helper failed: %v\n", err)
		os.Exit(1)
	}
	tok = strings.TrimSpace(tok)
	if tok == "" {
		helperDebugLog("FAIL profile=%q empty token", resolved)
		fmt.Fprintln(os.Stderr, "databricks-claude: credential helper got empty token")
		os.Exit(1)
	}
	helperDebugLog("OK profile=%q tok_len=%d tok_prefix=%q", resolved, len(tok), tok[:min(20, len(tok))])
	// Write raw token, no trailing newline. Desktop reads stdout verbatim.
	if _, err := io.WriteString(os.Stdout, tok); err != nil {
		helperDebugLog("FAIL stdout write err=%v", err)
		os.Exit(1)
	}
	os.Exit(0)
}

// runGenerateDesktopConfig discovers the AI Gateway URL for the active profile
// and writes a platform-appropriate Claude Desktop MDM config file.
//
// On darwin → .mobileconfig (Apple Configuration Profile).
// On windows → .reg (Windows Registry script).
// On other OSes both are written so the user can transfer them.
//
// If outputPath is non-empty, that single path is used (the platform is
// inferred from the file extension when present, otherwise from runtime.GOOS).
//
// binaryPathOverride lets MDM admins bake a fleet-wide path into the generated
// config (e.g. /usr/local/bin/databricks-claude-credential-helper) so the same
// .mobileconfig works on every endpoint regardless of the generating user's
// install layout. When empty, the path is derived from the running binary.
func runGenerateDesktopConfig(profile, outputPath, binaryPathOverride string) {
	log.SetOutput(io.Discard)

	resolved := profile
	if resolved == "" {
		if saved := loadState(); saved.Profile != "" {
			resolved = saved.Profile
		}
	}
	if resolved == "" {
		resolved = "DEFAULT"
	}

	host, err := DiscoverHost(resolved, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "databricks-claude: failed to discover host for profile %q: %v\n", resolved, err)
		fmt.Fprintf(os.Stderr, "Run 'databricks auth login --profile %s' first.\n", resolved)
		os.Exit(1)
	}

	tp := NewTokenProvider(resolved, "")
	tok, err := tp.Token(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "databricks-claude: failed to fetch token for profile %q: %v\n", resolved, err)
		os.Exit(1)
	}
	gatewayURL := ConstructGatewayURL(host, tok)

	helperPath, err := resolveHelperPath(binaryPathOverride)
	if err != nil {
		fmt.Fprintf(os.Stderr, "databricks-claude: %v\n", err)
		os.Exit(1)
	}

	if outputPath != "" {
		if err := writeDesktopConfigByPath(outputPath, gatewayURL, helperPath); err != nil {
			fmt.Fprintf(os.Stderr, "databricks-claude: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Wrote Claude Desktop config: %s\n", outputPath)
		printInstallInstructions(outputPath)
		os.Exit(0)
	}

	wrote := []string{}
	if runtime.GOOS == "darwin" || (runtime.GOOS != "darwin" && runtime.GOOS != "windows") {
		path := "databricks-claude-desktop.mobileconfig"
		content, err := buildMobileconfig(gatewayURL, helperPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "databricks-claude: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "databricks-claude: write %s: %v\n", path, err)
			os.Exit(1)
		}
		wrote = append(wrote, path)
	}
	if runtime.GOOS == "windows" || (runtime.GOOS != "darwin" && runtime.GOOS != "windows") {
		path := "databricks-claude-desktop.reg"
		content := buildRegFile(gatewayURL, helperPath)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "databricks-claude: write %s: %v\n", path, err)
			os.Exit(1)
		}
		wrote = append(wrote, path)
	}

	for _, p := range wrote {
		fmt.Fprintf(os.Stderr, "Wrote Claude Desktop config: %s\n", p)
	}
	for _, p := range wrote {
		printInstallInstructions(p)
	}
	os.Exit(0)
}

// resolveHelperPath returns the absolute path embedded into the generated
// Claude Desktop config. Order:
//  1. explicit override (e.g. /usr/local/bin/databricks-claude-credential-helper)
//  2. derived from the running binary: replace its basename with the
//     credential-helper alias name, preserving the install dir and any .exe
//     suffix on Windows.
func resolveHelperPath(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("cannot resolve own executable path: %w", err)
	}
	dir := filepath.Dir(exe)
	name := credentialHelperBinaryName
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(dir, name), nil
}

// writeDesktopConfigByPath chooses the format based on file extension (or the
// host OS when no recognised extension is present) and writes to outputPath.
func writeDesktopConfigByPath(outputPath, gatewayURL, exe string) error {
	lower := strings.ToLower(outputPath)
	var content string
	var err error
	switch {
	case strings.HasSuffix(lower, ".mobileconfig"):
		content, err = buildMobileconfig(gatewayURL, exe)
	case strings.HasSuffix(lower, ".reg"):
		content = buildRegFile(gatewayURL, exe)
	default:
		// Fall back to host platform.
		if runtime.GOOS == "windows" {
			content = buildRegFile(gatewayURL, exe)
		} else {
			content, err = buildMobileconfig(gatewayURL, exe)
		}
	}
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, []byte(content), 0o600)
}

// printInstallInstructions writes user-facing guidance for the produced file
// to stderr (so it doesn't pollute stdout if anything is piping the binary).
func printInstallInstructions(path string) {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".mobileconfig"):
		fmt.Fprintf(os.Stderr, `
To install on macOS:
  1. Open the file:    open %q
  2. System Settings → Privacy & Security → Profiles → install the
     "Claude Desktop Third-Party Inference" profile.
  3. Restart Claude Desktop.
`, path)
	case strings.HasSuffix(lower, ".reg"):
		fmt.Fprintf(os.Stderr, `
To install on Windows:
  1. Double-click %q (or run: reg import "%s") to merge the keys
     into HKEY_CURRENT_USER\SOFTWARE\Policies\Claude.
  2. Restart Claude Desktop.
`, path, path)
	}
}

// newUUID returns an RFC 4122 v4 UUID string built from 16 random bytes.
// Pure stdlib (crypto/rand) — no third-party dependency.
func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("crypto/rand: %w", err)
	}
	// Set version (4) and variant (RFC 4122) bits.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08X-%04X-%04X-%04X-%012X",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// buildMobileconfig renders the macOS Claude Desktop Configuration Profile.
// Two distinct UUIDs are generated: one for the inner payload, one for the
// outer profile wrapper.
func buildMobileconfig(gatewayURL, helperPath string) (string, error) {
	innerUUID, err := newUUID()
	if err != nil {
		return "", err
	}
	outerUUID, err := newUUID()
	if err != nil {
		return "", err
	}

	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
	<dict>
		<key>PayloadContent</key>
		<array>
			<dict>
				<key>PayloadType</key>
				<string>com.anthropic.claudefordesktop</string>
				<key>PayloadIdentifier</key>
				<string>com.anthropic.claudefordesktop.settings</string>
				<key>PayloadUUID</key>
				<string>` + innerUUID + `</string>
				<key>PayloadVersion</key>
				<integer>1</integer>
				<key>PayloadDisplayName</key>
				<string>Claude Desktop</string>
				<key>disableDeploymentModeChooser</key>
				<true/>
				<key>inferenceProvider</key>
				<string>gateway</string>
				<key>inferenceGatewayBaseUrl</key>
				<string>` + plistEscape(gatewayURL) + `</string>
				<key>inferenceGatewayAuthScheme</key>
				<string>bearer</string>
				<key>inferenceModels</key>
				<string>` + plistEscape(inferenceModelsJSON) + `</string>
				<key>inferenceCredentialHelper</key>
				<string>` + plistEscape(helperPath) + `</string>
				<key>inferenceCredentialHelperTtlSec</key>
				<integer>55</integer>
				<key>isClaudeCodeForDesktopEnabled</key>
				<true/>
				<key>isDesktopExtensionEnabled</key>
				<true/>
				<key>isDesktopExtensionDirectoryEnabled</key>
				<true/>
				<key>isDesktopExtensionSignatureRequired</key>
				<false/>
				<key>isLocalDevMcpEnabled</key>
				<true/>
			</dict>
		</array>
		<key>PayloadDisplayName</key>
		<string>Claude Desktop Third-Party Inference</string>
		<key>PayloadIdentifier</key>
		<string>com.anthropic.claudefordesktop.profile</string>
		<key>PayloadType</key>
		<string>Configuration</string>
		<key>PayloadUUID</key>
		<string>` + outerUUID + `</string>
		<key>PayloadVersion</key>
		<integer>1</integer>
		<key>PayloadScope</key>
		<string>User</string>
	</dict>
</plist>
`, nil
}

// plistEscape escapes characters that are illegal inside a plist <string>
// element: &, <, >. Quotes/apostrophes don't strictly need escaping inside
// element content but we encode them defensively for safety.
func plistEscape(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return r.Replace(s)
}

// buildRegFile renders a Windows .reg script that writes the Claude Desktop
// MDM keys under HKCU\SOFTWARE\Policies\Claude.
func buildRegFile(gatewayURL, helperPath string) string {
	// .reg uses CRLF line endings and a UTF-16-or-UTF-8-with-BOM header.
	// Plain UTF-8 with the documented header works on modern Windows.
	var b strings.Builder
	b.WriteString("Windows Registry Editor Version 5.00\r\n\r\n")
	b.WriteString("[HKEY_CURRENT_USER\\SOFTWARE\\Policies\\Claude]\r\n")
	b.WriteString(`"disableDeploymentModeChooser"=dword:00000001` + "\r\n")
	b.WriteString(`"inferenceProvider"="gateway"` + "\r\n")
	fmt.Fprintf(&b, "\"inferenceGatewayBaseUrl\"=\"%s\"\r\n", regEscape(gatewayURL))
	b.WriteString(`"inferenceGatewayAuthScheme"="bearer"` + "\r\n")
	fmt.Fprintf(&b, "\"inferenceModels\"=\"%s\"\r\n", regEscape(inferenceModelsJSON))
	fmt.Fprintf(&b, "\"inferenceCredentialHelper\"=\"%s\"\r\n", regEscape(helperPath))
	b.WriteString(`"inferenceCredentialHelperTtlSec"="55"` + "\r\n")
	b.WriteString(`"isClaudeCodeForDesktopEnabled"=dword:00000001` + "\r\n")
	b.WriteString(`"isDesktopExtensionEnabled"=dword:00000001` + "\r\n")
	b.WriteString(`"isDesktopExtensionDirectoryEnabled"=dword:00000001` + "\r\n")
	b.WriteString(`"isDesktopExtensionSignatureRequired"=dword:00000000` + "\r\n")
	b.WriteString(`"isLocalDevMcpEnabled"=dword:00000001` + "\r\n")
	return b.String()
}

// regEscape escapes a string for use inside a Windows .reg REG_SZ value:
// backslashes and quotes get backslash-escaped.
func regEscape(s string) string {
	r := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
	)
	return r.Replace(s)
}

// extractProfileFlag scans args for --profile/--profile=value and returns the
// profile string if present. Used by the early-exit credential-helper and
// generate-desktop-config paths so they don't have to wait for parseArgs.
func extractProfileFlag(args []string) string {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--profile" && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(a, "--profile=") {
			return strings.TrimPrefix(a, "--profile=")
		}
	}
	return ""
}

// extractOutputFlag is the analogous helper for --output / --output=value.
func extractOutputFlag(args []string) string {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--output" && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(a, "--output=") {
			return strings.TrimPrefix(a, "--output=")
		}
	}
	return ""
}

// extractBinaryPathFlag is the analogous helper for --binary-path. Used by
// MDM admins to override the credential-helper path embedded in the generated
// Claude Desktop config.
func extractBinaryPathFlag(args []string) string {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--binary-path" && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(a, "--binary-path=") {
			return strings.TrimPrefix(a, "--binary-path=")
		}
	}
	return ""
}

// hasFlag returns true if any element of args equals name (or starts with
// name+"="). Used for early-exit flag detection at the top of main().
func hasFlag(args []string, name string) bool {
	for _, a := range args {
		if a == name || strings.HasPrefix(a, name+"=") {
			return true
		}
	}
	return false
}
