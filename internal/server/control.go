package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/fox-toolkit/fox"
)

type annotationKey struct{ name string }

var (
	localPathKey = annotationKey{"local_path"}
	mountTypeKey = annotationKey{"type"}
	routeKey     = annotationKey{"route"}
)

type control struct {
	router *fox.Router
}

func newControl(router *fox.Router) *control {
	return &control{router: router}
}

type mountRequest struct {
	Path  string `json:"path"`
	Route string `json:"route"`
}

type unmountRequest struct {
	Route string `json:"route"`
}

type mountInfo struct {
	Route     string `json:"route"`
	LocalPath string `json:"local_path"`
	Type      string `json:"type"`
	Pattern   string `json:"pattern"`
}

type apiResponse struct {
	OK    bool   `json:"ok"`
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

func (ctrl *control) handleMount(w http.ResponseWriter, r *http.Request) {
	var req mountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse{Error: "invalid request body"})
		return
	}

	if req.Path == "" || req.Route == "" {
		writeJSON(w, http.StatusBadRequest, apiResponse{Error: "path and route are required"})
		return
	}

	absPath, err := filepath.Abs(req.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse{Error: fmt.Sprintf("invalid path: %s", err)})
		return
	}

	fi, err := os.Stat(absPath)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse{Error: fmt.Sprintf("path not found: %s", err)})
		return
	}

	pattern, handler, mountType, err := buildRoute(absPath, req.Route, fi.IsDir())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse{Error: err.Error()})
		return
	}

	methods := []string{http.MethodGet, http.MethodHead}
	_, err = ctrl.router.Add(
		methods,
		pattern,
		handler,
		fox.WithAnnotation(localPathKey, absPath),
		fox.WithAnnotation(mountTypeKey, mountType),
		fox.WithAnnotation(routeKey, req.Route),
	)
	if err != nil {
		writeJSON(w, http.StatusConflict, apiResponse{Error: fmt.Sprintf("failed to add route: %s", err)})
		return
	}

	writeJSON(w, http.StatusCreated, apiResponse{
		OK: true,
		Data: mountInfo{
			Route:     req.Route,
			LocalPath: absPath,
			Type:      mountType,
			Pattern:   pattern,
		},
	})
}

func (ctrl *control) handleUnmount(w http.ResponseWriter, r *http.Request) {
	var req unmountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse{Error: "invalid request body"})
		return
	}

	if req.Route == "" {
		writeJSON(w, http.StatusBadRequest, apiResponse{Error: "route is required"})
		return
	}

	methods := []string{http.MethodGet, http.MethodHead}

	// Find matching route by annotation
	var pattern string
	for route := range ctrl.router.Iter().All() {
		if route.Annotation(routeKey) == req.Route {
			pattern = route.Pattern()
			break
		}
	}

	if pattern == "" {
		writeJSON(w, http.StatusNotFound, apiResponse{Error: fmt.Sprintf("route %q not found", req.Route)})
		return
	}

	if _, err := ctrl.router.Delete(methods, pattern); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse{Error: fmt.Sprintf("failed to delete route: %s", err)})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{OK: true})
}

func (ctrl *control) handleList(w http.ResponseWriter, _ *http.Request) {
	var mounts []mountInfo
	seen := make(map[string]bool)

	for route := range ctrl.router.Iter().All() {
		localPath, ok := route.Annotation(localPathKey).(string)
		if !ok {
			continue
		}
		routeVal, _ := route.Annotation(routeKey).(string)
		mountType, _ := route.Annotation(mountTypeKey).(string)

		if seen[route.Pattern()] {
			continue
		}
		seen[route.Pattern()] = true

		mounts = append(mounts, mountInfo{
			Route:     routeVal,
			LocalPath: localPath,
			Type:      mountType,
			Pattern:   route.Pattern(),
		})
	}

	if mounts == nil {
		mounts = []mountInfo{}
	}

	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: mounts})
}

func buildRoute(absPath, route string, isDir bool) (pattern string, handler fox.HandlerFunc, mountType string, err error) {
	// Extract path portion (everything after hostname or the route itself if it starts with /)
	pathPortion := route
	if !strings.HasPrefix(route, "/") {
		// Has hostname: extract path after first /
		idx := strings.Index(route, "/")
		if idx == -1 {
			// Hostname only, no path
			pathPortion = "/"
		} else {
			pathPortion = route[idx:]
		}
	}

	if isDir {
		mountType = "directory"
		// Ensure path portion ends with /
		cleanPath := strings.TrimSuffix(pathPortion, "/") + "/"
		pattern = strings.TrimSuffix(route, "/") + "/*{filepath}"

		handler = fox.WrapH(http.StripPrefix(cleanPath, http.FileServer(http.Dir(absPath))))
	} else {
		mountType = "file"
		pattern = route
		filePath := absPath
		handler = fox.WrapF(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, filePath)
		})
	}

	return pattern, handler, mountType, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
