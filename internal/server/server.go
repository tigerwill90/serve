package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fox-toolkit/fox"
)

type Config struct {
	Host        string
	Port        string
	ControlHost string
	ControlPort string
}

type Server struct {
	router     *fox.Router
	publicSrv  *http.Server
	controlSrv *http.Server
}

func New(cfg Config) (*Server, error) {
	publicAddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(cfg.Host, cfg.Port))
	if err != nil {
		return nil, fmt.Errorf("cannot resolve public address: %w", err)
	}

	controlAddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(cfg.ControlHost, cfg.ControlPort))
	if err != nil {
		return nil, fmt.Errorf("cannot resolve control address: %w", err)
	}

	router := newPublicRouter(slog.NewTextHandler(os.Stdout, nil))

	ctrl := newControl(router)
	controlRouter := newControlRouter(ctrl)

	s := &Server{
		router: router,
		publicSrv: &http.Server{
			Addr:              publicAddr.String(),
			Handler:           router,
			ReadHeaderTimeout: 3 * time.Second,
			WriteTimeout:      0,
			IdleTimeout:       75 * time.Second,
			MaxHeaderBytes:    8192,
		},
		controlSrv: &http.Server{
			Addr:           controlAddr.String(),
			Handler:        controlRouter,
			ReadTimeout:    5 * time.Second,
			WriteTimeout:   5 * time.Second,
			IdleTimeout:    75 * time.Second,
			MaxHeaderBytes: 8192,
		},
	}

	return s, nil
}

func (s *Server) Run() error {
	publicLis, err := net.Listen("tcp", s.publicSrv.Addr)
	if err != nil {
		return fmt.Errorf("cannot listen on %s: %w", s.publicSrv.Addr, err)
	}

	controlLis, err := net.Listen("tcp", s.controlSrv.Addr)
	if err != nil {
		return fmt.Errorf("cannot listen on %s: %w", s.controlSrv.Addr, err)
	}

	sig := make(chan os.Signal, 2)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	srvErr := make(chan error, 2)

	go func() {
		srvErr <- s.publicSrv.Serve(publicLis)
	}()

	go func() {
		srvErr <- s.controlSrv.Serve(controlLis)
	}()

	fmt.Printf("File server listening on %s\n", s.publicSrv.Addr)
	fmt.Printf("Control API listening on %s\n", s.controlSrv.Addr)

	select {
	case err := <-srvErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-sig:
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var shutdownErr error
	if err := s.publicSrv.Shutdown(ctx); err != nil {
		shutdownErr = errors.Join(shutdownErr, fmt.Errorf("public server shutdown: %w", err))
	}
	if err := s.controlSrv.Shutdown(ctx); err != nil {
		shutdownErr = errors.Join(shutdownErr, fmt.Errorf("control server shutdown: %w", err))
	}

	fmt.Println("File server stopped")
	return shutdownErr
}

// newPublicRouter builds the router serving mounted files. Request paths are
// normalized before lookup: a missing or extra trailing slash, consecutive
// slashes and "." / ".." segments all redirect to the canonical path when it
// matches a mounted route. A ".." escaping above the root is rejected with a
// 400 by the router before lookup.
func newPublicRouter(logHandler slog.Handler) *fox.Router {
	return fox.MustRouter(
		fox.WithTrailingSlash(fox.RedirectSlash),
		fox.WithMergeSlashes(fox.RedirectPath),
		fox.WithCollapseDotSegments(fox.RedirectPath),
		fox.WithMiddleware(
			fox.Logger(logHandler),
			cacheControlMiddleware(),
		),
	)
}

const cacheControlValue = "no-store, max-age=0"

// cacheControlWriter applies Cache-Control just before the response headers are
// flushed rather than up front. Setting the header before calling the handler is
// not enough: net/http deletes Cache-Control, Content-Encoding, Etag and
// Last-Modified when the file server turns a filesystem error into an error
// response, so every 404 served by a mount would go out without the header.
type cacheControlWriter struct {
	fox.ResponseWriter
}

// setCacheControl is a no-op once the headers are flushed, since mutating them
// at that point has no effect on the response.
func (w *cacheControlWriter) setCacheControl() {
	if !w.Written() {
		w.Header().Set("Cache-Control", cacheControlValue)
	}
}

func (w *cacheControlWriter) WriteHeader(code int) {
	w.setCacheControl()
	w.ResponseWriter.WriteHeader(code)
}

func (w *cacheControlWriter) Write(b []byte) (int, error) {
	w.setCacheControl()
	return w.ResponseWriter.Write(b)
}

func (w *cacheControlWriter) WriteString(s string) (int, error) {
	w.setCacheControl()
	return w.ResponseWriter.WriteString(s)
}

func (w *cacheControlWriter) ReadFrom(r io.Reader) (int64, error) {
	w.setCacheControl()
	return w.ResponseWriter.ReadFrom(r)
}

func (w *cacheControlWriter) FlushError() error {
	w.setCacheControl()
	return w.ResponseWriter.FlushError()
}

func cacheControlMiddleware() fox.MiddlewareFunc {
	return func(next fox.HandlerFunc) fox.HandlerFunc {
		return func(c *fox.Context) {
			c.SetWriter(&cacheControlWriter{ResponseWriter: c.Writer()})
			next(c)
		}
	}
}
