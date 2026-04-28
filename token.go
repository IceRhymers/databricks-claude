package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/IceRhymers/databricks-claude/pkg/tokencache"
)

// fallbackCLIDirs lists install locations to probe when "databricks" is not on
// PATH. Order matters: most-likely first. GUI-launched subprocesses (e.g.
// Claude Desktop invoking the credential helper) inherit launchd's minimal
// PATH (/usr/bin:/bin:/usr/sbin:/sbin), which omits all of these.
var fallbackCLIDirs = []string{
	"/usr/local/bin",
	"/opt/homebrew/bin",
	"/opt/homebrew/sbin",
	".local/bin", // resolved against $HOME
	"go/bin",     // resolved against $HOME
	"bin",        // resolved against $HOME
}

// resolveDatabricksCLI returns an executable path for the Databricks CLI.
// Lookup order:
//  1. Absolute or path-qualified cmdName → returned unchanged (back-compat for tests).
//  2. $DATABRICKS_CLI env override, if set and executable.
//  3. exec.LookPath(cmdName), which honors the inherited PATH.
//  4. A scan of common install dirs (/usr/local/bin, /opt/homebrew/bin, ~/.local/bin, ~/go/bin, ~/bin).
//
// If none match, cmdName is returned unchanged so the eventual exec error
// surfaces with its original message.
func resolveDatabricksCLI(cmdName string) string {
	if cmdName == "" {
		cmdName = "databricks"
	}
	if filepath.IsAbs(cmdName) || filepath.Base(cmdName) != cmdName {
		return cmdName
	}
	if override := os.Getenv("DATABRICKS_CLI"); override != "" {
		if isExecutableFile(override) {
			return override
		}
	}
	if p, err := exec.LookPath(cmdName); err == nil {
		return p
	}
	home, _ := os.UserHomeDir()
	for _, dir := range fallbackCLIDirs {
		if !filepath.IsAbs(dir) {
			if home == "" {
				continue
			}
			dir = filepath.Join(home, dir)
		}
		candidate := filepath.Join(dir, cmdName)
		if isExecutableFile(candidate) {
			return candidate
		}
	}
	return cmdName
}

func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode()&0o111 != 0
}

// TokenProvider is an alias to the pkg type for backward compatibility.
type TokenProvider = tokencache.TokenProvider

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Expiry      string `json:"expiry"` // RFC3339 or Unix timestamp
}

// databricksFetcher implements tokencache.TokenFetcher using the Databricks CLI.
type databricksFetcher struct {
	profile string
	cmdName string
}

func (f *databricksFetcher) FetchToken(ctx context.Context) (string, time.Time, error) {
	fetchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(fetchCtx, resolveDatabricksCLI(f.cmdName), "auth", "token", "--profile", f.profile)
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

// NewTokenProvider creates a new TokenProvider backed by the Databricks CLI.
// cmdName defaults to "databricks" if empty.
func NewTokenProvider(profile, cmdName string) *TokenProvider {
	if cmdName == "" {
		cmdName = "databricks"
	}
	return tokencache.NewTokenProvider(&databricksFetcher{
		profile: profile,
		cmdName: cmdName,
	})
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

type authEnvResponse struct {
	Env map[string]string `json:"env"`
}

// DiscoverHost calls "databricks auth env --profile <profile> --output json"
// and extracts the DATABRICKS_HOST value from the response.
func DiscoverHost(profile, cmdName string) (string, error) {
	if cmdName == "" {
		cmdName = "databricks"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, resolveDatabricksCLI(cmdName), "auth", "env", "--profile", profile, "--output", "json")
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

// ResolveWorkspaceID calls the SCIM /Me endpoint and extracts x-databricks-org-id from response headers.
func ResolveWorkspaceID(host, token string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest(http.MethodGet, host+"/api/2.0/preview/scim/v2/Me", nil)
	if err != nil {
		return "", fmt.Errorf("failed to build SCIM request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("SCIM request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("SCIM request returned status %d", resp.StatusCode)
	}

	orgID := resp.Header.Get("x-databricks-org-id")
	if orgID == "" {
		return "", fmt.Errorf("x-databricks-org-id header not present in SCIM response")
	}
	return orgID, nil
}

// ConstructGatewayURL builds the AI Gateway URL using the workspace ID, falling back
// to the serving-endpoints path if workspace ID resolution fails.
func ConstructGatewayURL(host, token string) string {
	workspaceID, err := ResolveWorkspaceID(host, token)
	if err != nil {
		log.Printf("workspace ID resolution failed, falling back to serving-endpoints: %v", err)
		return host + "/serving-endpoints/anthropic"
	}
	gatewayURL := "https://" + workspaceID + ".ai-gateway.cloud.databricks.com/anthropic"
	log.Printf("resolved workspace ID %s, using AI Gateway URL: %s", workspaceID, gatewayURL)
	return gatewayURL
}
