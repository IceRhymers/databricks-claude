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

func TestWrapWithLifecycle_ShutdownClosesDoneCh_AfterPromotion(t *testing.T) {
	// Simulate a health-watcher takeover: IsOwner starts false but PromoteCh is
	// closed before /shutdown arrives, promoting this instance to owner.
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	doneCh := make(chan struct{})
	promoteCh := make(chan struct{})
	handler := WrapWithLifecycle(Config{
		Inner:     inner,
		IsOwner:   false,
		PromoteCh: promoteCh,
		DoneCh:    doneCh,
		LogPrefix: "test",
	})

	// Promote to owner (simulates health.WatchProxy onTakeover callback).
	close(promoteCh)

	// /shutdown should now trigger doneCh because PromoteCh is closed.
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
		t.Error("expected Exiting=true after promotion via PromoteCh")
	}

	select {
	case <-doneCh:
		// expected
	case <-time.After(time.Second):
		t.Error("doneCh was not closed after /shutdown following promotion")
	}
}

func TestWrapWithLifecycle_ShutdownNoExitBeforePromotion(t *testing.T) {
	// Before promotion, /shutdown from a non-owner should not close doneCh
	// (when a refcount path is in use — here we use no refcount so it always
	// shuts down; the test verifies that without PromoteCh the non-owner path
	// with RefcountPath set does not exit).
	// We test the inverse: no promoteCh, IsOwner false, RefcountPath set =>
	// /shutdown returns Exiting=false and does NOT close doneCh.
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	doneCh := make(chan struct{})
	// Use a temp file as a stand-in refcount path (we don't need real refcount
	// file logic — any non-empty path triggers the refcount branch, and
	// Release will return 0 for a missing file, so exiting = 0 && isOwner()).
	handler := WrapWithLifecycle(Config{
		Inner:        inner,
		IsOwner:      false,
		RefcountPath: "/tmp/nonexistent-refcount-test-file",
		DoneCh:       doneCh,
		LogPrefix:    "test",
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
	if resp.Exiting {
		t.Error("expected Exiting=false when not owner and no PromoteCh")
	}

	select {
	case <-doneCh:
		t.Error("doneCh should not be closed when non-owner without PromoteCh")
	case <-time.After(50 * time.Millisecond):
		// expected: doneCh stays open
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
