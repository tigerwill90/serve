package client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientMount(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/mounts" {
			t.Errorf("expected /v1/mounts, got %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var req MountRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatal(err)
		}
		if req.Path != "/tmp/test" || req.Route != "/foo" {
			t.Errorf("unexpected body: %s", body)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"data": map[string]string{
				"route":      "/foo",
				"local_path": "/tmp/test",
				"type":       "directory",
				"pattern":    "/foo/*{filepath}",
			},
		})
	}))
	defer srv.Close()

	c := &Client{
		baseURL:    srv.URL,
		httpClient: srv.Client(),
	}

	info, err := c.Mount("/tmp/test", "/foo")
	if err != nil {
		t.Fatal(err)
	}
	if info.Type != "directory" {
		t.Errorf("expected directory, got %s", info.Type)
	}
	if info.Route != "/foo" {
		t.Errorf("expected /foo, got %s", info.Route)
	}
}

func TestClientUnmount(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, httpClient: srv.Client()}

	if err := c.Unmount("/foo"); err != nil {
		t.Fatal(err)
	}
}

func TestClientList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"data": []map[string]string{
				{"route": "/foo", "local_path": "/tmp/test", "type": "directory", "pattern": "/foo/*{filepath}"},
			},
		})
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, httpClient: srv.Client()}

	mounts, err := c.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(mounts) != 1 {
		t.Fatalf("expected 1 mount, got %d", len(mounts))
	}
	if mounts[0].Route != "/foo" {
		t.Errorf("expected /foo, got %s", mounts[0].Route)
	}
}

func TestClientServerDown(t *testing.T) {
	c := New("127.0.0.1", "19999") // unlikely port

	_, err := c.Mount("/tmp", "/foo")
	if err == nil {
		t.Fatal("expected error when server is down")
	}
	if !strings.Contains(err.Error(), "server is not running") {
		t.Errorf("expected 'server is not running' in error, got: %s", err)
	}
}
