package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/fox-toolkit/fox"
)

// mountedRouter mounts dir at route on a router configured like the production
// public router and returns it.
func mountedRouter(t *testing.T, dir, route string) *fox.Router {
	t.Helper()

	ctrl, controlRouter := setupControl(t)
	if w := postMount(t, controlRouter, dir, route); w.Code != http.StatusCreated {
		t.Fatalf("mount %s failed: %d %s", route, w.Code, w.Body.String())
	}
	return ctrl.router
}

func serveDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("root"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "page.html"), []byte("page"), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func doRequest(t *testing.T, router *fox.Router, method, target string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, target, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestServeMountedDirectory(t *testing.T) {
	router := mountedRouter(t, serveDir(t), "/static")

	tests := []struct {
		name   string
		method string
		target string
		body   string
	}{
		{"index through trailing slash", http.MethodGet, "/static/", "root"},
		{"nested file", http.MethodGet, "/static/sub/page.html", "page"},
		{"head request", http.MethodHead, "/static/sub/page.html", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := doRequest(t, router, tc.method, tc.target)
			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", w.Code)
			}
			if got := w.Body.String(); got != tc.body {
				t.Errorf("expected body %q, got %q", tc.body, got)
			}
		})
	}
}

// TestServePathNormalization pins the router normalization options: a missing
// trailing slash, consecutive slashes and dot segments redirect to the
// canonical path, and a ".." escaping above the mount root is rejected.
func TestServePathNormalization(t *testing.T) {
	router := mountedRouter(t, serveDir(t), "/static")

	tests := []struct {
		name     string
		target   string
		status   int
		location string
	}{
		{"trailing slash added", "/static", http.StatusMovedPermanently, "/static/"},
		{"consecutive slashes merged", "/static//sub/page.html", http.StatusMovedPermanently, "/static/sub/page.html"},
		{"dot segment collapsed", "/static/./sub/page.html", http.StatusMovedPermanently, "/static/sub/page.html"},
		{"parent segment collapsed", "/static/sub/../sub/page.html", http.StatusMovedPermanently, "/static/sub/page.html"},
		{"escape above root rejected", "/static/../../etc/passwd", http.StatusBadRequest, ""},
		{"encoded escape above root not served", "/static/%2e%2e%2f%2e%2e/etc/passwd", http.StatusNotFound, ""},
		{"unmounted route", "/nope", http.StatusNotFound, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := doRequest(t, router, http.MethodGet, tc.target)
			if w.Code != tc.status {
				t.Fatalf("expected %d, got %d (location %q)", tc.status, w.Code, w.Header().Get("Location"))
			}
			if got := w.Header().Get("Location"); got != tc.location {
				t.Errorf("expected location %q, got %q", tc.location, got)
			}
		})
	}
}

// TestCacheControlAlwaysSet guards against net/http stripping Cache-Control from
// the error responses the file server generates.
func TestCacheControlAlwaysSet(t *testing.T) {
	dir := serveDir(t)
	router := mountedRouter(t, dir, "/static")

	filePath := filepath.Join(dir, "index.html")
	ctrl := newControl(router)
	if w := postMount(t, newControlRouter(ctrl), filePath, "/file"); w.Code != http.StatusCreated {
		t.Fatalf("mount file failed: %d %s", w.Code, w.Body.String())
	}

	targets := []string{
		"/static/",                 // served file
		"/static/sub/page.html",    // served nested file
		"/static",                  // trailing slash redirect
		"/static//sub/page.html",   // normalization redirect
		"/static/missing.html",     // file server 404
		"/static/../../etc/passwd", // rejected path
		"/file",                    // mounted single file
		"/nope",                    // no route
	}

	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			w := doRequest(t, router, http.MethodGet, target)
			if got := w.Header().Get("Cache-Control"); got != cacheControlValue {
				t.Errorf("expected Cache-Control %q on %d response, got %q", cacheControlValue, w.Code, got)
			}
		})
	}
}

func TestServeMountedFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(filePath, []byte(`{"key":"value"}`), 0644); err != nil {
		t.Fatal(err)
	}

	router := mountedRouter(t, filePath, "/config")

	w := doRequest(t, router, http.MethodGet, "/config")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Body.String(); got != `{"key":"value"}` {
		t.Errorf("unexpected body %q", got)
	}
}

func TestUnmountStopsServing(t *testing.T) {
	ctrl, controlRouter := setupControl(t)
	dir := serveDir(t)

	if w := postMount(t, controlRouter, dir, "/static"); w.Code != http.StatusCreated {
		t.Fatalf("mount failed: %d", w.Code)
	}
	if w := doRequest(t, ctrl.router, http.MethodGet, "/static/sub/page.html"); w.Code != http.StatusOK {
		t.Fatalf("expected 200 before unmount, got %d", w.Code)
	}

	if w := deleteMount(t, controlRouter, "/static"); w.Code != http.StatusOK {
		t.Fatalf("unmount failed: %d %s", w.Code, w.Body.String())
	}
	if w := doRequest(t, ctrl.router, http.MethodGet, "/static/sub/page.html"); w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after unmount, got %d", w.Code)
	}
}
