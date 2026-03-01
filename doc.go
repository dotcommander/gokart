// Package gokart provides thin wrappers around best-in-class Go packages with sensible defaults.
//
// GoKart is an opinionated service toolkit. Every component must justify its existence -
// we wrap battle-tested packages, not reinvent them.
//
// # Components
//
// Root module (github.com/dotcommander/gokart):
//   - Config: viper wrapper for config files + env vars
//   - State: JSON state persistence for CLI tools
//   - Server: HTTP server with graceful shutdown
//   - Logger: slog aliases (use gokart/logger directly)
//
// Submodules:
//   - gokart/logger: slog wrapper with JSON/text formatting
//   - gokart/cli: CLI framework wrapping cobra + lipgloss
//   - gokart/web: HTTP router, client, templ helpers, validator
//   - gokart/postgres: pgx/v5 connection pool
//   - gokart/sqlite: modernc.org/sqlite (zero CGO) with WAL mode
//   - gokart/cache: go-redis/v9 with convenience methods
//   - gokart/migrate: goose/v3 schema migrations
//   - gokart/ai: OpenAI client factory functions
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
//	log := logger.New(logger.Config{Level: "info", Format: "json"})
//
//	// Config
//	cfg, err := gokart.LoadConfig[AppConfig]("config.yaml")
//
//	// Router (gokart/web)
//	router := web.NewRouter(web.RouterConfig{Middleware: web.StandardMiddleware})
//
//	// PostgreSQL (gokart/postgres)
//	pool, err := postgres.Open(ctx, "postgres://localhost/mydb")
//
//	// Cache (gokart/cache)
//	c, err := cache.Open(ctx, "localhost:6379")
//
//	// Migrations (gokart/migrate)
//	migrate.Postgres(ctx, db, "migrations")
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
