package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"sync"

	"github.com/pressly/goose/v3"
)

// migrateMu protects goose global state during concurrent migrations.
// Goose uses package-level variables for configuration (dialect, table name, base FS),
// which means concurrent calls to SetDialect, SetTableName, or SetBaseFS from
// different goroutines could race. While the database itself handles locking for
// the actual migration execution, this mutex ensures our setup phase is atomic.
// This is a defensive measure for applications that might run migrations from
// multiple goroutines (e.g., in test parallelization scenarios).
var migrateMu sync.Mutex

// Config configures database migrations.
type Config struct {
	// Dir is the directory containing migration files.
	// Default: "migrations"
	Dir string

	// Table is the name of the migrations tracking table.
	// Default: "goose_db_version"
	Table string

	// Dialect is the database dialect (postgres, sqlite3, mysql).
	// Auto-detected if not specified.
	Dialect string

	// FS is an optional filesystem for embedded migrations.
	FS fs.FS

	// AllowMissing allows applying missing (out-of-order) migrations.
	// Default: false
	AllowMissing bool

	// NoVersioning disables version tracking (for one-off scripts).
	// Default: false
	NoVersioning bool
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Dir:   "migrations",
		Table: "goose_db_version",
	}
}

// Up runs all pending migrations.
//
// Example with file-based migrations:
//
//	db, _ := gokart.OpenPostgres(ctx, url)
//	err := migrate.Up(ctx, db.Config().ConnConfig.Database, migrate.Config{
//	    Dir:     "migrations",
//	    Dialect: "postgres",
//	})
//
// Example with embedded migrations:
//
//	//go:embed migrations/*.sql
//	var migrations embed.FS
//
//	err := migrate.Up(ctx, db, migrate.Config{
//	    FS:      migrations,
//	    Dir:     "migrations",
//	    Dialect: "postgres",
//	})
func Up(ctx context.Context, db *sql.DB, cfg Config) error {
	if err := withConfiguredGoose(ctx, db, &cfg, func() error {
		return goose.UpContext(ctx, db, cfg.Dir)
	}); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}

// Down rolls back the last migration.
func Down(ctx context.Context, db *sql.DB, cfg Config) error {
	if err := withConfiguredGoose(ctx, db, &cfg, func() error {
		return goose.DownContext(ctx, db, cfg.Dir)
	}); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	return nil
}

// DownTo rolls back to a specific version.
func DownTo(ctx context.Context, db *sql.DB, cfg Config, version int64) error {
	if err := withConfiguredGoose(ctx, db, &cfg, func() error {
		return goose.DownToContext(ctx, db, cfg.Dir, version)
	}); err != nil {
		return fmt.Errorf("rollback to version %d failed: %w", version, err)
	}

	return nil
}

// Reset rolls back all migrations.
func Reset(ctx context.Context, db *sql.DB, cfg Config) error {
	return DownTo(ctx, db, cfg, 0)
}

// Status prints the status of all migrations.
func Status(ctx context.Context, db *sql.DB, cfg Config) error {
	if err := withConfiguredGoose(ctx, db, &cfg, func() error {
		return goose.StatusContext(ctx, db, cfg.Dir)
	}); err != nil {
		return fmt.Errorf("status failed: %w", err)
	}

	return nil
}

// Version returns the current migration version.
func Version(ctx context.Context, db *sql.DB, cfg Config) (int64, error) {
	var version int64
	if err := withConfiguredGoose(ctx, db, &cfg, func() error {
		var err error
		version, err = goose.GetDBVersionContext(ctx, db)
		return err
	}); err != nil {
		return 0, fmt.Errorf("failed to get version: %w", err)
	}

	return version, nil
}

// Create creates a new migration file.
//
// Example:
//
//	err := migrate.Create("migrations", "add_users_table", "sql")
func Create(dir, name, migrationType string) error {
	if dir == "" {
		dir = "migrations"
	}

	if migrationType == "" {
		migrationType = "sql"
	}

	if err := goose.Create(nil, dir, name, migrationType); err != nil {
		return fmt.Errorf("failed to create migration: %w", err)
	}

	return nil
}

// Postgres is a convenience function for PostgreSQL migrations.
//
// Example:
//
//	pool, _ := gokart.OpenPostgres(ctx, url)
//	db := stdlib.OpenDBFromPool(pool)
//	err := migrate.Postgres(ctx, db, "migrations")
func Postgres(ctx context.Context, db *sql.DB, dir string) error {
	return Up(ctx, db, Config{
		Dir:     dir,
		Dialect: "postgres",
	})
}

// SQLite is a convenience function for SQLite migrations.
//
// Example:
//
//	db, _ := gokart.OpenSQLite("app.db")
//	err := migrate.SQLite(ctx, db, "migrations")
func SQLite(ctx context.Context, db *sql.DB, dir string) error {
	return Up(ctx, db, Config{
		Dir:     dir,
		Dialect: "sqlite3",
	})
}

// setupMigration applies common configuration for migration operations.
func setupMigration(cfg *Config) error {
	if cfg.Dir == "" {
		cfg.Dir = "migrations"
	}
	if cfg.Table != "" {
		goose.SetTableName(cfg.Table)
	}
	if cfg.Dialect != "" {
		if err := goose.SetDialect(cfg.Dialect); err != nil {
			return fmt.Errorf("invalid dialect: %w", err)
		}
	}
	if cfg.FS != nil {
		goose.SetBaseFS(cfg.FS)
	}
	return nil
}

func withConfiguredGoose(_ context.Context, _ *sql.DB, cfg *Config, fn func() error) error {
	migrateMu.Lock()
	defer migrateMu.Unlock()

	if err := setupMigration(cfg); err != nil {
		return err
	}

	if err := fn(); err != nil {
		return err
	}

	return nil
}
