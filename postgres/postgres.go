// Package postgres provides PostgreSQL utilities wrapping pgx/v5.
package postgres

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/dotcommander/gokart/internal/sqltx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config configures PostgreSQL connection pooling.
type Config struct {
	// URL is the connection string (required).
	// Format: postgres://user:password@host:port/database?sslmode=disable
	URL string `config:"url"`

	Host     string `config:"host" default:"localhost"`
	Port     int    `config:"port" default:"5432"`
	User     string `config:"user" default:"postgres"`
	Password string `config:"password"`
	DBName   string `config:"dbname" default:"postgres"`
	SSLMode  string `config:"sslmode" default:"disable"`

	// ConnectionString is retained for config-map compatibility.
	// Deprecated: use URL.
	ConnectionString string `config:"connection_string"`

	// MaxConns is the maximum number of connections in the pool.
	// Default: 25
	MaxConns int32 `config:"max_conns"`

	// MinConns is the minimum number of connections to keep open.
	// Default: 5
	MinConns int32 `config:"min_conns"`

	// MaxConnLifetime is how long a connection can be reused.
	// Default: 1 hour
	MaxConnLifetime time.Duration `config:"max_conn_lifetime"`

	// MaxConnIdleTime is how long a connection can be idle before closing.
	// Default: 30 minutes
	MaxConnIdleTime time.Duration `config:"max_conn_idle_time"`

	// HealthCheckPeriod is how often to check connection health.
	// Default: 1 minute
	HealthCheckPeriod time.Duration `config:"health_check_period"`
}

// PostgresConfig is retained for compatibility with applications that used
// the longer historical name.
type PostgresConfig = Config

// DSN returns URL when set, then ConnectionString, otherwise safely assembles
// a PostgreSQL URL from the discrete connection fields.
func (c Config) DSN() string {
	if c.URL != "" {
		return c.URL
	}
	if c.ConnectionString != "" {
		return c.ConnectionString
	}
	dsn := &url.URL{
		Scheme:  "postgres",
		User:    url.UserPassword(c.User, c.Password),
		Host:    net.JoinHostPort(c.Host, strconv.Itoa(c.Port)),
		Path:    "/" + c.DBName,
		RawPath: "/" + url.PathEscape(c.DBName),
	}
	query := dsn.Query()
	query.Set("sslmode", c.SSLMode)
	dsn.RawQuery = query.Encode()
	return dsn.String()
}

// BuildConnectionString is a compatibility alias for DSN.
func (c Config) BuildConnectionString() string { return c.DSN() }

// DefaultConfig returns production-ready defaults.
func DefaultConfig(url string) Config {
	return Config{
		URL:               url,
		MaxConns:          25,
		MinConns:          5,
		MaxConnLifetime:   time.Hour,
		MaxConnIdleTime:   30 * time.Minute,
		HealthCheckPeriod: time.Minute,
	}
}

// Open opens a PostgreSQL connection pool with default settings.
//
// Example:
//
//	pool, err := postgres.Open(ctx, "postgres://user:pass@localhost:5432/mydb")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer pool.Close()
//
//	var name string
//	err = pool.QueryRow(ctx, "SELECT name FROM users WHERE id = $1", 1).Scan(&name)
func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	return OpenWithConfig(ctx, DefaultConfig(url))
}

// OpenWithConfig opens a PostgreSQL connection pool with custom settings.
//
// Example:
//
//	pool, err := postgres.OpenWithConfig(ctx, postgres.Config{
//	    URL:      "postgres://user:pass@localhost:5432/mydb",
//	    MaxConns: 50,
//	    MinConns: 10,
//	})
func OpenWithConfig(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("invalid postgres config: %w", err)
	}

	applyPoolConfig(poolCfg, cfg)

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}

// NewPool constructs a pool without pinging it. Use OpenWithConfig when
// startup must verify database reachability.
func (c Config) NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(c.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse connection string: %w", err)
	}
	applyPoolConfig(poolCfg, c)
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}
	return pool, nil
}

func applyPoolConfig(poolCfg *pgxpool.Config, cfg Config) {
	defaults := DefaultConfig("")
	if cfg.MaxConns <= 0 {
		cfg.MaxConns = defaults.MaxConns
	}
	if cfg.MinConns <= 0 {
		cfg.MinConns = defaults.MinConns
	}
	if cfg.MaxConnLifetime <= 0 {
		cfg.MaxConnLifetime = defaults.MaxConnLifetime
	}
	if cfg.MaxConnIdleTime <= 0 {
		cfg.MaxConnIdleTime = defaults.MaxConnIdleTime
	}
	if cfg.HealthCheckPeriod <= 0 {
		cfg.HealthCheckPeriod = defaults.HealthCheckPeriod
	}
	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolCfg.HealthCheckPeriod = cfg.HealthCheckPeriod
}

// FromEnv opens a PostgreSQL pool using DATABASE_URL environment variable.
//
// Example:
//
//	// Reads DATABASE_URL from environment
//	pool, err := postgres.FromEnv(ctx)
func FromEnv(ctx context.Context) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig("")
	if err != nil {
		return nil, fmt.Errorf("parse postgres env config: %w", err)
	}
	applyPoolConfig(poolCfg, Config{})
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool from env: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}

// Transaction executes a function within a PostgreSQL transaction.
// Automatically commits on success, rolls back on error or panic.
//
// Example:
//
//	err := postgres.Transaction(ctx, pool, func(tx pgx.Tx) error {
//	    _, err := tx.Exec(ctx, "INSERT INTO users (name) VALUES ($1)", "John")
//	    if err != nil {
//	        return err
//	    }
//	    _, err = tx.Exec(ctx, "UPDATE accounts SET balance = balance - $1 WHERE user_id = $2", 100, 1)
//	    return err
//	})
func Transaction(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	return sqltx.Run(
		func() (pgx.Tx, error) {
			return pool.Begin(ctx)
		},
		func(tx pgx.Tx) error {
			return tx.Rollback(ctx)
		},
		func(tx pgx.Tx) error {
			return tx.Commit(ctx)
		},
		fn,
	)
}
