package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/dotcommander/gokart/cache"

	"github.com/spf13/viper"
)

// Config key constants — single source of truth between viper defaults and readers.
const (
	AppConfigKeyRedisAddr = "redis_addr"
)

// Context holds shared application dependencies.
type Context struct {
	Log   *slog.Logger
	Cache *cache.Cache
}

// New creates a new application context with config from viper.
func New(ctx context.Context, appName string, v *viper.Viper) (*Context, error) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	appCtx := &Context{
		Log: logger,
	}

	// Setup Redis cache
	redisAddr := v.GetString(AppConfigKeyRedisAddr)
	if redisAddr == "" {
		redisAddr = os.Getenv("REDIS_ADDR")
	}
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	cacheClient, err := cache.Open(ctx, redisAddr)
	if err != nil {
		return nil, fmt.Errorf("open cache: %w", err)
	}
	appCtx.Cache = cacheClient

	return appCtx, nil
}

// Close releases resources.
func (c *Context) Close() {
	if c.Cache != nil {
		c.Cache.Close()
	}
}
