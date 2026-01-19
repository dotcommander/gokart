// Example: Complete HTTP service combining multiple GoKart components.
//
// This example demonstrates a production-ready service pattern with:
//   - Structured logging
//   - Configuration from file + environment
//   - PostgreSQL database with connection pooling
//   - Redis caching with Remember pattern
//   - HTTP server with middleware and graceful shutdown
//   - Health check and API endpoints
//   - Transaction support
//
// Prerequisites:
//   - PostgreSQL running
//   - Redis running on localhost:6379
//   - config.yaml file (or environment variables)
//
// Run with: go run main.go
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/dotcommander/gokart"
	"github.com/dotcommander/gokart/logger"
	"github.com/dotcommander/gokart/postgres"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds application configuration
type Config struct {
	Port     int    `mapstructure:"port"`
	LogLevel string `mapstructure:"log_level"`

	Database struct {
		URL      string `mapstructure:"url"`
		MaxConns int32  `mapstructure:"max_conns"`
	} `mapstructure:"database"`

	Redis struct {
		Addr      string `mapstructure:"addr"`
		KeyPrefix string `mapstructure:"key_prefix"`
	} `mapstructure:"redis"`
}

// User represents a user in the system
type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// App holds application dependencies
type App struct {
	config *Config
	log    *slog.Logger
	db     *pgxpool.Pool
	cache  *gokart.Cache
}

func main() {
	ctx := context.Background()

	// Load configuration with defaults
	defaults := Config{
		Port:     8080,
		LogLevel: "info",
		Database: struct {
			URL      string `mapstructure:"url"`
			MaxConns int32  `mapstructure:"max_conns"`
		}{
			URL:      os.Getenv("DATABASE_URL"),
			MaxConns: 25,
		},
		Redis: struct {
			Addr      string `mapstructure:"addr"`
			KeyPrefix string `mapstructure:"key_prefix"`
		}{
			Addr:      "localhost:6379",
			KeyPrefix: "fullservice:",
		},
	}

	cfg, err := gokart.LoadConfigWithDefaults(defaults, "config.yaml")
	if err != nil {
		log.Printf("Config warning: %v (using defaults)", err)
		cfg = defaults
	}

	// Initialize logger
	appLog := logger.New(logger.Config{
		Level:  cfg.LogLevel,
		Format: "json",
	})

	appLog.Info("Starting application", "port", cfg.Port)

	// Initialize database (skip if no URL configured)
	var db *pgxpool.Pool
	if cfg.Database.URL != "" {
		db, err = postgres.OpenWithConfig(ctx, postgres.Config{
			URL:      cfg.Database.URL,
			MaxConns: cfg.Database.MaxConns,
		})
		if err != nil {
			appLog.Error("Database connection failed", "err", err)
			os.Exit(1)
		}
		defer db.Close()
		appLog.Info("Connected to PostgreSQL")
	} else {
		appLog.Warn("No DATABASE_URL configured, running without database")
	}

	// Initialize cache
	cache, err := gokart.OpenCacheWithConfig(ctx, gokart.CacheConfig{
		Addr:      cfg.Redis.Addr,
		KeyPrefix: cfg.Redis.KeyPrefix,
	})
	if err != nil {
		appLog.Warn("Cache connection failed, running without cache", "err", err)
		cache = nil
	} else {
		defer cache.Close()
		appLog.Info("Connected to Redis")
	}

	// Create application instance
	app := &App{
		config: &cfg,
		log:    appLog,
		db:     db,
		cache:  cache,
	}

	// Setup router
	router := gokart.NewRouter(gokart.RouterConfig{
		Middleware: gokart.StandardMiddleware,
		Timeout:    30 * time.Second,
	})

	// Health check
	router.Get("/health", app.healthHandler)

	// API routes
	router.Route("/api", func(r chi.Router) {
		r.Get("/users", app.listUsersHandler)
		r.Get("/users/{id}", app.getUserHandler)
		r.Post("/users", app.createUserHandler)
	})

	// Start server with graceful shutdown
	addr := fmt.Sprintf(":%d", cfg.Port)
	appLog.Info("Server starting", "addr", addr)

	if err := gokart.ListenAndServe(addr, router); err != nil {
		appLog.Error("Server error", "err", err)
		os.Exit(1)
	}
}

// healthHandler returns service health status
func (app *App) healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	health := map[string]string{
		"status": "ok",
	}

	// Check database
	if app.db != nil {
		if err := app.db.Ping(ctx); err != nil {
			health["database"] = "unhealthy"
			health["status"] = "degraded"
		} else {
			health["database"] = "healthy"
		}
	} else {
		health["database"] = "not configured"
	}

	// Check cache
	if app.cache != nil {
		if _, err := app.cache.Get(ctx, "_health_check"); err != nil && !gokart.IsNil(err) {
			health["cache"] = "unhealthy"
			health["status"] = "degraded"
		} else {
			health["cache"] = "healthy"
		}
	} else {
		health["cache"] = "not configured"
	}

	status := http.StatusOK
	if health["status"] == "degraded" {
		status = http.StatusServiceUnavailable
	}

	gokart.JSONStatus(w, status, health)
}

// listUsersHandler returns all users (with caching)
func (app *App) listUsersHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Try cache first
	if app.cache != nil {
		var users []User
		err := app.cache.RememberJSON(ctx, "users:all", 5*time.Minute, &users, func() (interface{}, error) {
			return app.fetchAllUsers(ctx)
		})
		if err == nil {
			gokart.JSON(w, users)
			return
		}
		app.log.Warn("Cache miss or error", "err", err)
	}

	// Fallback to database
	if app.db == nil {
		gokart.Error(w, http.StatusServiceUnavailable, "Database not available")
		return
	}

	users, err := app.fetchAllUsers(ctx)
	if err != nil {
		app.log.Error("Failed to fetch users", "err", err)
		gokart.Error(w, http.StatusInternalServerError, "Failed to fetch users")
		return
	}

	gokart.JSON(w, users)
}

// getUserHandler returns a single user by ID
func (app *App) getUserHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	// Try cache first
	cacheKey := fmt.Sprintf("user:%s", id)
	if app.cache != nil {
		var user User
		err := app.cache.GetJSON(ctx, cacheKey, &user)
		if err == nil {
			gokart.JSON(w, user)
			return
		}
	}

	if app.db == nil {
		gokart.Error(w, http.StatusServiceUnavailable, "Database not available")
		return
	}

	var user User
	err := app.db.QueryRow(ctx,
		"SELECT id, name, email, created_at FROM users WHERE id = $1",
		id,
	).Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			gokart.Error(w, http.StatusNotFound, "User not found")
			return
		}
		app.log.Error("Failed to fetch user", "err", err, "id", id)
		gokart.Error(w, http.StatusInternalServerError, "Failed to fetch user")
		return
	}

	// Cache the result
	if app.cache != nil {
		app.cache.SetJSON(ctx, cacheKey, user, 10*time.Minute)
	}

	gokart.JSON(w, user)
}

// createUserHandler creates a new user
func (app *App) createUserHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if app.db == nil {
		gokart.Error(w, http.StatusServiceUnavailable, "Database not available")
		return
	}

	var input struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		gokart.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if input.Name == "" || input.Email == "" {
		gokart.Error(w, http.StatusBadRequest, "Name and email are required")
		return
	}

	// Use transaction for the insert
	var user User
	err := postgres.Transaction(ctx, app.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			"INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, name, email, created_at",
			input.Name, input.Email,
		).Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
	})

	if err != nil {
		app.log.Error("Failed to create user", "err", err)
		gokart.Error(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	// Invalidate cache
	if app.cache != nil {
		app.cache.Delete(ctx, "users:all")
	}

	app.log.Info("User created", "id", user.ID, "email", user.Email)
	gokart.JSONStatus(w, http.StatusCreated, user)
}

// fetchAllUsers retrieves all users from the database
func (app *App) fetchAllUsers(ctx context.Context) ([]User, error) {
	if app.db == nil {
		return nil, sql.ErrNoRows
	}

	rows, err := app.db.Query(ctx, "SELECT id, name, email, created_at FROM users ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	return users, rows.Err()
}

// Example: Database schema for this service
//
// Run this SQL to create the required table:
//
//	CREATE TABLE users (
//	    id SERIAL PRIMARY KEY,
//	    name TEXT NOT NULL,
//	    email TEXT NOT NULL UNIQUE,
//	    created_at TIMESTAMP DEFAULT NOW()
//	);
//
// Example: config.yaml
//
//	port: 8080
//	log_level: debug
//
//	database:
//	  url: postgres://user:password@localhost:5432/mydb
//	  max_conns: 25
//
//	redis:
//	  addr: localhost:6379
//	  key_prefix: "myapp:"
//
// Example: Environment variables
//
// All config values can be overridden with env vars:
//   export PORT=9000
//   export LOG_LEVEL=debug
//   export DATABASE_URL=postgres://user:pass@localhost:5432/db
//   export DATABASE_MAX_CONNS=50
//   export REDIS_ADDR=redis.example.com:6379
