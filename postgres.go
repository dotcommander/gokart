package gokart

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresConfig configures PostgreSQL connection pooling.
type PostgresConfig struct {
	// URL is the connection string (required).
	// Format: postgres://user:password@host:port/database?sslmode=disable
	URL string

	// MaxConns is the maximum number of connections in the pool.
	// Default: 25
	MaxConns int32

	// MinConns is the minimum number of connections to keep open.
	// Default: 5
	MinConns int32

	// MaxConnLifetime is how long a connection can be reused.
	// Default: 1 hour
	MaxConnLifetime time.Duration

	// MaxConnIdleTime is how long a connection can be idle before closing.
	// Default: 30 minutes
	MaxConnIdleTime time.Duration

	// HealthCheckPeriod is how often to check connection health.
	// Default: 1 minute
	HealthCheckPeriod time.Duration
}

// DefaultPostgresConfig returns production-ready defaults.
func DefaultPostgresConfig(url string) PostgresConfig {
	return PostgresConfig{
		URL:               url,
		MaxConns:          25,
		MinConns:          5,
		MaxConnLifetime:   time.Hour,
		MaxConnIdleTime:   30 * time.Minute,
		HealthCheckPeriod: time.Minute,
	}
}

// OpenPostgres opens a PostgreSQL connection pool with default settings.
//
// Example:
//
//	pool, err := gokart.OpenPostgres(ctx, "postgres://user:pass@localhost:5432/mydb")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer pool.Close()
//
//	var name string
//	err = pool.QueryRow(ctx, "SELECT name FROM users WHERE id = $1", 1).Scan(&name)
func OpenPostgres(ctx context.Context, url string) (*pgxpool.Pool, error) {
	return OpenPostgresWithConfig(ctx, DefaultPostgresConfig(url))
}

// OpenPostgresWithConfig opens a PostgreSQL connection pool with custom settings.
//
// Example:
//
//	pool, err := gokart.OpenPostgresWithConfig(ctx, gokart.PostgresConfig{
//	    URL:      "postgres://user:pass@localhost:5432/mydb",
//	    MaxConns: 50,
//	    MinConns: 10,
//	})
func OpenPostgresWithConfig(ctx context.Context, cfg PostgresConfig) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid postgres URL: %w", err)
	}

	// Apply configuration
	if cfg.MaxConns > 0 {
		poolCfg.MaxConns = cfg.MaxConns
	}
	if cfg.MinConns > 0 {
		poolCfg.MinConns = cfg.MinConns
	}
	if cfg.MaxConnLifetime > 0 {
		poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	}
	if cfg.MaxConnIdleTime > 0 {
		poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	}
	if cfg.HealthCheckPeriod > 0 {
		poolCfg.HealthCheckPeriod = cfg.HealthCheckPeriod
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return pool, nil
}

// PostgresFromEnv opens a PostgreSQL pool using DATABASE_URL environment variable.
//
// Example:
//
//	// Reads DATABASE_URL from environment
//	pool, err := gokart.PostgresFromEnv(ctx)
func PostgresFromEnv(ctx context.Context) (*pgxpool.Pool, error) {
	// pgxpool.New automatically reads from DATABASE_URL
	pool, err := pgxpool.New(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres pool from env: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return pool, nil
}

// WithTransaction executes a function within a PostgreSQL transaction.
// Automatically commits on success, rolls back on error or panic.
//
// Example:
//
//	err := gokart.WithTransaction(ctx, pool, func(tx pgx.Tx) error {
//	    _, err := tx.Exec(ctx, "INSERT INTO users (name) VALUES ($1)", "John")
//	    if err != nil {
//	        return err
//	    }
//	    _, err = tx.Exec(ctx, "UPDATE accounts SET balance = balance - $1 WHERE user_id = $2", 100, 1)
//	    return err
//	})
func WithTransaction(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
