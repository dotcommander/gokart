// Example: HTTP server with GoKart.
//
// This example demonstrates:
//   - Chi router with standard middleware
//   - Request timeout handling
//   - Graceful shutdown with signal handling
//   - Response helpers for common HTTP patterns
//
// Run with: go run main.go
// Then test with:
//   curl http://localhost:8080/health
//   curl http://localhost:8080/api/users
//   curl http://localhost:8080/api/users/123
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/dotcommander/gokart"
	"github.com/go-chi/chi/v5"
)

func main() {
	// Create router with standard middleware
	// StandardMiddleware includes:
	//   - RequestID: injects X-Request-ID header
	//   - RealIP: extracts real IP from X-Forwarded-For/X-Real-IP
	//   - Logger: structured request/response logging
	//   - Recoverer: panic recovery with 500 response
	router := gokart.NewRouter(gokart.RouterConfig{
		Middleware: gokart.StandardMiddleware,
		Timeout:    30 * time.Second,
	})

	// Health check endpoint
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		gokart.JSON(w, map[string]string{"status": "ok"})
	})

	// API routes
	router.Route("/api", func(r chi.Router) {
		// List users
		r.Get("/users", listUsersHandler)

		// Get user by ID
		r.Get("/users/{id}", getUserHandler)

		// Create user
		r.Post("/users", createUserHandler)

		// Update user
		r.Put("/users/{id}", updateUserHandler)

		// Delete user
		r.Delete("/users/{id}", deleteUserHandler)
	})

	// Start server with graceful shutdown
	// This blocks until SIGINT (Ctrl+C) or SIGTERM
	log.Println("Server starting on :8080")
	if err := gokart.ListenAndServe(":8080", router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// listUsersHandler returns a list of users
func listUsersHandler(w http.ResponseWriter, r *http.Request) {
	users := []User{
		{ID: "1", Name: "Alice", Email: "alice@example.com"},
		{ID: "2", Name: "Bob", Email: "bob@example.com"},
	}

	// Use response helper for 200 OK with JSON
	gokart.JSON(w, users)
}

// getUserHandler returns a single user by ID
func getUserHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Simulate database lookup
	if id == "404" {
		gokart.Error(w, http.StatusNotFound, "User not found")
		return
	}

	user := User{ID: id, Name: "Alice", Email: "alice@example.com"}
	gokart.JSON(w, user)
}

// createUserHandler creates a new user
func createUserHandler(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		gokart.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate
	if user.Name == "" || user.Email == "" {
		gokart.Error(w, http.StatusBadRequest, "Name and email are required")
		return
	}

	// Simulate database insert
	user.ID = fmt.Sprintf("%d", time.Now().Unix())

	// Return 201 Created with location header
	gokart.JSONStatus(w, http.StatusCreated, user)
}

// updateUserHandler updates an existing user
func updateUserHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		gokart.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	user.ID = id
	gokart.JSON(w, user)
}

// deleteUserHandler deletes a user
func deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Simulate database delete
	log.Printf("Deleting user: %s", id)

	// Return 204 No Content
	gokart.NoContent(w)
}

// User represents a user in the system
type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Example: Custom shutdown timeout
//
// The default ListenAndServe uses a 30-second shutdown timeout.
// For custom timeout, use:
//
//	err := gokart.ListenAndServeWithTimeout(":8080", router, 60*time.Second)
//
// Example: Manual graceful shutdown
//
// If you need to run cleanup code before shutdown:
//
//	server := &http.Server{
//	    Addr:    ":8080",
//	    Handler: router,
//	}
//
//	go func() {
//	    log.Println("Server starting on :8080")
//	    if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
//	        log.Fatalf("Server failed: %v", err)
//	    }
//	}()
//
//	// Wait for interrupt signal
//	sig := make(chan os.Signal, 1)
//	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
//	<-sig
//
//	// Run cleanup
//	log.Println("Shutting down...")
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	if err := server.Shutdown(ctx); err != nil {
//	    log.Printf("Shutdown error: %v", err)
//	}
