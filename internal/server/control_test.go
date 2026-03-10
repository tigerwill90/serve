package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/fox-toolkit/fox"
)

func setupControl(t *testing.T) (*control, *fox.Router) {
	t.Helper()

	router := newTestRouter()
	ctrl := newControl(router)
	controlRouter := newControlRouter(ctrl)

	return ctrl, controlRouter
}

func postMount(t *testing.T, controlRouter *fox.Router, path, route string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(mountRequest{Path: path, Route: route})
	req := httptest.NewRequest(http.MethodPost, "/v1/mounts", bytes.NewReader(body))
	w := httptest.NewRecorder()
	controlRouter.ServeHTTP(w, req)
	return w
}

func deleteMount(t *testing.T, controlRouter *fox.Router, route string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(unmountRequest{Route: route})
	req := httptest.NewRequest(http.MethodDelete, "/v1/mounts", bytes.NewReader(body))
	w := httptest.NewRecorder()
	controlRouter.ServeHTTP(w, req)
	return w
}

func getList(t *testing.T, controlRouter *fox.Router) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/v1/mounts", nil)
	w := httptest.NewRecorder()
	controlRouter.ServeHTTP(w, req)
	return w
}

type testResponse struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

func decodeResponse(t *testing.T, w *httptest.ResponseRecorder) testResponse {
	t.Helper()
	var resp testResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return resp
}

func TestMountDirectory(t *testing.T) {
	_, controlRouter := setupControl(t)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<h1>hello</h1>"), 0644); err != nil {
		t.Fatal(err)
	}

	w := postMount(t, controlRouter, dir, "/static")
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeResponse(t, w)
	if !resp.OK {
		t.Fatalf("expected ok=true, got error: %s", resp.Error)
	}

	var info mountInfo
	if err := json.Unmarshal(resp.Data, &info); err != nil {
		t.Fatal(err)
	}
	if info.Type != "directory" {
		t.Errorf("expected type=directory, got %s", info.Type)
	}
	if info.Pattern != "/static/*{filepath}" {
		t.Errorf("expected pattern=/static/*{filepath}, got %s", info.Pattern)
	}
}

func TestMountFile(t *testing.T) {
	_, controlRouter := setupControl(t)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(filePath, []byte(`{"key":"value"}`), 0644); err != nil {
		t.Fatal(err)
	}

	w := postMount(t, controlRouter, filePath, "/config")
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeResponse(t, w)
	if !resp.OK {
		t.Fatalf("expected ok=true, got error: %s", resp.Error)
	}

	var info mountInfo
	if err := json.Unmarshal(resp.Data, &info); err != nil {
		t.Fatal(err)
	}
	if info.Type != "file" {
		t.Errorf("expected type=file, got %s", info.Type)
	}
	if info.Pattern != "/config" {
		t.Errorf("expected pattern=/config, got %s", info.Pattern)
	}
}

func TestMountWithHostname(t *testing.T) {
	_, controlRouter := setupControl(t)

	dir := t.TempDir()

	w := postMount(t, controlRouter, dir, "example.com/assets")
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeResponse(t, w)
	var info mountInfo
	if err := json.Unmarshal(resp.Data, &info); err != nil {
		t.Fatal(err)
	}
	if info.Pattern != "example.com/assets/*{filepath}" {
		t.Errorf("expected pattern=example.com/assets/*{filepath}, got %s", info.Pattern)
	}
}

func TestUnmount(t *testing.T) {
	_, controlRouter := setupControl(t)

	dir := t.TempDir()
	w := postMount(t, controlRouter, dir, "/files")
	if w.Code != http.StatusCreated {
		t.Fatalf("mount failed: %d %s", w.Code, w.Body.String())
	}

	w = deleteMount(t, controlRouter, "/files")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeResponse(t, w)
	if !resp.OK {
		t.Fatalf("expected ok=true, got error: %s", resp.Error)
	}
}

func TestUnmountNotFound(t *testing.T) {
	_, controlRouter := setupControl(t)

	w := deleteMount(t, controlRouter, "/nonexistent")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListMounts(t *testing.T) {
	_, controlRouter := setupControl(t)

	// Empty list initially
	w := getList(t, controlRouter)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	resp := decodeResponse(t, w)
	var mounts []mountInfo
	if err := json.Unmarshal(resp.Data, &mounts); err != nil {
		t.Fatal(err)
	}
	if len(mounts) != 0 {
		t.Fatalf("expected 0 mounts, got %d", len(mounts))
	}

	// Add two mounts
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	postMount(t, controlRouter, dir1, "/alpha")
	postMount(t, controlRouter, dir2, "/beta")

	w = getList(t, controlRouter)
	resp = decodeResponse(t, w)
	if err := json.Unmarshal(resp.Data, &mounts); err != nil {
		t.Fatal(err)
	}
	if len(mounts) != 2 {
		t.Fatalf("expected 2 mounts, got %d", len(mounts))
	}
}

func TestMountConflict(t *testing.T) {
	_, controlRouter := setupControl(t)

	dir := t.TempDir()
	w := postMount(t, controlRouter, dir, "/dup")
	if w.Code != http.StatusCreated {
		t.Fatalf("first mount failed: %d", w.Code)
	}

	w = postMount(t, controlRouter, dir, "/dup")
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 on duplicate mount, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMountInvalidPath(t *testing.T) {
	_, controlRouter := setupControl(t)

	w := postMount(t, controlRouter, "/nonexistent/path/xyz", "/test")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMountMissingFields(t *testing.T) {
	_, controlRouter := setupControl(t)

	// Missing path
	body, _ := json.Marshal(mountRequest{Route: "/test"})
	req := httptest.NewRequest(http.MethodPost, "/v1/mounts", bytes.NewReader(body))
	w := httptest.NewRecorder()
	controlRouter.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	// Missing route
	body, _ = json.Marshal(mountRequest{Path: "/tmp"})
	req = httptest.NewRequest(http.MethodPost, "/v1/mounts", bytes.NewReader(body))
	w = httptest.NewRecorder()
	controlRouter.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
