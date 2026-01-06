package gokart

import (
	"context"

	"github.com/dotcommander/gokart/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresConfig is an alias for postgres.Config.
// Deprecated: Use postgres.Config directly.
type PostgresConfig = postgres.Config

// DefaultPostgresConfig returns production-ready defaults.
// Deprecated: Use postgres.DefaultConfig directly.
func DefaultPostgresConfig(url string) PostgresConfig {
	return postgres.DefaultConfig(url)
}

// OpenPostgres opens a PostgreSQL connection pool with default settings.
// Deprecated: Use postgres.Open directly.
func OpenPostgres(ctx context.Context, url string) (*pgxpool.Pool, error) {
	return postgres.Open(ctx, url)
}

// OpenPostgresWithConfig opens a PostgreSQL connection pool with custom settings.
// Deprecated: Use postgres.OpenWithConfig directly.
func OpenPostgresWithConfig(ctx context.Context, cfg PostgresConfig) (*pgxpool.Pool, error) {
	return postgres.OpenWithConfig(ctx, cfg)
}

// PostgresFromEnv opens a PostgreSQL pool using DATABASE_URL environment variable.
// Deprecated: Use postgres.FromEnv directly.
func PostgresFromEnv(ctx context.Context) (*pgxpool.Pool, error) {
	return postgres.FromEnv(ctx)
}

// WithTransaction executes a function within a PostgreSQL transaction.
// Deprecated: Use postgres.Transaction directly.
func WithTransaction(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	return postgres.Transaction(ctx, pool, fn)
}
