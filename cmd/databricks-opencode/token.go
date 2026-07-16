package main

import "github.com/IceRhymers/databricks-agents/internal/core/dbxauth"

// opencode has two upstreams, which is why gateway paths stay launcher-side:
// a single per-tool path value cannot represent this. The token/auth machinery
// lives in internal/core/dbxauth (#218).
const (
	gatewayPath       = "/ai-gateway/anthropic"
	geminiGatewayPath = "/ai-gateway/gemini/v1beta"
)

// ConstructGatewayURL builds the AI Gateway URL for the OpenCode proxy endpoint.
// Format: {host}/ai-gateway/anthropic
// Uses /anthropic route with @ai-sdk/anthropic (Messages API).
func ConstructGatewayURL(host string) string {
	return dbxauth.GatewayURL(host, gatewayPath)
}

// ConstructGeminiGatewayURL builds the Databricks AI Gateway URL for the
// Gemini Native upstream. Format: {host}/ai-gateway/gemini/v1beta
//
// The /v1beta segment is included so that the proxy.UpstreamRoute path
// algebra resolves correctly: incoming /v1beta/models/<m>:generateContent
// has its /v1beta prefix stripped, then this base path is prepended,
// yielding /ai-gateway/gemini/v1beta/models/<m>:generateContent —
// the verified Databricks Gemini endpoint.
func ConstructGeminiGatewayURL(host string) string {
	return dbxauth.GatewayURL(host, geminiGatewayPath)
}
