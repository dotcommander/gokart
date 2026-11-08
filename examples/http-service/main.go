// Example: Minimal HTTP service using gokart.
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/dotcommander/gokart/web"
)

func main() {
	// Create router with standard middleware
	router := web.NewRouter(web.RouterConfig{
		Middleware: web.StandardMiddleware,
		Timeout:    30 * time.Second,
	})

	// Health check
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		web.JSON(w, map[string]string{"status": "ok"})
	})

	// API routes
	router.Get("/api/hello", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			name = "World"
		}
		web.JSON(w, map[string]string{"message": "Hello, " + name + "!"})
	})

	log.Println("Server starting on :8080")
	if err := web.ListenAndServeWithTimeout(":8080", router, 10*time.Second); err != nil {
		log.Fatal(err)
	}
}
