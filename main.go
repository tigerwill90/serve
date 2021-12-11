package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func main() {
	var port string
	flag.StringVar(&port, "port", "80", "port to serve content")
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

	mux := http.NewServeMux()

	path := filepath.Clean(flag.Args()[0])
	fi, err := os.Stat(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if fi.IsDir() {
		mux.Handle("/", http.StripPrefix("/", loggingMiddleware(path, fi.IsDir(), cacheControlMiddleware(http.FileServer(http.Dir(path))))))
	} else {
		mux.HandleFunc(fmt.Sprintf("/%s", filepath.Base(path)), loggingMiddlewareFunc(path, fi.IsDir(), cacheControlMiddlewareFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, path)
		})))
	}

	srv := http.Server{
		Handler:      mux,
		ReadTimeout:  0,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  5 * time.Second,
	}

	lis, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot listen on %s: %s\n", tcpAddr.String(), err)
		os.Exit(1)
	}

	sig := make(chan os.Signal, 2)
	signal.Notify(sig, os.Interrupt, os.Kill)

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

func loggingMiddlewareFunc(root string, isDir bool, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		path := root
		if isDir {
			path = filepath.Clean(filepath.Join(root, r.URL.Path))
		}
		fi, err := os.Stat(path)
		if err == nil && !fi.IsDir() {
			log.Printf("[%s] served %s in %s", r.Method, r.URL.Path, time.Since(start))
		}
	}
}

func loggingMiddleware(root string, isDir bool, next http.Handler) http.Handler {
	return loggingMiddlewareFunc(root, isDir, next.ServeHTTP)
}

func cacheControlMiddleware(next http.Handler) http.Handler {
	return cacheControlMiddlewareFunc(next.ServeHTTP)
}

func cacheControlMiddlewareFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store, max-age=0")
		next.ServeHTTP(w, r)
	}
}

func unquoteCodePoint(s string) string {
	r, err := strconv.ParseInt(strings.TrimPrefix(s, "\\U"), 16, 32)
	if err != nil {
		panic(err)
	}
	return string(r)
}
