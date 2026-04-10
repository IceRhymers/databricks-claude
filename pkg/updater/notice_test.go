package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPrintUpdateNotice_NoUpdate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(githubRelease{
			TagName:    "v0.10.0",
			HTMLURL:    "https://github.com/test/repo/releases/tag/v0.10.0",
			Prerelease: false,
		})
	}))
	defer srv.Close()

	origClient := http.DefaultClient
	http.DefaultClient = srv.Client()
	http.DefaultClient.Transport = rewriteTransport{target: srv.URL}
	defer func() { http.DefaultClient = origClient }()

	cacheDir := t.TempDir()

	// Capture stderr.
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	PrintUpdateNotice(Config{
		RepoSlug:       "test/repo",
		CurrentVersion: "0.10.0",
		BinaryName:     "mybin",
		CacheFile:      filepath.Join(cacheDir, "cache.json"),
	})

	w.Close()
	os.Stderr = oldStderr

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if strings.Contains(output, "update available") {
		t.Errorf("unexpected update notice: %s", output)
	}
}

func TestPrintUpdateNotice_UpdateAvailable(t *testing.T) {
	assetName := fmt.Sprintf("mybin-%s-%s", runtime.GOOS, runtime.GOARCH)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(githubRelease{
			TagName:    "v0.11.0",
			HTMLURL:    "https://github.com/test/repo/releases/tag/v0.11.0",
			Prerelease: false,
			Assets: []githubAsset{
				{Name: assetName, BrowserDownloadURL: "https://dl.example.com/mybin"},
			},
		})
	}))
	defer srv.Close()

	origClient := http.DefaultClient
	http.DefaultClient = srv.Client()
	http.DefaultClient.Transport = rewriteTransport{target: srv.URL}
	defer func() { http.DefaultClient = origClient }()

	cacheDir := t.TempDir()

	// Capture stderr.
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	PrintUpdateNotice(Config{
		RepoSlug:       "test/repo",
		CurrentVersion: "0.10.0",
		BinaryName:     "mybin",
		CacheFile:      filepath.Join(cacheDir, "cache.json"),
	})

	w.Close()
	os.Stderr = oldStderr

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "update available") {
		t.Errorf("expected update notice, got: %q", output)
	}
	if !strings.Contains(output, "mybin") {
		t.Errorf("expected binary name in output, got: %q", output)
	}
}
