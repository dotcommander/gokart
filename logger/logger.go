// Package logger provides structured logging utilities wrapping log/slog.
package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// Config configures structured logging behavior.
type Config struct {
	Level  string    // debug, info, warn, error (default: info)
	Format string    // json, text (default: json)
	Output io.Writer // default: os.Stderr
}

// New creates a new structured logger with sensible defaults.
//
// Default configuration:
//   - Level: info
//   - Format: json
//   - Output: os.Stderr
//
// Example:
//
//	log := logger.New(logger.Config{
//	    Level:  "debug",
//	    Format: "text",
//	})
//	log.Info("server started", "port", 8080)
func New(cfg Config) *slog.Logger {
	level := parseLevel(cfg.Level)

	output := cfg.Output
	if output == nil {
		output = os.Stderr
	}

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

// NewDefault creates a logger with default settings (info level, JSON format, stderr).
func NewDefault() *slog.Logger {
	return New(Config{})
}

func parseLevel(level string) slog.Level {
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

// NewFile creates a logger that writes to a temp file.
//
// This is useful for TUI applications where stdout must remain clean.
// The logger writes JSON-formatted structured logs.
//
// Returns the logger, a cleanup function to close the file, and any error.
// The cleanup function should be deferred immediately after calling this.
//
// Log file location: {os.TempDir()}/{appName}.log
//
// Example:
//
//	log, cleanup, err := logger.NewFile("myapp")
//	if err != nil {
//	    return err
//	}
//	defer cleanup()
//
//	log.Info("application started")
//	// Logs written to /tmp/myapp.log (or equivalent)
func NewFile(appName string) (*slog.Logger, func(), error) {
	path := Path(appName)

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, func() {}, err
	}

	handler := slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)

	cleanup := func() {
		file.Close()
	}

	return logger, cleanup, nil
}

// Path returns the path where file logs are written.
//
// Example:
//
//	path := logger.Path("myapp")
//	// Returns: /tmp/myapp.log (on Unix systems)
func Path(appName string) string {
	return filepath.Join(os.TempDir(), appName+".log")
}
