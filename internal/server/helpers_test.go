package server

import (
	"io"
	"log/slog"

	"github.com/fox-toolkit/fox"
)

// newTestRouter mirrors the production public router so tests exercise the real
// normalization and middleware configuration.
func newTestRouter() *fox.Router {
	return newPublicRouter(slog.NewTextHandler(io.Discard, nil))
}
