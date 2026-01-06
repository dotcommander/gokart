package gokart

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// LogConfig configures structured logging behavior.
type LogConfig struct {
	Level  string    // debug, info, warn, error (default: info)
	Format string    // json, text (default: json)
	Output io.Writer // default: os.Stderr
}

// NewLogger creates a new structured logger with sensible defaults.
//
// Default configuration:
//   - Level: info
//   - Format: json
//   - Output: os.Stderr
//
// Example:
//
//	log := gokart.NewLogger(gokart.LogConfig{
//	    Level:  "debug",
//	    Format: "text",
//	})
//	log.Info("server started", "port", 8080)
func NewLogger(cfg LogConfig) *slog.Logger {
	// Default to info level
	level := parseLogLevel(cfg.Level)

	// Default to os.Stderr
	output := cfg.Output
	if output == nil {
		output = os.Stderr
	}

	// Default to JSON format
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: level,
	}

	switch strings.ToLower(cfg.Format) {
	case "text":
		handler = slog.NewTextHandler(output, opts)
	default:
		handler = slog.NewJSONHandler(output, opts)
	}

	return slog.New(handler)
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
