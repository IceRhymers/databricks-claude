package proxy

import (
	"bytes"
	"context"
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
type Config struct {
	InferenceUpstream string
	OTELUpstream      string
	UCMetricsTable    string
	UCLogsTable       string
	TokenSource       TokenSource
	Verbose           bool
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

// NewServer returns an http.Handler that routes requests to the
// inference upstream (default) and the OTEL upstream (/otel/).
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

	// OTEL proxy — /otel/ route
	otelProxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			token, err := config.TokenSource.Token(req.Context())
			if err != nil {
				log.Printf("databricks-claude: token fetch error (otel): %v", err)
			}
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("x-api-key", token)

			// Pick the correct UC table based on whether this is a logs or metrics request.
			ucTable := config.UCMetricsTable
			if strings.Contains(req.URL.Path, "/v1/logs") {
				ucTable = config.UCLogsTable
			}
			req.Header.Set("X-Databricks-UC-Table-Name", ucTable)

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
	mux.Handle("/", RecoveryHandler(inferenceProxy))

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
