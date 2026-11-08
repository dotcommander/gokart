//go:build ignore

// Example: The same HTTP service with no GoKart imports at all.
//
// This is philosophy point 7 made concrete: an application can delete the
// wrapper without redesigning its business logic. The handlers below are
// identical to docs/examples/http-server; only the plumbing is upstream:
//
//	gokart/web router + middleware  ->  net/http ServeMux + your middleware
//	gokart/web JSON/Error helpers   ->  encoding/json
//	gokart/web ListenAndServe       ->  http.Server + signal-aware Shutdown
//
// What you lose by deleting: bounded binding defaults, the standard
// middleware bundle, and the graceful-shutdown helper — you own those
// policies again. What you gain: zero GoKart dependency.
//
// Run with: go run docs/examples/upstream-direct/main.go
// Then: curl http://localhost:8080/health
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/hello", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			name = "World"
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "Hello, " + name + "!"})
	})

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}()

	fmt.Println("Server starting on :8080 (upstream-direct, no GoKart)")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
