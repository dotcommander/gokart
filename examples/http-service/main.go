// Example: Minimal HTTP service using gokart.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/dotcommander/gokart"
)

func main() {
	// Create router with standard middleware
	router := gokart.NewRouter(gokart.RouterConfig{
		Middleware: gokart.StandardMiddleware,
		Timeout:    30 * time.Second,
	})

	// Health check
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// API routes
	router.Get("/api/hello", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			name = "World"
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "Hello, " + name + "!"})
	})

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
