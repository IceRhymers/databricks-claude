<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-04-06 | Updated: 2026-04-06 -->

# proxy

## Purpose
HTTP/WebSocket reverse proxy with two routes: inference (`/`) and OTEL (`/otel/`). Injects fresh Databricks OAuth tokens per-request, supports optional API key authentication, TLS listeners, WebSocket upgrades (for databricks-codex), and includes security checks and log sanitization.

## Key Files

| File | Description |
|------|-------------|
| `proxy.go` | `Config` struct, `TokenSource` interface, `NewServer` (mux with inference + OTEL routes), `Start` (bind listener with optional TLS), `RecoveryHandler`, `requireAPIKey` middleware, `handleWebSocket` (bidirectional pipe for Codex), `ValidateTLSConfig` |
| `checks.go` | `SecurityChecks` -- startup warnings for running as root or permissive umask |
| `sanitize.go` | `sanitizeLogOutput` -- redacts Bearer tokens, `dapi-*` PATs, and `x-api-key` values from log strings |
| `proxy_test.go` | Comprehensive tests for inference routing, OTEL routing, token injection, WebSocket handling, API key auth, error responses, panic recovery |
| `checks_test.go` | Tests for security check warnings |
| `sanitize_test.go` | Tests for sensitive value redaction patterns |

## For AI Agents

### Working In This Directory
- The `TokenSource` interface has a single method: `Token(ctx) (string, error)`. The root package's `TokenProvider` satisfies it.
- **Two proxy routes**: `/` for inference (AI Gateway), `/otel/` for telemetry. Path algebra prepends the upstream's base path.
- **WebSocket support** exists for databricks-codex (Codex CLI), not for Claude Code (which uses HTTP+SSE). The `isWebSocketUpgrade` check is passive -- zero overhead for non-upgrade requests.
- `FlushInterval: -1` on both reverse proxies enables streaming (SSE).
- All routes are wrapped in `RecoveryHandler` for panic safety (returns HTTP 502).
- `requireAPIKey` middleware wraps the entire mux when `APIKey` is non-empty.
- `UCMetricsTable` may be empty (e.g., databricks-codex) -- in that case the `X-Databricks-UC-Table-Name` header is omitted for metrics paths.
- OTEL route distinguishes logs vs metrics by checking if the path contains `/v1/logs`.

### Testing Requirements
- Run: `go test ./pkg/proxy/...`
- Tests use `httptest.NewServer` and mock `TokenSource` implementations.
- `getuid` in `checks.go` is overridable for testing root-detection logic.

### Common Patterns
- Token injection: both `Authorization: Bearer` and `x-api-key` headers are set on every proxied request.
- Verbose logging: error responses (4xx+) are logged with first 500 chars, sanitized via `sanitizeLogOutput`.

## Dependencies

### Internal
- None

### External
- `net/http/httputil`, `crypto/tls`, `net`, `net/url` (stdlib)

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
