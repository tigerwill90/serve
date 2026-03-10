# CLAUDE.md

## Project Overview

**Serve** is a lightweight HTTP file server CLI tool written in Go. It allows users to serve files over HTTP and dynamically mount/unmount directories and files at runtime without restarting the server.

- **Module**: `github.com/tigerwill90/serve`
- **Go version**: 1.24.0 (toolchain 1.24.7)
- **License**: MIT

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

- **Public server** (default port 8080): Serves files and directory listings
- **Control server** (default port 8081): JSON API for managing mounts at runtime

### Key Dependencies

- `github.com/fox-toolkit/fox` - HTTP router with annotation support (used to track mount metadata)
- `github.com/urfave/cli/v3` - CLI framework

## Common Commands

```bash
# Run all tests
go test ./...

# Build the binary
go build -o serve .

# Format code
go fmt ./...

# Vet code
go vet ./...
```

## Testing

- Standard Go `testing` package with `net/http/httptest` for HTTP tests
- Test files are co-located with source files (`*_test.go`)
- Shared test helpers live in `internal/server/helpers_test.go`
- All tests are fast and self-contained (no external dependencies)

## Code Conventions

- **Error handling**: Always wrap errors with context using `fmt.Errorf("description: %w", err)`
- **Context**: Pass `context.Context` through handlers and server operations
- **JSON API format**: Use the standard `apiResponse` struct with `ok`, `data`, and `error` fields
- **HTTP status codes**: Use appropriate codes (201 Created, 400 Bad Request, 404 Not Found, 409 Conflict)
- **Naming**: Standard Go conventions (CamelCase exports, camelCase unexported)
- **Package visibility**: Implementation details go in `internal/`; CLI commands in `cmd/`
- **No linter config**: Code should pass `go fmt` and `go vet`
