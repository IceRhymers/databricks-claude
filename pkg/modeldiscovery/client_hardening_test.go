package modeldiscovery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestNewClientRefusesCrossHostRedirect verifies the discovery client refuses a
// redirect to a different host, so the OAuth bearer token can never be replayed
// against an attacker-chosen host.
func TestNewClientRefusesCrossHostRedirect(t *testing.T) {
	// other is the redirect target on a different host; it must never be hit.
	var otherHit bool
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		otherHit = true
		w.WriteHeader(http.StatusOK)
	}))
	defer other.Close()

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL+r.URL.Path, http.StatusFound)
	}))
	defer origin.Close()

	_, err := ListServices(context.Background(), NewClient(), origin.URL, "tok")
	if err == nil {
		t.Fatal("expected error on cross-host redirect, got nil")
	}
	if !strings.Contains(err.Error(), "cross-host redirect") {
		t.Errorf("expected cross-host redirect error, got: %v", err)
	}
	if otherHit {
		t.Error("redirect target on a different host was reached — token could leak")
	}
}

// TestGetServiceEscapesFQN verifies a crafted service FQN is escaped as a single
// path segment rather than traversing to another API path.
func TestGetServiceEscapesFQN(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		_ = json.NewEncoder(w).Encode(wireService{FullName: "x.y.z", SupportedAPITypes: messages})
	}))
	defer srv.Close()

	// A malicious FQN attempting path traversal / a query break-out.
	_, err := GetService(context.Background(), NewClient(), srv.URL, "tok", "a/../../secret?x=1")
	if err != nil {
		t.Fatalf("GetService: %v", err)
	}
	// The escaped path must stay under the model-services collection — no raw
	// "/../" traversal and no unescaped "?" splitting off a query.
	if !strings.HasPrefix(gotPath, modelServicesPath+"/") {
		t.Errorf("request escaped the model-services path: %q", gotPath)
	}
	if strings.Contains(gotPath, "/../") {
		t.Errorf("path traversal not escaped: %q", gotPath)
	}
}
