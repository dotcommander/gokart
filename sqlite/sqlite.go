// Package sqlite provides SQLite database utilities wrapping modernc.org/sqlite.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/dotcommander/gokart/internal/sqltx"
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
	// Default: 4
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
		Path:        path,
		WALMode:     true,
		BusyTimeout: 5 * time.Second,
		// MaxOpenConns: 4 — SQLite is single-writer; more connections just queue.
		MaxOpenConns:    4,
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
	dsn := buildDSN(cfg)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	configurePool(db, cfg)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	return db, nil
}

func buildDSN(cfg Config) string {
	var params []string

	if cfg.Path != ":memory:" {
		params = append(params, "_txlock=immediate")
	}

	if cfg.BusyTimeout > 0 {
		params = append(params, fmt.Sprintf("_pragma=busy_timeout(%d)", cfg.BusyTimeout.Milliseconds()))
	}
	if cfg.ForeignKeys {
		params = append(params, "_pragma=foreign_keys(1)")
	}
	if cfg.WALMode {
		params = append(params, "_pragma=journal_mode(WAL)", "_pragma=synchronous(NORMAL)")
	}
	// Performance pragmas applied to every connection via DSN.
	params = append(params,
		"_pragma=cache_size(-2000)",
		"_pragma=temp_store(MEMORY)",
	)

	return fmt.Sprintf("file:%s?%s", cfg.Path, strings.Join(params, "&"))
}

func configurePool(db *sql.DB, cfg Config) {
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
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
	cfg.WALMode = false // WAL not supported for :memory:
	// CRITICAL: Must be 1 connection. Each SQLite connection to :memory:
	// gets its own separate database. Multiple connections = multiple DBs
	// that don't share data. This is a common trap.
	cfg.MaxOpenConns = 1
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
	return sqltx.Run(
		func() (*sql.Tx, error) {
			return db.BeginTx(ctx, nil)
		},
		func(tx *sql.Tx) error {
			return tx.Rollback()
		},
		func(tx *sql.Tx) error {
			return tx.Commit()
		},
		fn,
	)
}
