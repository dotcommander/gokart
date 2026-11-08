//go:build ignore

// Example: HTTP server with GoKart.
//
// This example demonstrates:
//   - Chi router with standard middleware
//   - Request timeout handling
//   - Graceful shutdown with signal handling
//   - Response helpers for common HTTP patterns
//   - Bounded JSON binding and validation (BindJSON / BindAndValidate)
//
// Run with: go run main.go
// Then test with:
//
//	curl http://localhost:8080/health
//	curl http://localhost:8080/api/users
//	curl http://localhost:8080/api/users/123
package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/dotcommander/gokart/web"
	"github.com/go-chi/chi/v5"
)

var validator = web.NewStandardValidator()

func main() {
	// Create router with standard middleware
	// StandardMiddleware includes:
	//   - RequestID: injects X-Request-ID header
	//   - RealIP: extracts real IP from X-Forwarded-For/X-Real-IP
	//   - Logger: structured request/response logging
	//   - Recoverer: panic recovery with 500 response
	router := web.NewRouter(web.RouterConfig{
		Middleware: web.StandardMiddleware,
		Timeout:    30 * time.Second,
	})

	// Health check endpoint
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		web.JSON(w, map[string]string{"status": "ok"})
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
	if err := web.ListenAndServe(":8080", router); err != nil {
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
	web.JSON(w, users)
}

// getUserHandler returns a single user by ID
func getUserHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Simulate database lookup
	if id == "404" {
		web.Error(w, http.StatusNotFound, "User not found")
		return
	}

	user := User{ID: id, Name: "Alice", Email: "alice@example.com"}
	web.JSON(w, user)
}

// createUserHandler creates a new user
func createUserHandler(w http.ResponseWriter, r *http.Request) {
	var user User
	// Bounded binding (10 MiB default) plus validation in one call.
	fields, err := web.BindAndValidate(r, validator, &user)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			web.Error(w, http.StatusRequestEntityTooLarge, "request too large")
		} else {
			web.Error(w, http.StatusBadRequest, "Invalid request body")
		}
		return
	}
	if len(fields) > 0 {
		web.JSONStatus(w, http.StatusUnprocessableEntity, map[string]any{"errors": fields})
		return
	}

	// Simulate database insert
	user.ID = fmt.Sprintf("%d", time.Now().Unix())

	// Return 201 Created with location header
	web.JSONStatus(w, http.StatusCreated, user)
}

// updateUserHandler updates an existing user
func updateUserHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var user User
	if err := web.BindJSON(r, &user); err != nil {
		web.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	user.ID = id
	web.JSON(w, user)
}

// deleteUserHandler deletes a user
func deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Simulate database delete
	log.Printf("Deleting user: %s", id)

	// Return 204 No Content
	web.NoContent(w)
}

// User represents a user in the system
type User struct {
	ID    string `json:"id"`
	Name  string `json:"name" validate:"notblank"`
	Email string `json:"email" validate:"required,email"`
}

// Example: Custom shutdown timeout
//
// The default ListenAndServe uses a 30-second shutdown timeout.
// For custom timeout, use:
//
//	err := web.ListenAndServeWithTimeout(":8080", router, 60*time.Second)
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
