package gokart

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"sync"

	"github.com/pressly/goose/v3"
)

// migrateMu protects goose global state during concurrent migrations.
var migrateMu sync.Mutex

// MigrateConfig configures database migrations.
type MigrateConfig struct {
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

// DefaultMigrateConfig returns sensible defaults.
func DefaultMigrateConfig() MigrateConfig {
	return MigrateConfig{
		Dir:   "migrations",
		Table: "goose_db_version",
	}
}

// Migrate runs all pending migrations.
//
// Example with file-based migrations:
//
//	db, _ := gokart.OpenPostgres(ctx, url)
//	err := gokart.Migrate(ctx, db.Config().ConnConfig.Database, gokart.MigrateConfig{
//	    Dir:     "migrations",
//	    Dialect: "postgres",
//	})
//
// Example with embedded migrations:
//
//	//go:embed migrations/*.sql
//	var migrations embed.FS
//
//	err := gokart.Migrate(ctx, db, gokart.MigrateConfig{
//	    FS:      migrations,
//	    Dir:     "migrations",
//	    Dialect: "postgres",
//	})
func Migrate(ctx context.Context, db *sql.DB, cfg MigrateConfig) error {
	migrateMu.Lock()
	defer migrateMu.Unlock()

	if err := setupMigration(&cfg); err != nil {
		return err
	}

	if err := goose.UpContext(ctx, db, cfg.Dir); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}

// MigrateUp runs all pending migrations.
func MigrateUp(ctx context.Context, db *sql.DB, cfg MigrateConfig) error {
	return Migrate(ctx, db, cfg)
}

// MigrateDown rolls back the last migration.
func MigrateDown(ctx context.Context, db *sql.DB, cfg MigrateConfig) error {
	migrateMu.Lock()
	defer migrateMu.Unlock()

	if err := setupMigration(&cfg); err != nil {
		return err
	}

	if err := goose.DownContext(ctx, db, cfg.Dir); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	return nil
}

// MigrateDownTo rolls back to a specific version.
func MigrateDownTo(ctx context.Context, db *sql.DB, cfg MigrateConfig, version int64) error {
	migrateMu.Lock()
	defer migrateMu.Unlock()

	if err := setupMigration(&cfg); err != nil {
		return err
	}

	if err := goose.DownToContext(ctx, db, cfg.Dir, version); err != nil {
		return fmt.Errorf("rollback to version %d failed: %w", version, err)
	}

	return nil
}

// MigrateReset rolls back all migrations.
func MigrateReset(ctx context.Context, db *sql.DB, cfg MigrateConfig) error {
	return MigrateDownTo(ctx, db, cfg, 0)
}

// MigrateStatus prints the status of all migrations.
func MigrateStatus(ctx context.Context, db *sql.DB, cfg MigrateConfig) error {
	migrateMu.Lock()
	defer migrateMu.Unlock()

	if err := setupMigration(&cfg); err != nil {
		return err
	}

	if err := goose.StatusContext(ctx, db, cfg.Dir); err != nil {
		return fmt.Errorf("status failed: %w", err)
	}

	return nil
}

// MigrateVersion returns the current migration version.
func MigrateVersion(ctx context.Context, db *sql.DB, cfg MigrateConfig) (int64, error) {
	migrateMu.Lock()
	defer migrateMu.Unlock()

	if err := setupMigration(&cfg); err != nil {
		return 0, err
	}

	version, err := goose.GetDBVersionContext(ctx, db)
	if err != nil {
		return 0, fmt.Errorf("failed to get version: %w", err)
	}

	return version, nil
}

// MigrateCreate creates a new migration file.
//
// Example:
//
//	err := gokart.MigrateCreate("migrations", "add_users_table", "sql")
func MigrateCreate(dir, name, migrationType string) error {
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

// PostgresMigrate is a convenience function for PostgreSQL migrations.
//
// Example:
//
//	pool, _ := gokart.OpenPostgres(ctx, url)
//	db := stdlib.OpenDBFromPool(pool)
//	err := gokart.PostgresMigrate(ctx, db, "migrations")
func PostgresMigrate(ctx context.Context, db *sql.DB, dir string) error {
	return Migrate(ctx, db, MigrateConfig{
		Dir:     dir,
		Dialect: "postgres",
	})
}

// SQLiteMigrate is a convenience function for SQLite migrations.
//
// Example:
//
//	db, _ := gokart.OpenSQLite("app.db")
//	err := gokart.SQLiteMigrate(ctx, db, "migrations")
func SQLiteMigrate(ctx context.Context, db *sql.DB, dir string) error {
	return Migrate(ctx, db, MigrateConfig{
		Dir:     dir,
		Dialect: "sqlite3",
	})
}

// setupMigration applies common configuration for migration operations.
func setupMigration(cfg *MigrateConfig) error {
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
