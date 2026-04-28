package main

import (
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// uuidPattern matches the canonical 8-4-4-4-12 hex UUID layout produced by
// newUUID(). Hex is upper-case to mirror the format string in newUUID.
var uuidPattern = regexp.MustCompile(`^[0-9A-F]{8}-[0-9A-F]{4}-4[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$`)

func TestNewUUID_FormatAndUniqueness(t *testing.T) {
	a, err := newUUID()
	if err != nil {
		t.Fatalf("newUUID: %v", err)
	}
	b, err := newUUID()
	if err != nil {
		t.Fatalf("newUUID: %v", err)
	}
	if !uuidPattern.MatchString(a) {
		t.Errorf("UUID %q does not match v4 layout", a)
	}
	if !uuidPattern.MatchString(b) {
		t.Errorf("UUID %q does not match v4 layout", b)
	}
	if a == b {
		t.Errorf("two consecutive UUIDs collided: %q", a)
	}
}

func TestBuildMobileconfig_ContainsRequiredKeys(t *testing.T) {
	gateway := "https://abc-123.ai-gateway.cloud.databricks.com/anthropic"
	helper := "/usr/local/bin/databricks-claude"
	out, err := buildMobileconfig(gateway, helper)
	if err != nil {
		t.Fatalf("buildMobileconfig: %v", err)
	}

	// Spot-check the must-have keys and values.
	for _, want := range []string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<key>PayloadType</key>`,
		`<string>com.anthropic.claudefordesktop</string>`,
		`<key>inferenceProvider</key>`,
		`<string>gateway</string>`,
		`<key>inferenceGatewayBaseUrl</key>`,
		`<string>` + gateway + `</string>`,
		`<key>inferenceCredentialHelper</key>`,
		`<string>` + helper + `</string>`,
		`<key>inferenceCredentialHelperTtlSec</key>`,
		`<integer>55</integer>`,
		`<key>inferenceModels</key>`,
		`databricks-claude-opus-4-7`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("mobileconfig missing %q", want)
		}
	}
}

func TestBuildMobileconfig_EscapesSpecialChars(t *testing.T) {
	// Gateway URL with an ampersand should be plist-escaped.
	gateway := "https://example.com/anthropic?a=1&b=2"
	helper := "/Applications/Foo & Bar/databricks-claude"
	out, err := buildMobileconfig(gateway, helper)
	if err != nil {
		t.Fatalf("buildMobileconfig: %v", err)
	}
	if strings.Contains(out, "?a=1&b=2</string>") {
		t.Errorf("ampersand in gateway URL was not escaped")
	}
	if !strings.Contains(out, "?a=1&amp;b=2") {
		t.Errorf("expected escaped ampersand, got: %s", out)
	}
	if !strings.Contains(out, "Foo &amp; Bar") {
		t.Errorf("expected escaped ampersand in helper path")
	}
}

func TestBuildMobileconfig_UniqueUUIDs(t *testing.T) {
	out, err := buildMobileconfig("https://x", "/bin/x")
	if err != nil {
		t.Fatalf("buildMobileconfig: %v", err)
	}
	uuidsRe := regexp.MustCompile(`<string>([0-9A-F]{8}-[0-9A-F]{4}-4[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12})</string>`)
	matches := uuidsRe.FindAllStringSubmatch(out, -1)
	if len(matches) != 2 {
		t.Fatalf("expected exactly 2 UUIDs in mobileconfig, got %d", len(matches))
	}
	if matches[0][1] == matches[1][1] {
		t.Errorf("inner and outer UUIDs must differ, got %q for both", matches[0][1])
	}
}

func TestBuildRegFile_ContainsRequiredKeys(t *testing.T) {
	gateway := "https://abc-123.ai-gateway.cloud.databricks.com/anthropic"
	helper := `C:\Program Files\databricks-claude\databricks-claude.exe`
	out := buildRegFile(gateway, helper)

	for _, want := range []string{
		`Windows Registry Editor Version 5.00`,
		`[HKEY_CURRENT_USER\SOFTWARE\Policies\Claude]`,
		`"disableDeploymentModeChooser"=dword:00000001`,
		`"inferenceProvider"="gateway"`,
		`"inferenceGatewayBaseUrl"="` + gateway + `"`,
		`"inferenceGatewayAuthScheme"="bearer"`,
		`"inferenceCredentialHelperTtlSec"="55"`,
		`"isClaudeCodeForDesktopEnabled"=dword:00000001`,
		`"isDesktopExtensionSignatureRequired"=dword:00000000`,
		`"isLocalDevMcpEnabled"=dword:00000001`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf(".reg missing %q", want)
		}
	}

	// Backslashes in the helper path must be doubled.
	if !strings.Contains(out, `C:\\Program Files\\databricks-claude\\databricks-claude.exe`) {
		t.Errorf("helper path backslashes not escaped, got: %s", out)
	}

	// JSON quotes in the inferenceModels string must be escaped.
	if !strings.Contains(out, `\"name\":\"databricks-claude-opus-4-7\"`) {
		t.Errorf("inferenceModels JSON quotes not escaped")
	}

	// Lines must end with CRLF (the .reg format requires it).
	if !strings.Contains(out, "\r\n") {
		t.Errorf(".reg output is missing CRLF line endings")
	}
}

func TestRegEscape(t *testing.T) {
	cases := map[string]string{
		`plain`:                      `plain`,
		`with "quotes"`:              `with \"quotes\"`,
		`C:\path\to\bin`:             `C:\\path\\to\\bin`,
		`"quote at start`:            `\"quote at start`,
		`back\slash and "quote"`:     `back\\slash and \"quote\"`,
	}
	for in, want := range cases {
		if got := regEscape(in); got != want {
			t.Errorf("regEscape(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPlistEscape(t *testing.T) {
	cases := map[string]string{
		`plain`:           `plain`,
		`a & b`:           `a &amp; b`,
		`<tag>`:           `&lt;tag&gt;`,
		`he said "hi"`:    `he said &quot;hi&quot;`,
		`it's & <ok>`:     `it&apos;s &amp; &lt;ok&gt;`,
	}
	for in, want := range cases {
		if got := plistEscape(in); got != want {
			t.Errorf("plistEscape(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExtractProfileFlag(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{}, ""},
		{[]string{"--credential-helper"}, ""},
		{[]string{"--profile", "prod", "--credential-helper"}, "prod"},
		{[]string{"--credential-helper", "--profile", "dev"}, "dev"},
		{[]string{"--profile=staging"}, "staging"},
		{[]string{"--credential-helper", "--profile=qa"}, "qa"},
		// Bare --profile without a value must not panic and must return "".
		{[]string{"--profile"}, ""},
	}
	for _, c := range cases {
		if got := extractProfileFlag(c.args); got != c.want {
			t.Errorf("extractProfileFlag(%v) = %q, want %q", c.args, got, c.want)
		}
	}
}

func TestExtractOutputFlag(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{}, ""},
		{[]string{"--generate-desktop-config"}, ""},
		{[]string{"--output", "/tmp/foo.mobileconfig"}, "/tmp/foo.mobileconfig"},
		{[]string{"--output=/tmp/foo.reg"}, "/tmp/foo.reg"},
		{[]string{"--generate-desktop-config", "--output", "/tmp/x"}, "/tmp/x"},
	}
	for _, c := range cases {
		if got := extractOutputFlag(c.args); got != c.want {
			t.Errorf("extractOutputFlag(%v) = %q, want %q", c.args, got, c.want)
		}
	}
}

func TestHasFlag(t *testing.T) {
	if !hasFlag([]string{"--credential-helper"}, "--credential-helper") {
		t.Error("hasFlag should match exact flag")
	}
	if !hasFlag([]string{"--credential-helper=true"}, "--credential-helper") {
		t.Error("hasFlag should match --flag=value form")
	}
	if hasFlag([]string{"--credential-helper-thing"}, "--credential-helper") {
		t.Error("hasFlag must not match a flag with a longer name")
	}
	if hasFlag([]string{"--other"}, "--credential-helper") {
		t.Error("hasFlag should not match unrelated flag")
	}
	if hasFlag(nil, "--credential-helper") {
		t.Error("hasFlag on nil should return false")
	}
}

func TestIsCredentialHelperBinaryName(t *testing.T) {
	cases := []struct {
		arg0 string
		want bool
	}{
		{"databricks-claude-credential-helper", true},
		{"/usr/local/bin/databricks-claude-credential-helper", true},
		{"/opt/homebrew/bin/databricks-claude-credential-helper", true},
		{"databricks-claude-credential-helper.exe", true},
		// Backslash path-separator handling is exercised in the platform-specific
		// test below; filepath.Base on darwin/linux treats backslashes as
		// literal characters, so we don't include a Windows-style path here.
		// Main binary name must NOT trigger helper dispatch.
		{"databricks-claude", false},
		{"/usr/local/bin/databricks-claude", false},
		{"databricks-claude.exe", false},
		// Near-misses must not match.
		{"databricks-claude-credential-helper-extra", false},
		{"my-databricks-claude-credential-helper", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isCredentialHelperBinaryName(c.arg0); got != c.want {
			t.Errorf("isCredentialHelperBinaryName(%q) = %v, want %v", c.arg0, got, c.want)
		}
	}
}

func TestExtractDatabricksCLIPathFlag(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{}, ""},
		{[]string{"--generate-desktop-config"}, ""},
		{[]string{"--databricks-cli-path", "/usr/local/bin/databricks"}, "/usr/local/bin/databricks"},
		{[]string{"--databricks-cli-path=/opt/homebrew/bin/databricks"}, "/opt/homebrew/bin/databricks"},
		{[]string{"--generate-desktop-config", "--databricks-cli-path", "/x"}, "/x"},
		// Bare flag without a value must not panic and must return "".
		{[]string{"--databricks-cli-path"}, ""},
	}
	for _, c := range cases {
		if got := extractDatabricksCLIPathFlag(c.args); got != c.want {
			t.Errorf("extractDatabricksCLIPathFlag(%v) = %q, want %q", c.args, got, c.want)
		}
	}
}

func TestExtractBinaryPathFlag(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{}, ""},
		{[]string{"--generate-desktop-config"}, ""},
		{[]string{"--binary-path", "/usr/local/bin/databricks-claude-credential-helper"}, "/usr/local/bin/databricks-claude-credential-helper"},
		{[]string{"--binary-path=/opt/homebrew/bin/databricks-claude-credential-helper"}, "/opt/homebrew/bin/databricks-claude-credential-helper"},
		{[]string{"--generate-desktop-config", "--binary-path", "/x"}, "/x"},
		// Bare --binary-path without a value must not panic and must return "".
		{[]string{"--binary-path"}, ""},
	}
	for _, c := range cases {
		if got := extractBinaryPathFlag(c.args); got != c.want {
			t.Errorf("extractBinaryPathFlag(%v) = %q, want %q", c.args, got, c.want)
		}
	}
}

func TestResolveHelperPath_Override(t *testing.T) {
	override := "/usr/local/bin/databricks-claude-credential-helper"
	got, err := resolveHelperPath(override)
	if err != nil {
		t.Fatalf("resolveHelperPath: %v", err)
	}
	if got != override {
		t.Errorf("resolveHelperPath(%q) = %q, want %q", override, got, override)
	}
}

func TestResolveHelperPath_DerivedFromExecutable(t *testing.T) {
	// With no override, the helper path is the running test binary's
	// directory + the credential-helper alias name. We can't predict the
	// exact dir but we can assert the basename.
	got, err := resolveHelperPath("")
	if err != nil {
		t.Fatalf("resolveHelperPath: %v", err)
	}
	wantBase := credentialHelperBinaryName
	if runtime.GOOS == "windows" {
		wantBase += ".exe"
	}
	if filepath.Base(got) != wantBase {
		t.Errorf("resolveHelperPath(\"\") basename = %q, want %q (full=%q)", filepath.Base(got), wantBase, got)
	}
}
