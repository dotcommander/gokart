package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/pressly/goose/v3"
)

// Config configures database migrations.
type Config struct {
	// Dir is the directory containing migration files.
	// Default: "migrations"
	Dir string

	// Table is the name of the migrations tracking table.
	// Default: "goose_db_version"
	Table string

	// Dialect is the required database dialect (postgres, sqlite3, mysql).
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

func newGooseProvider(cfg Config, db *sql.DB) (*goose.Provider, error) {
	if cfg.Dir == "" {
		cfg.Dir = "migrations"
	}
	if cfg.Dialect == "" {
		return nil, fmt.Errorf("migration dialect is required")
	}

	var fsys fs.FS
	if cfg.FS != nil {
		var err error
		fsys, err = fs.Sub(cfg.FS, cfg.Dir)
		if err != nil {
			return nil, fmt.Errorf("open migration directory %q: %w", cfg.Dir, err)
		}
	} else {
		fsys = os.DirFS(cfg.Dir)
	}

	opts := make([]goose.ProviderOption, 0, 3)
	if cfg.Table != "" {
		opts = append(opts, goose.WithTableName(cfg.Table))
	}
	if cfg.AllowMissing {
		opts = append(opts, goose.WithAllowOutofOrder(true))
	}
	if cfg.NoVersioning {
		opts = append(opts, goose.WithDisableVersioning(true))
	}

	provider, err := goose.NewProvider(goose.Dialect(cfg.Dialect), db, fsys, opts...)
	if err != nil {
		return nil, fmt.Errorf("configure migration provider: %w", err)
	}
	return provider, nil
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
	provider, err := newGooseProvider(cfg, db)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}
	return nil
}

// Down rolls back the last migration.
func Down(ctx context.Context, db *sql.DB, cfg Config) error {
	provider, err := newGooseProvider(cfg, db)
	if err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}
	if _, err := provider.Down(ctx); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}
	return nil
}

// DownTo rolls back to a specific version.
func DownTo(ctx context.Context, db *sql.DB, cfg Config, version int64) error {
	provider, err := newGooseProvider(cfg, db)
	if err != nil {
		return fmt.Errorf("rollback to version %d failed: %w", version, err)
	}
	if _, err := provider.DownTo(ctx, version); err != nil {
		return fmt.Errorf("rollback to version %d failed: %w", version, err)
	}
	return nil
}

// Reset rolls back all migrations.
func Reset(ctx context.Context, db *sql.DB, cfg Config) error {
	return DownTo(ctx, db, cfg, 0)
}

// MigrationStatus reports whether one discovered migration has been applied.
type MigrationStatus struct {
	Version   int64
	Applied   bool
	AppliedAt time.Time
}

// Status verifies that migration status can be loaded. It is retained for
// compatibility; use MigrationStatuses when the caller needs the results.
func Status(ctx context.Context, db *sql.DB, cfg Config) error {
	_, err := MigrationStatuses(ctx, db, cfg)
	return err
}

// MigrationStatuses returns the status of every discovered migration in
// provider order.
func MigrationStatuses(ctx context.Context, db *sql.DB, cfg Config) ([]MigrationStatus, error) {
	provider, err := newGooseProvider(cfg, db)
	if err != nil {
		return nil, fmt.Errorf("status failed: %w", err)
	}
	providerStatuses, err := provider.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("status failed: %w", err)
	}

	statuses := make([]MigrationStatus, 0, len(providerStatuses))
	for _, status := range providerStatuses {
		statuses = append(statuses, MigrationStatus{
			Version:   status.Source.Version,
			Applied:   status.State == goose.StateApplied,
			AppliedAt: status.AppliedAt,
		})
	}

	return statuses, nil
}

// Version returns the current migration version.
func Version(ctx context.Context, db *sql.DB, cfg Config) (int64, error) {
	provider, err := newGooseProvider(cfg, db)
	if err != nil {
		return 0, fmt.Errorf("failed to get version: %w", err)
	}
	version, err := provider.GetDBVersion(ctx)
	if err != nil {
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
