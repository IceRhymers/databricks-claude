package main

import "github.com/IceRhymers/databricks-agents/internal/core/dbxauth"

// gatewayPath is claude's AI Gateway suffix. Per-tool value; the token/auth
// machinery itself lives in internal/core/dbxauth (#218).
const gatewayPath = "/ai-gateway/anthropic"

// ConstructGatewayURL returns the AI Gateway base URL for the given Databricks host.
func ConstructGatewayURL(host string) string {
	return dbxauth.GatewayURL(host, gatewayPath)
}
