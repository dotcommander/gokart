package gokart

import (
	"context"
	"database/sql"

	"github.com/dotcommander/gokart/sqlite"
)

// SQLiteConfig is an alias for sqlite.Config.
// Deprecated: Use sqlite.Config directly.
type SQLiteConfig = sqlite.Config

// DefaultSQLiteConfig returns production-ready defaults.
// Deprecated: Use sqlite.DefaultConfig directly.
func DefaultSQLiteConfig(path string) SQLiteConfig {
	return sqlite.DefaultConfig(path)
}

// OpenSQLite opens a SQLite database with default settings.
// Deprecated: Use sqlite.Open directly.
func OpenSQLite(path string) (*sql.DB, error) {
	return sqlite.Open(path)
}

// OpenSQLiteContext opens a SQLite database with context.
// Deprecated: Use sqlite.OpenContext directly.
func OpenSQLiteContext(ctx context.Context, path string) (*sql.DB, error) {
	return sqlite.OpenContext(ctx, path)
}

// OpenSQLiteWithConfig opens a SQLite database with custom settings.
// Deprecated: Use sqlite.OpenWithConfig directly.
func OpenSQLiteWithConfig(ctx context.Context, cfg SQLiteConfig) (*sql.DB, error) {
	return sqlite.OpenWithConfig(ctx, cfg)
}

// SQLiteInMemory creates an in-memory SQLite database for testing.
// Deprecated: Use sqlite.InMemory directly.
func SQLiteInMemory() (*sql.DB, error) {
	return sqlite.InMemory()
}

// SQLiteTransaction executes a function within a SQLite transaction.
// Deprecated: Use sqlite.Transaction directly.
func SQLiteTransaction(ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) error) error {
	return sqlite.Transaction(ctx, db, fn)
}
