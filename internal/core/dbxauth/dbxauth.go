// Package dbxauth shells out to the Databricks CLI for OAuth tokens and
// workspace host discovery, on behalf of every launcher.
//
// Before #218 each launcher carried its own byte-identical copy of this logic.
// The copies had drifted in two ways that mattered:
//
//   - Argument order. claude took (profile, cmdName); codex and opencode took
//     (cmdName, profile). Both are (string, string), so a call site with the
//     wrong order compiled silently.
//   - CLI resolution. Only claude routed the CLI name through internal/core/cli.
//     codex and opencode exec'd the bare name, so under launchd/systemd's
//     minimal PATH their token fetch could fail to find a CLI that their own
//     authcheck path had just resolved successfully.
//
// This package is the single implementation. It always resolves through
// internal/core/cli, which fixes the second divergence for codex and opencode.
//
// Per-tool knowledge stays in the launchers: each owns its gateway path
// constant(s) and passes them to GatewayURL. That seam is not stylistic —
// opencode has two upstreams (Anthropic and Gemini Native), so a single
// per-tool path value cannot represent it.
//
// PRECONDITION — the MDM tier is process-global. cli.ResolveDatabricksCLI
// consults an MDM-managed databricksCliPath via a package-level reader that
// callers wire with cli.SetMDMReader. Only databricks-claude wires it (in
// main.go, above every early-exit dispatcher); the default is a no-op reader,
// so for codex and opencode the MDM tier is simply skipped. That is correct:
// the MDM domain is claude-branded and only Claude Desktop provisions the key.
// This coupling predates #218 — authcheck already resolves through the same
// global — and aligning both packages on an explicit reader is tracked
// separately (see AGENTS.md).
package dbxauth

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/IceRhymers/databricks-agents/internal/core/cli"
	"github.com/IceRhymers/databricks-agents/internal/core/tokencache"
)

// fetchTimeout bounds each Databricks CLI subprocess.
const fetchTimeout = 10 * time.Second

// Config identifies which Databricks CLI to run, and against which profile.
//
// Use keyed literals. Both fields are strings, so a positional form would let
// the two swap unnoticed — which is exactly how the pre-#218 launchers drifted
// into opposite argument orders. Keying makes a swap conspicuous at the call
// site; it does not make it impossible, and no option available at this scope
// does (a named-type conversion would just relocate the swap to the conversion).
// Mirrors updater.Config.
type Config struct {
	// Profile is the Databricks CLI profile. "" resolves to "DEFAULT".
	Profile string
	// CLIPath is the Databricks CLI binary. "" resolves to "databricks".
	// Either way the value is resolved through internal/core/cli, which honors
	// $DATABRICKS_CLI, the MDM-managed path, PATH, and the fallback dirs.
	CLIPath string
}

// resolve applies both defaults and returns an executable CLI path.
func (c Config) resolve() (cliPath, profile string) {
	profile = c.Profile
	if profile == "" {
		profile = "DEFAULT"
	}
	cmdName := c.CLIPath
	if cmdName == "" {
		cmdName = "databricks"
	}
	return cli.ResolveDatabricksCLI(cmdName), profile
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Expiry      string `json:"expiry"` // RFC3339 or Unix timestamp
}

type authEnvResponse struct {
	Env map[string]string `json:"env"`
}

// Fetcher implements tokencache.TokenFetcher against the Databricks CLI.
type Fetcher struct {
	cfg Config
}

// NewFetcher returns a Fetcher for cfg.
func NewFetcher(cfg Config) *Fetcher {
	return &Fetcher{cfg: cfg}
}

// FetchToken runs "databricks auth token --profile <profile>".
func (f *Fetcher) FetchToken(ctx context.Context) (string, time.Time, error) {
	fetchCtx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	cliPath, profile := f.cfg.resolve()
	cmd := exec.CommandContext(fetchCtx, cliPath, "auth", "token", "--profile", profile)
	out, err := cmd.Output()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("databricks auth token failed: %w", err)
	}

	resp, err := parseTokenResponse(out)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to parse token response: %w", err)
	}

	return resp.AccessToken, resp.expiryTime(), nil
}

// NewProvider returns a caching TokenProvider backed by the Databricks CLI.
func NewProvider(cfg Config) *tokencache.TokenProvider {
	return tokencache.NewTokenProvider(NewFetcher(cfg))
}

// DiscoverHost runs "databricks auth env --profile <profile> --output json"
// and extracts DATABRICKS_HOST.
func DiscoverHost(cfg Config) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	cliPath, profile := cfg.resolve()
	cmd := exec.CommandContext(ctx, cliPath, "auth", "env", "--profile", profile, "--output", "json")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("databricks auth env failed: %w", err)
	}

	var resp authEnvResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return "", fmt.Errorf("failed to parse auth env response: %w", err)
	}

	host, ok := resp.Env["DATABRICKS_HOST"]
	if !ok || host == "" {
		return "", fmt.Errorf("DATABRICKS_HOST not found in auth env response")
	}
	return host, nil
}

// GatewayURL joins a Databricks host and a per-tool AI Gateway path.
// The path stays launcher-side: it is the only genuinely per-tool value here,
// and opencode needs two of them.
func GatewayURL(host, path string) string {
	return strings.TrimRight(host, "/") + path
}

// parseTokenResponse decodes the JSON output from "databricks auth token".
func parseTokenResponse(data []byte) (*tokenResponse, error) {
	var resp tokenResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if resp.AccessToken == "" {
		return nil, fmt.Errorf("empty access_token in response")
	}
	return &resp, nil
}

// expiryTime parses the Expiry field, falling back to 55 minutes from now.
func (r *tokenResponse) expiryTime() time.Time {
	if r.Expiry != "" {
		// Try RFC3339 first
		if t, err := time.Parse(time.RFC3339, r.Expiry); err == nil {
			return t
		}
		// Try Unix timestamp (seconds)
		if secs, err := strconv.ParseInt(r.Expiry, 10, 64); err == nil {
			return time.Unix(secs, 0)
		}
	}
	// Conservative default: 55-minute expiry
	return time.Now().Add(55 * time.Minute)
}
