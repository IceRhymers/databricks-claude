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

// TestListServicesBootstrapsOrgID verifies ListServices recovers from the
// header-less "Invalid MetastoreId" 400 the model-services LIST endpoint returns
// by reading the echoed X-Databricks-Org-Id off the rejection and retrying once
// with it. Without the bootstrap, discovery fails with a 400 on this workspace.
func TestListServicesBootstrapsOrgID(t *testing.T) {
	const wantOrg = "7474650869313380"
	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		// The gateway echoes the caller's org id on every response, including
		// the 400 rejection.
		w.Header().Set("X-Databricks-Org-Id", wantOrg)
		if r.Header.Get("X-Databricks-Org-Id") == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error_code":"MALFORMED_REQUEST","message":"Invalid MetastoreId: "}`))
			return
		}
		if got := r.Header.Get("X-Databricks-Org-Id"); got != wantOrg {
			t.Errorf("retry sent org id %q, want %q", got, wantOrg)
		}
		_ = json.NewEncoder(w).Encode(wireListResponse{
			ModelServices: []wireService{{
				Name:              "model-services/system.ai.claude-opus-4-8",
				SupportedAPITypes: messages,
			}},
		})
	}))
	defer srv.Close()

	services, err := ListServices(context.Background(), NewClient(), srv.URL, "tok")
	if err != nil {
		t.Fatalf("ListServices: %v", err)
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts (reject then bootstrapped retry), got %d", attempts)
	}
	if len(services) != 1 || services[0].FQN != "system.ai.claude-opus-4-8" {
		t.Errorf("unexpected services: %+v", services)
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
