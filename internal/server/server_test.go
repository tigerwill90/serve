package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestPublicRouterNormalization pins the URL normalization contract of the public
// router: a request for a non canonical path is redirected to the canonical one
// rather than served in place, so each mounted file is reachable under a single URL.
func TestPublicRouterNormalization(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("hello"), 0o600); err != nil {
		t.Fatalf("cannot write fixture: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o750); err != nil {
		t.Fatalf("cannot create fixture dir: %v", err)
	}

	router := newPublicRouter(io.Discard)
	pattern, handler, _, err := buildRoute(dir, "/static", true)
	if err != nil {
		t.Fatalf("cannot build route: %v", err)
	}
	if _, err := router.Add([]string{http.MethodGet, http.MethodHead}, pattern, handler); err != nil {
		t.Fatalf("cannot add route: %v", err)
	}

	cases := []struct {
		name     string
		target   string
		location string
		want     int
	}{
		{name: "canonical directory path is served", target: "/static/", want: http.StatusOK},
		{name: "missing trailing slash redirects", target: "/static", want: http.StatusMovedPermanently, location: "/static/"},
		{name: "repeated slashes are merged", target: "/static//index.html", want: http.StatusMovedPermanently, location: "/static/index.html"},
		{name: "current directory segment is collapsed", target: "/static/./index.html", want: http.StatusMovedPermanently, location: "/static/index.html"},
		{name: "parent directory segment is collapsed", target: "/static/sub/../index.html", want: http.StatusMovedPermanently, location: "/static/index.html"},
		{name: "collapsing outside the mount does not match", target: "/static/../index.html", want: http.StatusNotFound},
		{name: "escaping above root is rejected", target: "/../etc/passwd", want: http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.target, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tc.want {
				t.Errorf("GET %s: got status %d, want %d", tc.target, w.Code, tc.want)
			}
			if got := w.Header().Get("Location"); got != tc.location {
				t.Errorf("GET %s: got Location %q, want %q", tc.target, got, tc.location)
			}
			if got := w.Header().Get("Cache-Control"); got != "no-store, max-age=0" {
				t.Errorf("GET %s: got Cache-Control %q, want %q", tc.target, got, "no-store, max-age=0")
			}
		})
	}
}

// TestPublicRouterServesMountedFile checks that the normalization options do not
// interfere with serving a file mounted at an exact route.
func TestPublicRouterServesMountedFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(file, []byte("content"), 0o600); err != nil {
		t.Fatalf("cannot write fixture: %v", err)
	}

	router := newPublicRouter(io.Discard)
	pattern, handler, mountType, err := buildRoute(file, "/notes.txt", false)
	if err != nil {
		t.Fatalf("cannot build route: %v", err)
	}
	if mountType != "file" {
		t.Fatalf("got mount type %q, want %q", mountType, "file")
	}
	if _, err := router.Add([]string{http.MethodGet, http.MethodHead}, pattern, handler); err != nil {
		t.Fatalf("cannot add route: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/notes.txt", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != "content" {
		t.Errorf("got body %q, want %q", w.Body.String(), "content")
	}
}
