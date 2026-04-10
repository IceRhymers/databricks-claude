package lifecycle

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWrapWithLifecycle_HealthDelegates(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	doneCh := make(chan struct{})
	handler := WrapWithLifecycle(Config{
		Inner:     inner,
		DoneCh:    doneCh,
		LogPrefix: "test",
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /health returned %d, want 200", rec.Code)
	}
}

func TestWrapWithLifecycle_ShutdownClosesDoneCh_NoRefcount(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	doneCh := make(chan struct{})
	handler := WrapWithLifecycle(Config{
		Inner:     inner,
		DoneCh:    doneCh,
		LogPrefix: "test",
	})

	req := httptest.NewRequest(http.MethodPost, "/shutdown", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("POST /shutdown returned %d, want 200", rec.Code)
	}

	var resp shutdownResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode shutdown response: %v", err)
	}
	if !resp.Exiting {
		t.Error("expected Exiting=true when no refcount path")
	}

	select {
	case <-doneCh:
		// expected
	case <-time.After(time.Second):
		t.Error("doneCh was not closed after /shutdown")
	}
}

func TestWrapWithLifecycle_ShutdownMethodNotAllowed(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	doneCh := make(chan struct{})
	handler := WrapWithLifecycle(Config{
		Inner:     inner,
		DoneCh:    doneCh,
		LogPrefix: "test",
	})

	req := httptest.NewRequest(http.MethodGet, "/shutdown", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /shutdown returned %d, want 405", rec.Code)
	}
}

func TestWrapWithLifecycle_APIKeyEnforced(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	doneCh := make(chan struct{})
	handler := WrapWithLifecycle(Config{
		Inner:     inner,
		APIKey:    "secret123",
		DoneCh:    doneCh,
		LogPrefix: "test",
	})

	// No auth header.
	req := httptest.NewRequest(http.MethodPost, "/shutdown", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("POST /shutdown without key returned %d, want 401", rec.Code)
	}

	// Correct auth header.
	req = httptest.NewRequest(http.MethodPost, "/shutdown", nil)
	req.Header.Set("Authorization", "Bearer secret123")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("POST /shutdown with correct key returned %d, want 200", rec.Code)
	}
}
