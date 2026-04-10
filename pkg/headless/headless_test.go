package headless

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnsure_ManagedSessionSkips(t *testing.T) {
	t.Setenv("DATABRICKS_CLAUDE_MANAGED", "1")
	// Port 99999 has nothing listening. Without the guard this would fatalf.
	Ensure(Config{
		Port:          99999,
		ManagedEnvVar: "DATABRICKS_CLAUDE_MANAGED",
		LogPrefix:     "test",
	})
}

func TestEnsure_AlreadyHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	port := srv.Listener.Addr().(*net.TCPAddr).Port

	// Should return immediately without trying to start a new proxy.
	Ensure(Config{
		Port:      port,
		LogPrefix: "test",
	})
}
