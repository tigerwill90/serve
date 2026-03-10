# CLAUDE.md

## Project Overview

**Serve** is a lightweight HTTP file server CLI tool written in Go. It allows users to serve files over HTTP and dynamically mount/unmount directories and files at runtime without restarting the server.

- **Module**: `github.com/tigerwill90/serve`
- **Go version**: 1.26.0
- **License**: MIT

Serve is a production-grade system. Code must be safety-first: assert preconditions and postconditions,
bound all resource allocation, handle all error paths explicitly, and never trust input. Performance matters, but correctness comes first.

We have a zero technical debt policy. If you touch code and notice something that could be better, whether in performance, architecture,
readability, safety, or error handling, fix it now. Deferring improvements almost always means they never happen. This applies to new features,
refactors, bug fixes, and code review feedback equally. The only acceptable reason to defer an improvement is if it requires a scope that would
block the current change from shipping safely.

## Writing Style

When writing prose (PR descriptions, README sections, comments), avoid AI-typical formatting: no em dashes (—), no excessive bullet points, no superlatives. Write in a natural, concise, honest tone. Never fabricate performance claims or technical justifications.

## Architecture

The project follows a clean separation between CLI, server, and client layers:

```
main.go                          # Entry point
cmd/                             # CLI commands (urfave/cli/v3)
  root.go                        # Root command and flag setup
  start.go                       # `serve start` - starts the file server
  mount.go                       # `serve mount <path> <route>` - mount a directory/file
  unmount.go                     # `serve unmount <route>` - unmount a route
  list.go                        # `serve list` - list active mounts
internal/
  server/                        # HTTP server implementation
    server.go                    # Public file server setup and lifecycle
    control.go                   # Control API (mount/unmount/list endpoints)
    control_test.go              # Control API tests
    helpers_test.go              # Shared test utilities
  client/                        # HTTP client for the control API
    client.go                    # Client implementation
    client_test.go               # Client tests
```

### Two-Server Design

The public server (default 127.0.0.1:8080) serves files and directory listings. The control server (default 127.0.0.1:8081) exposes a JSON API for managing mounts at runtime. Both servers start together and shut down gracefully on SIGINT/SIGTERM with a 5-second timeout.

### Control API

All endpoints live under `/v1/mounts` and return JSON with the `apiResponse` envelope:

```json
{"ok": true, "data": ..., "error": ""}
```

| Method | Endpoint | Purpose | Success Status |
|--------|----------|---------|----------------|
| POST | `/v1/mounts` | Mount a directory or file | 201 Created |
| DELETE | `/v1/mounts` | Unmount a route | 200 OK |
| GET | `/v1/mounts` | List active mounts | 200 OK |

POST and DELETE accept JSON bodies. POST expects `{"path": "...", "route": "..."}`. DELETE expects `{"route": "..."}`. Error responses use 400 (bad input), 404 (not found), 409 (conflict), or 500 (internal).

### Route Pattern Rules

Directories mount with a wildcard suffix for subpath matching: route `/static` becomes pattern `/static/*{filepath}`. Files mount as exact patterns without wildcards. Hostname-scoped routes (e.g. `example.com/assets`) are supported.

### Middleware

The public server applies two middleware layers via the Fox router: `fox.Logger()` for structured request logging and a custom `cacheControlMiddleware()` that sets `Cache-Control: no-store, max-age=0` on every response.

### Key Dependencies

- `github.com/fox-toolkit/fox` v0.27.1 - HTTP router with annotation support (used to store mount metadata on routes)
- `github.com/urfave/cli/v3` v3.7.0 - CLI framework

Fox annotations attach `mountInfo` metadata (route, local path, type, pattern) directly to routes, which the list endpoint reads back when enumerating mounts.

## Common Commands

```bash
# Run all tests
go test ./...

# Run tests with race detector (matches CI)
go test -race -count=1 ./...

# Build the binary
go build -o serve .

# Format code
go fmt ./...

# Vet code
go vet ./...

# Lint (requires golangci-lint v2)
golangci-lint run

# Lint and auto-fix (useful for struct field alignment and formatting)
golangci-lint run --fix
```

## CI

GitHub Actions runs on pull requests to master/main. Two jobs:

1. **Lint**: golangci-lint v2.11 with the config in `.golangci.yml`
2. **Test**: `go test -race -count=1 ./...` across Go 1.26 and stable

### Linter Configuration

Defined in `.golangci.yml` (golangci-lint v2 format). Enabled linters: errcheck, gosec, govet (all checks), ineffassign, staticcheck, unused. Formatter: gofmt. Linter rules are relaxed for test files.

## Testing

- Standard Go `testing` package with `net/http/httptest` for HTTP tests
- Test files are co-located with source files (`*_test.go`)
- Shared test helpers live in `internal/server/helpers_test.go`
- All tests are fast and self-contained (no external dependencies)
- Tests use `t.TempDir()` for filesystem isolation and `t.Helper()` on helper functions
- Client tests spin up mock HTTP servers with httptest to validate request/response handling

## Code Conventions

- **Error handling**: Always wrap errors with context using `fmt.Errorf("description: %w", err)`
- **Context**: Pass `context.Context` through handlers and server operations
- **JSON API format**: Use the standard `apiResponse` struct with `ok`, `data`, and `error` fields
- **HTTP status codes**: Use appropriate codes (201 Created, 400 Bad Request, 404 Not Found, 409 Conflict)
- **Naming**: Standard Go conventions (CamelCase exports, camelCase unexported)
- **Package visibility**: Implementation details go in `internal/`; CLI commands in `cmd/`
- **Resource cleanup**: Always defer `Body.Close()` after HTTP calls; use deferred cleanup for listeners and servers
- **Server timeouts**: Public server uses 3s read / unlimited write / 60s idle. Control server uses 5s read / 5s write / 60s idle.
