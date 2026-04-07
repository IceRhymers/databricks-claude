package main

import (
	"net"
	"net/http"

	"github.com/IceRhymers/databricks-claude/pkg/proxy"
	"github.com/IceRhymers/databricks-claude/pkg/tokencache"
)

// ProxyConfig holds the configuration for the proxy server.
type ProxyConfig struct {
	InferenceUpstream string
	OTELUpstream      string
	UCMetricsTable    string
	UCLogsTable       string
	TokenProvider     *tokencache.TokenProvider
	Verbose           bool
	APIKey            string
	TLSCertFile       string
	TLSKeyFile        string
	ToolName          string // reported by /health endpoint
	Version           string // reported by /health endpoint
}

// recoveryHandler wraps h with panic recovery, returning 502 on panic.
func recoveryHandler(next http.Handler) http.Handler {
	return proxy.RecoveryHandler(next)
}

// NewProxyServer returns an http.Handler that routes requests to the
// inference upstream (default) and the OTEL upstream (/otel/).
func NewProxyServer(config *ProxyConfig) http.Handler {
	return proxy.NewServer(&proxy.Config{
		InferenceUpstream: config.InferenceUpstream,
		OTELUpstream:      config.OTELUpstream,
		UCMetricsTable:    config.UCMetricsTable,
		UCLogsTable:       config.UCLogsTable,
		TokenSource:       config.TokenProvider,
		Verbose:           config.Verbose,
		APIKey:            config.APIKey,
		ToolName:          config.ToolName,
		Version:           config.Version,
	})
}

// ServeProxy starts the proxy on the given listener.
// When config.TLSCertFile and config.TLSKeyFile are both set, the listener serves TLS.
func ServeProxy(config *ProxyConfig, handler http.Handler, ln net.Listener) error {
	_, err := proxy.Serve(ln, handler, config.TLSCertFile, config.TLSKeyFile)
	return err
}

// StartProxy binds to 127.0.0.1:0, starts serving, and returns the listener.
// Used in tests and as a fallback when ServeProxy is not applicable.
func StartProxy(config *ProxyConfig, handler http.Handler) (net.Listener, error) {
	return proxy.Start(handler, config.TLSCertFile, config.TLSKeyFile)
}
