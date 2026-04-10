package headless

import (
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
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
		Scheme:    "http",
		LogPrefix: "test",
	})
}

func TestBuildArgs_NoTLS(t *testing.T) {
	args := buildArgs(Config{Port: 8080})
	expected := []string{"--headless", "--port=8080"}
	if !reflect.DeepEqual(args, expected) {
		t.Errorf("expected %v, got %v", expected, args)
	}
}

func TestBuildArgs_WithTLSCert(t *testing.T) {
	args := buildArgs(Config{Port: 8080, TLSCert: "/path/to/cert.pem"})
	expected := []string{"--headless", "--port=8080", "--tls-cert=/path/to/cert.pem"}
	if !reflect.DeepEqual(args, expected) {
		t.Errorf("expected %v, got %v", expected, args)
	}
}

func TestBuildArgs_WithTLSKey(t *testing.T) {
	args := buildArgs(Config{Port: 8080, TLSKey: "/path/to/key.pem"})
	expected := []string{"--headless", "--port=8080", "--tls-key=/path/to/key.pem"}
	if !reflect.DeepEqual(args, expected) {
		t.Errorf("expected %v, got %v", expected, args)
	}
}

func TestBuildArgs_WithTLSCertAndKey(t *testing.T) {
	args := buildArgs(Config{
		Port:    8443,
		TLSCert: "/cert.pem",
		TLSKey:  "/key.pem",
	})
	expected := []string{"--headless", "--port=8443", "--tls-cert=/cert.pem", "--tls-key=/key.pem"}
	if !reflect.DeepEqual(args, expected) {
		t.Errorf("expected %v, got %v", expected, args)
	}
}
