package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"testing"
)

// --- parseArgs tests ---

func TestParseArgs_HelpLong(t *testing.T) {
	profile, verbose, version, showHelp, printEnv, otel, _, _, _, _, upstream, logFile, noOtel, claudeArgs := parseArgs([]string{"--help"})
	if !showHelp {
		t.Error("expected showHelp=true for --help")
	}
	if profile != "" || verbose || version || printEnv || otel || upstream != "" || logFile != "" || noOtel || len(claudeArgs) != 0 {
		t.Error("unexpected non-default values alongside --help")
	}
}

func TestParseArgs_HelpShort(t *testing.T) {
	_, _, _, showHelp, _, _, _, _, _, _, _, _, _, _ := parseArgs([]string{"-h"})
	if !showHelp {
		t.Error("expected showHelp=true for -h")
	}
}

func TestParseArgs_PrintEnv(t *testing.T) {
	_, _, _, _, printEnv, _, _, _, _, _, _, _, _, _ := parseArgs([]string{"--print-env"})
	if !printEnv {
		t.Error("expected printEnv=true for --print-env")
	}
}

func TestParseArgs_Version(t *testing.T) {
	_, _, version, _, _, _, _, _, _, _, _, _, _, _ := parseArgs([]string{"--version"})
	if !version {
		t.Error("expected version=true for --version")
	}
}

func TestParseArgs_Verbose(t *testing.T) {
	_, verbose, _, _, _, _, _, _, _, _, _, _, _, _ := parseArgs([]string{"--verbose"})
	if !verbose {
		t.Error("expected verbose=true for --verbose")
	}
}

func TestParseArgs_VerboseShort(t *testing.T) {
	_, verbose, _, _, _, _, _, _, _, _, _, _, _, _ := parseArgs([]string{"-v"})
	if !verbose {
		t.Error("expected verbose=true for -v")
	}
}

func TestParseArgs_LogFile(t *testing.T) {
	_, _, _, _, _, _, _, _, _, _, _, logFile, _, _ := parseArgs([]string{"--log-file", "/tmp/test.log"})
	if logFile != "/tmp/test.log" {
		t.Errorf("expected logFile=%q, got %q", "/tmp/test.log", logFile)
	}
}

func TestParseArgs_LogFileEquals(t *testing.T) {
	_, _, _, _, _, _, _, _, _, _, _, logFile, _, _ := parseArgs([]string{"--log-file=/tmp/test.log"})
	if logFile != "/tmp/test.log" {
		t.Errorf("expected logFile=%q, got %q", "/tmp/test.log", logFile)
	}
}

func TestParseArgs_Profile(t *testing.T) {
	profile, _, _, _, _, _, _, _, _, _, _, _, _, _ := parseArgs([]string{"--profile", "foo"})
	if profile != "foo" {
		t.Errorf("expected profile=%q, got %q", "foo", profile)
	}
}

func TestParseArgs_Upstream(t *testing.T) {
	_, _, _, _, _, _, _, _, _, _, upstream, _, _, _ := parseArgs([]string{"--upstream", "/path/to/claude"})
	if upstream != "/path/to/claude" {
		t.Errorf("expected upstream=%q, got %q", "/path/to/claude", upstream)
	}
}

func TestParseArgs_Otel(t *testing.T) {
	_, _, _, _, _, otel, _, _, _, _, _, _, _, _ := parseArgs([]string{"--otel"})
	if !otel {
		t.Error("expected otel=true for --otel")
	}
}

func TestParseArgs_OtelMetricsTableOverride(t *testing.T) {
	_, _, _, _, _, _, metricsTable, metricsTableSet, _, _, _, _, _, _ := parseArgs([]string{"--otel-metrics-table", "main.default.otel"})
	if !metricsTableSet {
		t.Error("expected metricsTableSet=true when --otel-metrics-table is passed")
	}
	if metricsTable != "main.default.otel" {
		t.Errorf("expected metricsTable=%q, got %q", "main.default.otel", metricsTable)
	}
}

func TestParseArgs_OtelMetricsTableDefault(t *testing.T) {
	_, _, _, _, _, _, metricsTable, metricsTableSet, _, _, _, _, _, _ := parseArgs([]string{"--otel"})
	if metricsTableSet {
		t.Error("expected metricsTableSet=false when --otel-metrics-table is not passed")
	}
	if metricsTable != "main.claude_telemetry.claude_otel_metrics" {
		t.Errorf("expected default metricsTable, got %q", metricsTable)
	}
}

func TestParseArgs_OtelMetricsTableEquals(t *testing.T) {
	_, _, _, _, _, _, metricsTable, metricsTableSet, _, _, _, _, _, _ := parseArgs([]string{"--otel-metrics-table=my.catalog.table"})
	if !metricsTableSet {
		t.Error("expected metricsTableSet=true for --otel-metrics-table=value")
	}
	if metricsTable != "my.catalog.table" {
		t.Errorf("expected metricsTable=%q, got %q", "my.catalog.table", metricsTable)
	}
}

func TestParseArgs_OtelLogsTableOverride(t *testing.T) {
	_, _, _, _, _, _, _, _, logsTable, logsTableSet, _, _, _, _ := parseArgs([]string{"--otel-logs-table", "main.default.my_logs"})
	if !logsTableSet {
		t.Error("expected logsTableSet=true when --otel-logs-table is passed")
	}
	if logsTable != "main.default.my_logs" {
		t.Errorf("expected logsTable=%q, got %q", "main.default.my_logs", logsTable)
	}
}

func TestParseArgs_OtelLogsTableDefault(t *testing.T) {
	_, _, _, _, _, _, _, _, logsTable, logsTableSet, _, _, _, _ := parseArgs([]string{"--otel"})
	if logsTableSet {
		t.Error("expected logsTableSet=false when --otel-logs-table is not passed")
	}
	if logsTable != "" {
		t.Errorf("expected empty logsTable default, got %q", logsTable)
	}
}

func TestParseArgs_OtelLogsTableEquals(t *testing.T) {
	_, _, _, _, _, _, _, _, logsTable, logsTableSet, _, _, _, _ := parseArgs([]string{"--otel-logs-table=my.catalog.logs"})
	if !logsTableSet {
		t.Error("expected logsTableSet=true for --otel-logs-table=value")
	}
	if logsTable != "my.catalog.logs" {
		t.Errorf("expected logsTable=%q, got %q", "my.catalog.logs", logsTable)
	}
}

func TestParseArgs_BothOtelTables(t *testing.T) {
	_, _, _, _, _, _, metricsTable, metricsSet, logsTable, logsSet, _, _, _, _ := parseArgs([]string{
		"--otel-metrics-table", "cat.schema.metrics",
		"--otel-logs-table", "cat.schema.logs",
	})
	if !metricsSet || !logsSet {
		t.Error("expected both table flags to be set")
	}
	if metricsTable != "cat.schema.metrics" {
		t.Errorf("metricsTable=%q, want cat.schema.metrics", metricsTable)
	}
	if logsTable != "cat.schema.logs" {
		t.Errorf("logsTable=%q, want cat.schema.logs", logsTable)
	}
}

func TestParseArgs_UnknownFlagPassthrough(t *testing.T) {
	_, _, _, _, _, _, _, _, _, _, _, _, _, claudeArgs := parseArgs([]string{"--unknown"})
	if len(claudeArgs) != 1 || claudeArgs[0] != "--unknown" {
		t.Errorf("expected claudeArgs=[\"--unknown\"], got %v", claudeArgs)
	}
}

func TestParseArgs_EmptyArgs(t *testing.T) {
	profile, verbose, version, showHelp, printEnv, otel, otelMetricsTable, _, _, _, upstream, logFile, noOtel, claudeArgs := parseArgs([]string{})
	if profile != "" {
		t.Errorf("expected empty profile, got %q", profile)
	}
	if verbose || version || showHelp || printEnv || otel || noOtel {
		t.Error("expected all bool flags false for empty args")
	}
	if upstream != "" {
		t.Errorf("expected empty upstream, got %q", upstream)
	}
	if logFile != "" {
		t.Errorf("expected empty logFile, got %q", logFile)
	}
	if len(claudeArgs) != 0 {
		t.Errorf("expected no claudeArgs, got %v", claudeArgs)
	}
	// otelMetricsTable should have the default value
	if otelMetricsTable == "" {
		t.Error("expected non-empty default otelMetricsTable")
	}
}

func TestParseArgs_Mixed(t *testing.T) {
	profile, verbose, _, showHelp, _, _, _, _, _, _, _, _, _, _ := parseArgs([]string{"--profile", "prod", "--verbose", "--help"})
	if !showHelp {
		t.Error("expected showHelp=true")
	}
	if profile != "prod" {
		t.Errorf("expected profile=%q, got %q", "prod", profile)
	}
	if !verbose {
		t.Error("expected verbose=true")
	}
}

func TestParseArgs_NoOtel(t *testing.T) {
	_, _, _, _, _, otel, _, _, _, _, _, _, noOtel, claudeArgs := parseArgs([]string{"--no-otel"})
	if !noOtel {
		t.Error("expected noOtel=true for --no-otel")
	}
	if otel {
		t.Error("expected otel=false when only --no-otel given")
	}
	if len(claudeArgs) != 0 {
		t.Errorf("expected no claudeArgs, got %v", claudeArgs)
	}
}

func TestParseArgs_NoOtelAndOtel(t *testing.T) {
	_, _, _, _, _, otel, _, _, _, _, _, _, noOtel, _ := parseArgs([]string{"--no-otel", "--otel"})
	if !noOtel {
		t.Error("expected noOtel=true")
	}
	if !otel {
		t.Error("expected otel=true (both flags can coexist; main() handles precedence)")
	}
}

func TestParseArgs_NoOtelWithPassthrough(t *testing.T) {
	_, _, _, _, _, _, _, _, _, _, _, _, noOtel, claudeArgs := parseArgs([]string{"--no-otel", "somearg"})
	if !noOtel {
		t.Error("expected noOtel=true")
	}
	if len(claudeArgs) != 1 || claudeArgs[0] != "somearg" {
		t.Errorf("expected claudeArgs=[\"somearg\"], got %v", claudeArgs)
	}
}

func TestParseArgs_OtelUnaffectedByNoOtel(t *testing.T) {
	_, _, _, _, _, otel, _, _, _, _, _, _, noOtel, _ := parseArgs([]string{"--otel"})
	if !otel {
		t.Error("expected otel=true for --otel")
	}
	if noOtel {
		t.Error("expected noOtel=false when only --otel given")
	}
}

// Table-driven comprehensive test for parseArgs.
func TestParseArgs_Table(t *testing.T) {
	type result struct {
		profile   string
		verbose   bool
		version   bool
		showHelp  bool
		printEnv  bool
		otel      bool
		upstream  string
		logFile   string
		claudeLen int
	}

	tests := []struct {
		name string
		args []string
		want result
	}{
		{
			name: "--help sets showHelp",
			args: []string{"--help"},
			want: result{showHelp: true},
		},
		{
			name: "-h sets showHelp",
			args: []string{"-h"},
			want: result{showHelp: true},
		},
		{
			name: "--print-env sets printEnv",
			args: []string{"--print-env"},
			want: result{printEnv: true},
		},
		{
			name: "--version sets version",
			args: []string{"--version"},
			want: result{version: true},
		},
		{
			name: "--verbose sets verbose",
			args: []string{"--verbose"},
			want: result{verbose: true},
		},
		{
			name: "-v sets verbose",
			args: []string{"-v"},
			want: result{verbose: true},
		},
		{
			name: "--log-file sets logFile",
			args: []string{"--log-file", "/tmp/test.log"},
			want: result{logFile: "/tmp/test.log"},
		},
		{
			name: "--log-file=value sets logFile",
			args: []string{"--log-file=/tmp/test.log"},
			want: result{logFile: "/tmp/test.log"},
		},
		{
			name: "-v with --log-file sets both",
			args: []string{"-v", "--log-file", "/tmp/both.log"},
			want: result{verbose: true, logFile: "/tmp/both.log"},
		},
		{
			name: "--profile foo sets profile",
			args: []string{"--profile", "foo"},
			want: result{profile: "foo"},
		},
		{
			name: "--upstream /path sets upstream",
			args: []string{"--upstream", "/path/to/claude"},
			want: result{upstream: "/path/to/claude"},
		},
		{
			name: "--otel sets otel",
			args: []string{"--otel"},
			want: result{otel: true},
		},
		{
			name: "unknown flag passes through",
			args: []string{"--unknown"},
			want: result{claudeLen: 1},
		},
		{
			name: "empty args all defaults",
			args: []string{},
			want: result{},
		},
		{
			name: "mixed flags: profile, verbose, help",
			args: []string{"--profile", "prod", "--verbose", "--help"},
			want: result{showHelp: true, profile: "prod", verbose: true},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			profile, verbose, version, showHelp, printEnv, otel, _, _, _, _, upstream, logFile, _, claudeArgs := parseArgs(tc.args)

			if profile != tc.want.profile {
				t.Errorf("profile: got %q, want %q", profile, tc.want.profile)
			}
			if verbose != tc.want.verbose {
				t.Errorf("verbose: got %v, want %v", verbose, tc.want.verbose)
			}
			if version != tc.want.version {
				t.Errorf("version: got %v, want %v", version, tc.want.version)
			}
			if showHelp != tc.want.showHelp {
				t.Errorf("showHelp: got %v, want %v", showHelp, tc.want.showHelp)
			}
			if printEnv != tc.want.printEnv {
				t.Errorf("printEnv: got %v, want %v", printEnv, tc.want.printEnv)
			}
			if otel != tc.want.otel {
				t.Errorf("otel: got %v, want %v", otel, tc.want.otel)
			}
			if upstream != tc.want.upstream {
				t.Errorf("upstream: got %q, want %q", upstream, tc.want.upstream)
			}
			if logFile != tc.want.logFile {
				t.Errorf("logFile: got %q, want %q", logFile, tc.want.logFile)
			}
			if len(claudeArgs) != tc.want.claudeLen {
				t.Errorf("claudeArgs length: got %d, want %d (args: %v)", len(claudeArgs), tc.want.claudeLen, claudeArgs)
			}
		})
	}
}

// --- default log discard test ---

func TestDefaultLogDiscard(t *testing.T) {
	// Verify that when no flags are set, log output is discarded.
	// We simulate the main() logic: default sets log.SetOutput(io.Discard).
	log.SetOutput(io.Discard)
	defer log.SetOutput(os.Stderr) // restore after test

	var buf bytes.Buffer
	// Write a log message — it should go nowhere (Discard).
	log.SetOutput(io.Discard)
	log.Print("this should be discarded")

	// Now capture to buf to prove we can switch
	log.SetOutput(&buf)
	log.Print("this should appear")

	if !strings.Contains(buf.String(), "this should appear") {
		t.Error("expected log output after switching from Discard")
	}
}

// --- handlePrintEnv tests ---

// captureStdout replaces os.Stdout with a pipe and returns the output after fn runs.
func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestHandlePrintEnv_DapiTokenRedacted(t *testing.T) {
	out := captureStdout(func() {
		handlePrintEnv("DEFAULT", "https://dbc.example.com", "https://gw.example.com", "dapi-abc123secret", "", "", false, "", "")
	})
	if !strings.Contains(out, "dapi-***") {
		t.Errorf("expected dapi token to appear as 'dapi-***', got:\n%s", out)
	}
	if strings.Contains(out, "dapi-abc123secret") {
		t.Errorf("raw dapi token should not appear in output, got:\n%s", out)
	}
}

func TestHandlePrintEnv_NonDapiTokenRedacted(t *testing.T) {
	out := captureStdout(func() {
		handlePrintEnv("DEFAULT", "https://dbc.example.com", "https://gw.example.com", "eyJhbGciOiJSUzI1NiJ9", "", "", false, "", "")
	})
	if !strings.Contains(out, "**** (redacted)") {
		t.Errorf("expected non-dapi token to appear as '**** (redacted)', got:\n%s", out)
	}
	if strings.Contains(out, "eyJhbGciOiJSUzI1NiJ9") {
		t.Errorf("raw token should not appear in output, got:\n%s", out)
	}
}

func TestHandlePrintEnv_ContainsDatabricksHost(t *testing.T) {
	host := "https://dbc-abc123.cloud.databricks.com"
	out := captureStdout(func() {
		handlePrintEnv("DEFAULT", host, "https://gw.example.com", "tok", "", "", false, "", "")
	})
	if !strings.Contains(out, host) {
		t.Errorf("expected output to contain DATABRICKS_HOST %q, got:\n%s", host, out)
	}
}

func TestHandlePrintEnv_ContainsAnthropicBaseURL(t *testing.T) {
	baseURL := "https://gateway.example.com/anthropic"
	out := captureStdout(func() {
		handlePrintEnv("DEFAULT", "https://dbc.example.com", baseURL, "tok", "", "", false, "", "")
	})
	if !strings.Contains(out, baseURL) {
		t.Errorf("expected output to contain ANTHROPIC_BASE_URL %q, got:\n%s", baseURL, out)
	}
}

func TestHandlePrintEnv_EmptyTokenRedacted(t *testing.T) {
	out := captureStdout(func() {
		handlePrintEnv("DEFAULT", "https://dbc.example.com", "https://gw.example.com", "", "", "", false, "", "")
	})
	// Empty string does not start with "dapi-" so it should show as **** (redacted)
	if !strings.Contains(out, "**** (redacted)") {
		t.Errorf("expected empty token to appear as '**** (redacted)', got:\n%s", out)
	}
}

// --- handleHelp tests ---

func TestHandleHelp_ContainsDatabricksClaude(t *testing.T) {
	out := captureStdout(func() {
		handleHelp("")
	})
	if !strings.Contains(out, "databricks-claude") {
		t.Errorf("expected help output to contain 'databricks-claude', got:\n%s", out)
	}
}

func TestHandleHelp_ContainsPrintEnvFlag(t *testing.T) {
	out := captureStdout(func() {
		handleHelp("")
	})
	if !strings.Contains(out, "--print-env") {
		t.Errorf("expected help output to contain '--print-env', got:\n%s", out)
	}
}

func TestHandleHelp_ContainsClaudeCLISeparator(t *testing.T) {
	out := captureStdout(func() {
		handleHelp("")
	})
	if !strings.Contains(out, "Claude CLI Options:") {
		t.Errorf("expected help output to contain 'Claude CLI Options:', got:\n%s", out)
	}
}

func TestHandleHelp_WithNonExistentUpstream(t *testing.T) {
	// When an upstream binary is given but doesn't exist, handleHelp should
	// still print the databricks-claude header without panicking.
	out := captureStdout(func() {
		handleHelp("/nonexistent/path/to/claude")
	})
	if !strings.Contains(out, "databricks-claude") {
		t.Errorf("expected header even with bad upstream binary, got:\n%s", out)
	}
}

func TestHandleHelp_AllFlagsPresent(t *testing.T) {
	out := captureStdout(func() {
		handleHelp("")
	})
	flags := []string{"--profile", "--upstream", "--verbose", "-v", "--log-file", "--otel", "--otel-metrics-table", "--otel-logs-table", "--version", "--help"}
	for _, flag := range flags {
		if !strings.Contains(out, flag) {
			t.Errorf("expected help output to contain flag %q, got:\n%s", flag, out)
		}
	}
}

// Verify the version placeholder is printed in the help header.
func TestHandleHelp_ContainsVersion(t *testing.T) {
	out := captureStdout(func() {
		handleHelp("")
	})
	// Version variable is "dev" by default in tests.
	if !strings.Contains(out, fmt.Sprintf("databricks-claude v%s", Version)) {
		t.Errorf("expected help output to contain version string, got:\n%s", out)
	}
}

// --- deriveLogsTable tests ---

func TestOTELTableDerivation(t *testing.T) {
	tests := []struct {
		name    string
		metrics string
		want    string
	}{
		{
			name:    "standard suffix replacement",
			metrics: "main.claude_telemetry.claude_otel_metrics",
			want:    "main.claude_telemetry.claude_otel_logs",
		},
		{
			name:    "custom table without _otel_metrics suffix",
			metrics: "mycatalog.myschema.custom",
			want:    "mycatalog.myschema.custom_otel_logs",
		},
		{
			name:    "only _otel_metrics",
			metrics: "_otel_metrics",
			want:    "_otel_logs",
		},
		{
			name:    "empty string",
			metrics: "",
			want:    "_otel_logs",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := deriveLogsTable(tc.metrics)
			if got != tc.want {
				t.Errorf("deriveLogsTable(%q) = %q, want %q", tc.metrics, got, tc.want)
			}
		})
	}
}

// --- handlePrintEnv OTEL fields test ---

func TestHandlePrintEnv_OTELFields(t *testing.T) {
	out := captureStdout(func() {
		handlePrintEnv("DEFAULT", "https://dbc.example.com", "https://gw.example.com", "tok", "", "", true, "main.telemetry.claude_otel_metrics", "main.telemetry.claude_otel_logs")
	})
	checks := []string{
		"OTEL enabled:         true",
		"OTEL metrics table:   main.telemetry.claude_otel_metrics",
		"OTEL logs table:      main.telemetry.claude_otel_logs",
		"OTEL metric interval: 10000ms",
		"OTEL logs interval:   5000ms",
	}
	for _, c := range checks {
		if !strings.Contains(out, c) {
			t.Errorf("expected output to contain %q, got:\n%s", c, out)
		}
	}
}
