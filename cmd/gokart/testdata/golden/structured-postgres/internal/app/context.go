package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/spf13/viper"
)

// Config key constants — single source of truth between viper defaults and readers.
const (
	AppConfigKeyDatabaseURL  = "database_url"
)

// Context holds shared application dependencies.
type Context struct {
	Log *slog.Logger
	Pool *pgxpool.Pool
}

// New creates a new application context with config from viper.
func New(ctx context.Context, appName string, v *viper.Viper) (*Context, error) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	appCtx := &Context{
		Log: logger,
	}

	// Setup PostgreSQL connection pool
	dbURL := v.GetString(AppConfigKeyDatabaseURL)
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		logger.Warn("DATABASE_URL not set, PostgreSQL features will not work")
	} else {
		cfg, err := pgxpool.ParseConfig(dbURL)
		if err != nil {
			return nil, fmt.Errorf("parse postgres config: %w", err)
		}
		cfg.MaxConns = 25
		cfg.MinConns = 5
		cfg.MaxConnLifetime = time.Hour
		pool, err := pgxpool.NewWithConfig(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("connect to postgres: %w", err)
		}
		appCtx.Pool = pool
	}

	return appCtx, nil
}

// Close releases resources.
func (c *Context) Close() {
	if c.Pool != nil {
		c.Pool.Close()
	}
}