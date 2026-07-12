package app

import (
	"context"
	"log/slog"
	"os"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	"github.com/spf13/viper"
)

// Config key constants — single source of truth between viper defaults and readers.
const (
	AppConfigKeyOpenAIAPIKey = "openai_api_key"
)

// Context holds shared application dependencies.
type Context struct {
	Log *slog.Logger
	AI  openai.Client
}

// New creates a new application context with config from viper.
func New(ctx context.Context, appName string, v *viper.Viper) (*Context, error) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	appCtx := &Context{
		Log: logger,
	}

	// Setup OpenAI client
	apiKey := v.GetString(AppConfigKeyOpenAIAPIKey)
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if apiKey == "" {
		logger.Warn("OPENAI_API_KEY not set, AI features will not work")
	}
	appCtx.AI = openai.NewClient(option.WithAPIKey(apiKey))

	return appCtx, nil
}

// Close releases resources.
func (c *Context) Close() {
}