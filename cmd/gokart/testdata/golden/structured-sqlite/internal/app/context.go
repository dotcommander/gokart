package app

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/dotcommander/gokart/sqlite"

	"github.com/spf13/viper"
)

// Config key constants — single source of truth between viper defaults and readers.
const (
	AppConfigKeyDBPath       = "db_path"
)

// Context holds shared application dependencies.
type Context struct {
	Log *slog.Logger
	DB  *sql.DB
}

// New creates a new application context with config from viper.
func New(ctx context.Context, appName string, v *viper.Viper) (*Context, error) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	appCtx := &Context{
		Log: logger,
	}

	// Setup SQLite database
	dbPath := v.GetString(AppConfigKeyDBPath)
	if dbPath == "" {
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			return nil, err
		}
		dbDir := filepath.Join(cacheDir, appName)
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return nil, err
		}
		dbPath = filepath.Join(dbDir, "data.db")
	}

	db, err := sqlite.Open(dbPath)
	if err != nil {
		return nil, err
	}
	appCtx.DB = db

	return appCtx, nil
}

// Close releases resources.
func (c *Context) Close() {
	if c.DB != nil {
		c.DB.Close()
	}
}