package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"
)

func main() {
	var port uint
	flag.UintVar(&port, "port", 80, "port to serve content")
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Fprintln(os.Stderr, "A path to a directory or a file is required!")
		os.Exit(1)
	}

	mux := http.NewServeMux()

	path := filepath.Clean(flag.Args()[0])
	fi, err := os.Stat(path)
	if err == nil {
		if fi.IsDir() {
			mux.Handle("/", http.StripPrefix("/", cacheControlMiddleware(http.FileServer(http.Dir(path)))))
		} else {
			mux.HandleFunc(fmt.Sprintf("/%s", filepath.Base(path)), cacheControlMiddlewareFunc(func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, path)
			}))
		}
	} else {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	srv := http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  0,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  5 * time.Second,
	}

	sig := make(chan os.Signal, 2)
	signal.Notify(sig, os.Interrupt, os.Kill)

	srvErr := make(chan error)

	go func() {
		srvErr <- srv.ListenAndServe()
	}()

	fmt.Printf("File server accept now connection on port %d\n", port)

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

func cacheControlMiddleware(next http.Handler) http.Handler {
	return cacheControlMiddlewareFunc(next.ServeHTTP)
}

func cacheControlMiddlewareFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store, max-age=0")
		next.ServeHTTP(w, r)
	}
}
