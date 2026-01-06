package gokart

import (
	"log/slog"

	"github.com/dotcommander/gokart/logger"
)

// LogConfig is an alias for logger.Config.
// Deprecated: Use logger.Config directly.
type LogConfig = logger.Config

// NewLogger creates a new structured logger with sensible defaults.
// Deprecated: Use logger.New directly.
func NewLogger(cfg LogConfig) *slog.Logger {
	return logger.New(cfg)
}

// NewFileLogger creates a logger that writes to a temp file.
// Deprecated: Use logger.NewFile directly.
func NewFileLogger(appName string) (*slog.Logger, func(), error) {
	return logger.NewFile(appName)
}

// LogPath returns the path where file logs are written.
// Deprecated: Use logger.Path directly.
func LogPath(appName string) string {
	return logger.Path(appName)
}
