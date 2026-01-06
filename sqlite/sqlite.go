// Package sqlite provides SQLite database utilities wrapping modernc.org/sqlite.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Config configures SQLite connection behavior.
type Config struct {
	// Path is the database file path. Use ":memory:" for in-memory database.
	Path string

	// WALMode enables Write-Ahead Logging for better concurrency.
	// Default: true
	WALMode bool

	// BusyTimeout is how long to wait for locks.
	// Default: 5 seconds
	BusyTimeout time.Duration

	// MaxOpenConns is the maximum number of open connections.
	// Default: 25
	MaxOpenConns int

	// MaxIdleConns is the maximum number of idle connections.
	// Default: 5
	MaxIdleConns int

	// ConnMaxLifetime is how long a connection can be reused.
	// Default: 1 hour
	ConnMaxLifetime time.Duration

	// ForeignKeys enables foreign key constraints.
	// Default: true
	ForeignKeys bool
}

// DefaultConfig returns production-ready defaults.
func DefaultConfig(path string) Config {
	return Config{
		Path:            path,
		WALMode:         true,
		BusyTimeout:     5 * time.Second,
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		ForeignKeys:     true,
	}
}

// Open opens a SQLite database with default settings.
//
// Uses modernc.org/sqlite (pure Go, zero CGO) with production-ready defaults:
//   - WAL mode for better concurrency
//   - Foreign keys enabled
//   - Connection pooling configured
//   - Performance pragmas applied
//
// Example:
//
//	db, err := sqlite.Open("app.db")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer db.Close()
func Open(path string) (*sql.DB, error) {
	return OpenContext(context.Background(), path)
}

// OpenContext opens a SQLite database with context.
func OpenContext(ctx context.Context, path string) (*sql.DB, error) {
	return OpenWithConfig(ctx, DefaultConfig(path))
}

// OpenWithConfig opens a SQLite database with custom settings.
//
// Example:
//
//	db, err := sqlite.OpenWithConfig(ctx, sqlite.Config{
//	    Path:         "app.db",
//	    WALMode:      true,
//	    MaxOpenConns: 50,
//	})
func OpenWithConfig(ctx context.Context, cfg Config) (*sql.DB, error) {
	// Build DSN with pragmas
	dsn := cfg.Path
	if cfg.Path != ":memory:" {
		dsn = fmt.Sprintf("file:%s?_txlock=immediate", cfg.Path)
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// Configure connection pool
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	// Apply pragmas
	pragmas := []string{}

	if cfg.WALMode {
		pragmas = append(pragmas, "PRAGMA journal_mode=WAL")
		pragmas = append(pragmas, "PRAGMA synchronous=NORMAL")
	}

	if cfg.BusyTimeout > 0 {
		pragmas = append(pragmas, fmt.Sprintf("PRAGMA busy_timeout=%d", cfg.BusyTimeout.Milliseconds()))
	}

	if cfg.ForeignKeys {
		pragmas = append(pragmas, "PRAGMA foreign_keys=ON")
	}

	// Performance pragmas
	pragmas = append(pragmas,
		"PRAGMA cache_size=-2000",  // 2MB cache
		"PRAGMA temp_store=MEMORY", // temp tables in memory
	)

	for _, pragma := range pragmas {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("set pragma %q: %w", pragma, err)
		}
	}

	// Verify connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	return db, nil
}

// InMemory creates an in-memory SQLite database for testing.
//
// Example:
//
//	db, err := sqlite.InMemory()
//	if err != nil {
//	    t.Fatal(err)
//	}
//	defer db.Close()
func InMemory() (*sql.DB, error) {
	cfg := DefaultConfig(":memory:")
	cfg.WALMode = false        // WAL not supported for :memory:
	cfg.MaxOpenConns = 1       // Single connection for in-memory to share state
	cfg.MaxIdleConns = 1
	return OpenWithConfig(context.Background(), cfg)
}

// Transaction executes a function within a SQLite transaction.
// Automatically commits on success, rolls back on error or panic.
//
// Example:
//
//	err := sqlite.Transaction(ctx, db, func(tx *sql.Tx) error {
//	    _, err := tx.ExecContext(ctx, "INSERT INTO users (name) VALUES (?)", "John")
//	    return err
//	})
func Transaction(ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
