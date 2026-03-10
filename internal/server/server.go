package server

import (
	"context"
	"errors"
	"fmt"
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

	controlAddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(cfg.Host, cfg.ControlPort))
	if err != nil {
		return nil, fmt.Errorf("cannot resolve control address: %w", err)
	}

	router := fox.MustRouter(
		fox.WithMiddleware(
			fox.Logger(slog.NewTextHandler(os.Stdout, nil)),
			cacheControlMiddleware(),
		),
	)

	ctrl := newControl(router)
	controlRouter := newControlRouter(ctrl)

	s := &Server{
		router: router,
		publicSrv: &http.Server{
			Addr:           publicAddr.String(),
			Handler:        router,
			ReadTimeout:    3 * time.Second,
			WriteTimeout:   0,
			IdleTimeout:    60 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
		controlSrv: &http.Server{
			Addr:         controlAddr.String(),
			Handler:      controlRouter,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			IdleTimeout:  60 * time.Second,
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

func cacheControlMiddleware() fox.MiddlewareFunc {
	return func(next fox.HandlerFunc) fox.HandlerFunc {
		return func(c *fox.Context) {
			c.SetHeader("Cache-Control", "no-store, max-age=0")
			next(c)
		}
	}
}
