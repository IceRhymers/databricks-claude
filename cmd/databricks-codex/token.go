package main

import "github.com/IceRhymers/databricks-agents/internal/core/dbxauth"

// gatewayPath is codex's AI Gateway suffix — the OpenAI-shaped route, distinct
// from claude's /ai-gateway/anthropic. Per-tool value; the token/auth machinery
// lives in internal/core/dbxauth (#218).
const gatewayPath = "/ai-gateway/openai/v1"

// ConstructGatewayURL builds the AI Gateway URL for the Codex proxy endpoint.
// Format: {host}/ai-gateway/openai/v1
func ConstructGatewayURL(host string) string {
	return dbxauth.GatewayURL(host, gatewayPath)
}
