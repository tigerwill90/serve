package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fox-toolkit/fox"
)

func main() {
	var port string
	flag.StringVar(&port, "port", "8080", "port to serve content")
	var host string
	flag.StringVar(&host, "host", "0.0.0.0", "host to serve content")
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Fprintln(os.Stderr, "A path to a directory or a file is required!")
		os.Exit(1)
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(host, port))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot resolve tcp address: %s\n", err)
		os.Exit(1)
	}

	path := filepath.Clean(flag.Args()[0])
	fi, err := os.Stat(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	f := fox.MustRouter(
		fox.WithMiddleware(
			fox.Logger(slog.NewTextHandler(os.Stdout, nil)),
			cacheControlMiddleware(),
		),
	)

	if fi.IsDir() {
		fileServer := http.FileServer(http.Dir(path))
		f.MustAdd([]string{http.MethodGet, http.MethodHead}, "/*{filepath}", func(c *fox.Context) {
			http.StripPrefix("/", fileServer).ServeHTTP(c.Writer(), c.Request())
		})
	} else {
		f.MustAdd([]string{http.MethodGet, http.MethodHead}, fmt.Sprintf("/%s", filepath.Base(path)), func(c *fox.Context) {
			http.ServeFile(c.Writer(), c.Request(), path)
		})
	}

	srv := http.Server{
		Handler:        f,
		ReadTimeout:    3 * time.Second,
		WriteTimeout:   0,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	lis, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot listen on %s: %s\n", tcpAddr.String(), err)
		os.Exit(1)
	}

	sig := make(chan os.Signal, 2)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	srvErr := make(chan error)

	go func() {
		srvErr <- srv.Serve(lis)
	}()

	fmt.Printf("File server accept now connection on %s %s\n\n", tcpAddr.String(), unquoteCodePoint("\\U0001f680"))

	select {
	case err := <-srvErr:
		fmt.Fprintln(os.Stderr, err)
		break
	case <-sig:
		break
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("File server stopped")
}

func cacheControlMiddleware() fox.MiddlewareFunc {
	return func(next fox.HandlerFunc) fox.HandlerFunc {
		return func(c *fox.Context) {
			c.SetHeader("Cache-Control", "no-store, max-age=0")
			next(c)
		}
	}
}

func unquoteCodePoint(s string) string {
	r, err := strconv.ParseInt(strings.TrimPrefix(s, "\\U"), 16, 32)
	if err != nil {
		panic(err)
	}
	return string(rune(r))
}
