package proxy

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// TokenSource provides tokens for upstream authentication.
type TokenSource interface {
	Token(ctx context.Context) (string, error)
}

// Config holds the configuration for the proxy server.
//
// WebSocket note: databricks-claude (Claude Code) uses HTTP with SSE for
// streaming and never issues WebSocket upgrades. databricks-codex (Codex CLI)
// uses WebSocket for its inference connection. The proxy handles both
// transparently — WebSocket upgrades are detected per-request and routed
// through handleWebSocket; plain HTTP falls through to httputil.ReverseProxy.
// Centralising this here avoids maintaining a forked proxy in databricks-codex.
type Config struct {
	InferenceUpstream string
	OTELUpstream      string
	// UCMetricsTable is the Unity Catalog table for OTEL metrics.
	// Leave empty if the caller does not emit metrics (e.g. databricks-codex,
	// which has no native metrics support). When empty the
	// X-Databricks-UC-Table-Name header is omitted for metrics requests.
	UCMetricsTable string
	UCLogsTable    string
	TokenSource    TokenSource
	Verbose        bool
}

// RecoveryHandler wraps h with panic recovery, returning 502 on panic.
func RecoveryHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("databricks-claude: proxy panic recovered: %v", err)
				http.Error(w, "Internal proxy error", http.StatusBadGateway)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// isWebSocketUpgrade returns true if the request is a WebSocket upgrade.
//
// Claude Code never sends WebSocket upgrades (it uses SSE over HTTP). This
// check exists to support databricks-codex, whose upstream uses WebSocket for
// inference. The check is passive — non-upgrade requests fall through to the
// normal httputil.ReverseProxy path with zero overhead.
func isWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

// handleWebSocket proxies a WebSocket upgrade request to the upstream,
// injecting a fresh Bearer token. After a successful 101 response it pipes
// data bidirectionally between client and upstream until either side closes.
//
// Why this exists here: Codex (databricks-codex) connects to its inference
// endpoint over WebSocket rather than HTTP/SSE. The token injection and
// bidirectional piping logic is centralised in this shared package so
// databricks-codex can import it rather than maintaining its own copy.
func handleWebSocket(w http.ResponseWriter, r *http.Request, upstream *url.URL, config *Config) {
	token, err := config.TokenSource.Token(r.Context())
	if err != nil {
		log.Printf("databricks-claude: ws token fetch error: %v", err)
		http.Error(w, "token fetch failed", http.StatusBadGateway)
		return
	}

	// Build upstream path: prepend upstream base path to request path.
	basePath := strings.TrimRight(upstream.Path, "/")
	upstreamPath := basePath + r.URL.Path

	// Determine host:port for dialing.
	dialHost := upstream.Host
	useTLS := upstream.Scheme == "https" || upstream.Scheme == "wss"
	if !strings.Contains(dialHost, ":") {
		if useTLS {
			dialHost += ":443"
		} else {
			dialHost += ":80"
		}
	}

	if config.Verbose {
		log.Printf("databricks-claude: ws upgrade → %s%s (tls=%v)", upstream.Host, upstreamPath, useTLS)
	}

	// Dial upstream.
	var upstreamConn net.Conn
	if useTLS {
		upstreamConn, err = tls.Dial("tcp", dialHost, &tls.Config{
			ServerName: upstream.Hostname(),
		})
	} else {
		upstreamConn, err = net.Dial("tcp", dialHost)
	}
	if err != nil {
		log.Printf("databricks-claude: ws dial failed: %v", err)
		http.Error(w, "upstream dial failed", http.StatusBadGateway)
		return
	}
	defer upstreamConn.Close()

	// Build the HTTP upgrade request for the upstream.
	upgradeReq, err := http.NewRequest(r.Method, (&url.URL{
		Path:     upstreamPath,
		RawQuery: r.URL.RawQuery,
	}).RequestURI(), nil)
	if err != nil {
		log.Printf("databricks-claude: ws build request failed: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Copy original headers, then override auth + host.
	for k, vv := range r.Header {
		for _, v := range vv {
			upgradeReq.Header.Add(k, v)
		}
	}
	upgradeReq.Header.Set("Authorization", "Bearer "+token)
	upgradeReq.Header.Set("x-api-key", token)
	upgradeReq.Header.Set("x-databricks-use-coding-agent-mode", "true")
	upgradeReq.Host = upstream.Host
	upgradeReq.Header.Set("Host", upstream.Host)

	// Send the upgrade request to the upstream.
	if err := upgradeReq.Write(upstreamConn); err != nil {
		log.Printf("databricks-claude: ws write upgrade failed: %v", err)
		http.Error(w, "upstream write failed", http.StatusBadGateway)
		return
	}

	// Read the upstream response.
	br := bufio.NewReader(upstreamConn)
	upstreamResp, err := http.ReadResponse(br, upgradeReq)
	if err != nil {
		log.Printf("databricks-claude: ws read response failed: %v", err)
		http.Error(w, "upstream response failed", http.StatusBadGateway)
		return
	}

	// If the upstream did not switch protocols, forward the error to the client.
	if upstreamResp.StatusCode != http.StatusSwitchingProtocols {
		body, _ := io.ReadAll(upstreamResp.Body)
		upstreamResp.Body.Close()
		log.Printf("databricks-claude: ws upgrade rejected: %d %s", upstreamResp.StatusCode, string(body))
		w.WriteHeader(upstreamResp.StatusCode)
		w.Write(body) //nolint:errcheck
		return
	}

	// Hijack the client connection.
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		log.Printf("databricks-claude: ResponseWriter does not support hijacking")
		http.Error(w, "hijack not supported", http.StatusInternalServerError)
		return
	}
	clientConn, clientBuf, err := hijacker.Hijack()
	if err != nil {
		log.Printf("databricks-claude: ws hijack failed: %v", err)
		return
	}
	defer clientConn.Close()

	// Forward the 101 Switching Protocols response to the client.
	if err := upstreamResp.Write(clientConn); err != nil {
		log.Printf("databricks-claude: ws write 101 to client failed: %v", err)
		return
	}

	if config.Verbose {
		log.Printf("databricks-claude: ws connected, piping data")
	}

	// Bidirectional pipe. When either direction finishes, close both.
	done := make(chan struct{}, 2)

	// Client → Upstream
	go func() {
		defer func() { done <- struct{}{} }()
		// Flush any buffered data from the client's bufio reader first.
		if clientBuf != nil && clientBuf.Reader.Buffered() > 0 {
			buffered := make([]byte, clientBuf.Reader.Buffered())
			n, _ := clientBuf.Read(buffered)
			if n > 0 {
				upstreamConn.Write(buffered[:n]) //nolint:errcheck
			}
		}
		io.Copy(upstreamConn, clientConn) //nolint:errcheck
	}()

	// Upstream → Client
	go func() {
		defer func() { done <- struct{}{} }()
		// Flush any buffered data from the upstream's bufio reader first.
		if br.Buffered() > 0 {
			buffered := make([]byte, br.Buffered())
			n, _ := br.Read(buffered)
			if n > 0 {
				clientConn.Write(buffered[:n]) //nolint:errcheck
			}
		}
		io.Copy(clientConn, upstreamConn) //nolint:errcheck
	}()

	// Wait for one direction to finish, then close both.
	<-done

	if config.Verbose {
		log.Printf("databricks-claude: ws connection closed")
	}
}

// NewServer returns an http.Handler that routes requests to the
// inference upstream (default) and the OTEL upstream (/otel/).
//
// Inference route behaviour:
//   - WebSocket upgrade requests (used by databricks-codex / Codex CLI) are
//     handled via handleWebSocket with bidirectional piping and fresh token injection.
//   - All other requests (used by databricks-claude / Claude Code, which speaks
//     HTTP + SSE) are handled by httputil.ReverseProxy.
//
// OTEL route (/otel/):
//   - /v1/logs paths → UCLogsTable header
//   - all other paths → UCMetricsTable header (omitted if UCMetricsTable is empty)
func NewServer(config *Config) http.Handler {
	mux := http.NewServeMux()

	inferenceUpstream, err := url.Parse(config.InferenceUpstream)
	if err != nil {
		log.Fatalf("databricks-claude: invalid InferenceUpstream %q: %v", config.InferenceUpstream, err)
	}

	otelUpstream, err := url.Parse(config.OTELUpstream)
	if err != nil {
		log.Fatalf("databricks-claude: invalid OTELUpstream %q: %v", config.OTELUpstream, err)
	}

	// Inference proxy — default route
	inferenceProxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			token, err := config.TokenSource.Token(req.Context())
			if err != nil {
				// Log the error but let the upstream return an auth failure rather
				// than crashing; the empty bearer will be rejected by the upstream.
				log.Printf("databricks-claude: token fetch error: %v", err)
			}
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("x-api-key", token) // Anthropic SDK sends x-api-key; overwrite the "proxy-managed" placeholder
			req.Header.Set("x-databricks-use-coding-agent-mode", "true")

			req.URL.Scheme = inferenceUpstream.Scheme
			req.URL.Host = inferenceUpstream.Host
			req.Host = inferenceUpstream.Host // Override Host header — upstream rejects localhost
			// Prepend the upstream base path to the incoming request path.
			basePath := strings.TrimRight(inferenceUpstream.Path, "/")
			req.URL.Path = basePath + req.URL.Path
			req.URL.RawPath = ""

			if config.Verbose {
				log.Printf("databricks-claude: inference → %s %s%s", req.Method, req.URL.Host, req.URL.Path)
			}
		},
		ModifyResponse: func(resp *http.Response) error {
			if config.Verbose && resp.StatusCode >= 400 {
				body, err := io.ReadAll(resp.Body)
				if err == nil {
					// Log first 500 chars of error response
					snippet := string(body)
					if len(snippet) > 500 {
						snippet = snippet[:500] + "..."
					}
					log.Printf("databricks-claude: upstream error %d: %s", resp.StatusCode, snippet)
					// Put the body back so the caller still gets it
					resp.Body = io.NopCloser(bytes.NewReader(body))
				}
			}
			return nil
		},
		FlushInterval: -1,
	}

	// Wrap inference proxy with WebSocket upgrade detection.
	// Claude Code never sends WebSocket upgrades; this branch exists for
	// databricks-codex (Codex CLI), which uses WebSocket for inference.
	inferenceHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isWebSocketUpgrade(r) {
			handleWebSocket(w, r, inferenceUpstream, config)
			return
		}
		inferenceProxy.ServeHTTP(w, r)
	})

	// OTEL proxy — /otel/ route
	otelProxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			token, err := config.TokenSource.Token(req.Context())
			if err != nil {
				log.Printf("databricks-claude: token fetch error (otel): %v", err)
			}
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("x-api-key", token)

			// Pick the correct UC table based on whether this is a logs or metrics
			// request. UCMetricsTable may be empty (e.g. databricks-codex has no
			// native metrics); in that case the header is omitted for metrics paths.
			ucTable := config.UCLogsTable
			if !strings.Contains(req.URL.Path, "/v1/logs") {
				ucTable = config.UCMetricsTable
			}
			if ucTable != "" {
				req.Header.Set("X-Databricks-UC-Table-Name", ucTable)
			}

			// Strip the /otel prefix and prepend the upstream base path.
			stripped := strings.TrimPrefix(req.URL.Path, "/otel")
			basePath := strings.TrimRight(otelUpstream.Path, "/")
			req.URL.Scheme = otelUpstream.Scheme
			req.URL.Host = otelUpstream.Host
			req.Host = otelUpstream.Host
			req.URL.Path = basePath + stripped
			req.URL.RawPath = ""

			if config.Verbose {
				log.Printf("databricks-claude: otel → %s %s%s", req.Method, req.URL.Host, req.URL.Path)
			}
		},
		ModifyResponse: func(resp *http.Response) error {
			if config.Verbose || resp.StatusCode >= 400 {
				body, err := io.ReadAll(resp.Body)
				if err == nil {
					snippet := string(body)
					if len(snippet) > 500 {
						snippet = snippet[:500] + "..."
					}
					if resp.StatusCode >= 400 {
						log.Printf("databricks-claude: otel upstream error %d: %s", resp.StatusCode, snippet)
					} else {
						log.Printf("databricks-claude: otel ← %d (%d bytes)", resp.StatusCode, len(body))
					}
					resp.Body = io.NopCloser(bytes.NewReader(body))
				}
			}
			return nil
		},
		FlushInterval: -1,
	}

	mux.Handle("/otel/", RecoveryHandler(otelProxy))
	mux.Handle("/", RecoveryHandler(inferenceHandler))

	return mux
}

// Start binds to 127.0.0.1:0, starts serving, and returns the listener.
// Callers read l.Addr() to discover the assigned port.
func Start(handler http.Handler) (net.Listener, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	go func() {
		if err := http.Serve(l, handler); err != nil {
			// http.Serve returns when the listener is closed; that is expected
			// during shutdown and not worth logging as an error.
			log.Printf("databricks-claude: proxy stopped: %v", err)
		}
	}()
	return l, nil
}
