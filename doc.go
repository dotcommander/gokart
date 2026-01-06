// Package gokart provides thin wrappers around best-in-class Go packages with sensible defaults.
//
// GoKart is an opinionated service toolkit. Every component must justify its existence -
// we wrap battle-tested packages, not reinvent them.
//
// # Components
//
//   - Logger: slog wrapper with JSON/text formatting
//   - Config: viper wrapper for config files + env vars
//   - Router: chi wrapper with standard middleware
//   - HTTP Client: retryablehttp wrapper with automatic retries
//   - Validator: go-playground/validator with JSON field names
//   - PostgreSQL: pgx/v5 connection pool
//   - SQLite: modernc.org/sqlite (zero CGO) with WAL mode
//   - Templates: a-h/templ HTTP integration helpers
//   - Cache: go-redis/v9 with convenience methods
//   - Migrations: goose/v3 schema migrations
//
// # Design Principles
//
//   - Thin wrappers: No business logic, just factory functions
//   - Sensible defaults: Zero-config works for development and production
//   - Best-in-class: Wrap proven packages, don't reinvent
//   - Fight for inclusion: stdlib-sufficient things stay in stdlib
//
// # Quick Start
//
//	// Logger
//	log := gokart.NewLogger(gokart.LogConfig{Level: "info", Format: "json"})
//
//	// Config
//	cfg, err := gokart.LoadConfig[AppConfig]("config.yaml")
//
//	// Router
//	router := gokart.NewRouter(gokart.RouterConfig{Middleware: gokart.StandardMiddleware})
//
//	// PostgreSQL
//	pool, err := gokart.OpenPostgres(ctx, "postgres://localhost/mydb")
//
//	// Cache
//	cache, err := gokart.OpenCache(ctx, "localhost:6379")
//
//	// Migrations
//	gokart.PostgresMigrate(ctx, db, "migrations")
//
// # CLI Subpackage
//
// For CLI applications, use the gokart/cli subpackage which wraps cobra and lipgloss:
//
//	import "github.com/dotcommander/gokart/cli"
//
//	app := cli.NewApp("myapp", "1.0.0").
//	    WithDescription("My application").
//	    WithStandardFlags()
//	app.Run()
//
// # Not Included
//
// GoKart intentionally excludes things where stdlib is sufficient:
//
//   - Error handling: use errors.Is/As and fmt.Errorf("%w", err)
//   - File operations: use os, io, filepath
//   - String manipulation: use strings
//   - Environment variables: viper.AutomaticEnv() handles this
//
// Domain-specific packages (AI/ML, document processing) belong in separate modules.
package gokart
