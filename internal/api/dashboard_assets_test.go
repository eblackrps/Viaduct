package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eblackrps/viaduct/internal/store"
)

func TestServer_Handler_ServesBuiltDashboardAndFallbackRoutes_Expected(t *testing.T) {
	t.Parallel()

	dashboardDir := builtDashboardFixture(t)
	server := NewServer(nil, store.NewMemoryStore(), 0, nil)
	server.SetDashboardDir(dashboardDir)

	handler := server.Handler()

	for _, requestPath := range []string{"/", "/workspaces/alpha"} {
		request := httptest.NewRequest(http.MethodGet, requestPath, nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want %d", requestPath, recorder.Code, http.StatusOK)
		}
		if body := recorder.Body.String(); !strings.Contains(body, "<div id=\"root\"></div>") {
			t.Fatalf("%s body missing dashboard shell: %q", requestPath, body)
		}
		if got := recorder.Header().Get("Cache-Control"); got != "no-cache" {
			t.Fatalf("%s Cache-Control = %q, want no-cache", requestPath, got)
		}
		if got := recorder.Header().Get("Content-Security-Policy"); !strings.Contains(got, "default-src 'self'") {
			t.Fatalf("%s CSP = %q, want dashboard policy", requestPath, got)
		}
	}
}

func TestServer_Handler_ServesDashboardAssetsWithImmutableCache_Expected(t *testing.T) {
	t.Parallel()

	dashboardDir := builtDashboardFixture(t)
	server := NewServer(nil, store.NewMemoryStore(), 0, nil)
	server.SetDashboardDir(dashboardDir)

	request := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control = %q, want immutable asset policy", got)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "console.log('viaduct');") {
		t.Fatalf("asset body = %q, want built asset contents", body)
	}
}

func TestServer_Handler_MissingDashboardAssetReturnsNotFound_Expected(t *testing.T) {
	t.Parallel()

	dashboardDir := builtDashboardFixture(t)
	server := NewServer(nil, store.NewMemoryStore(), 0, nil)
	server.SetDashboardDir(dashboardDir)

	request := httptest.NewRequest(http.MethodGet, "/assets/missing.js", nil)
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestResolveDashboardAssetDir_PrefersBuiltDashboardDir_Expected(t *testing.T) {
	t.Parallel()

	dashboardDir := builtDashboardFixture(t)
	if got := ResolveDashboardAssetDir(dashboardDir); got != dashboardDir {
		t.Fatalf("ResolveDashboardAssetDir(%q) = %q, want %q", dashboardDir, got, dashboardDir)
	}
}

func builtDashboardFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "assets"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte(`<!doctype html><html><body><div id="root"></div><script type="module" src="/assets/app.js"></script></body></html>`), 0o644); err != nil {
		t.Fatalf("WriteFile(index.html) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "assets", "app.js"), []byte(`console.log('viaduct');`), 0o644); err != nil {
		t.Fatalf("WriteFile(app.js) error = %v", err)
	}
	return root
}
