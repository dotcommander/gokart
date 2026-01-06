package gokart

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
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

// NewFileLogger creates a logger that writes to a temp file.
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
//	log, cleanup, err := gokart.NewFileLogger("myapp")
//	if err != nil {
//	    return err
//	}
//	defer cleanup()
//
//	log.Info("application started")
//	// Logs written to /tmp/myapp.log (or equivalent)
func NewFileLogger(appName string) (*slog.Logger, func(), error) {
	path := LogPath(appName)

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

// LogPath returns the path where file logs are written.
//
// Example:
//
//	path := gokart.LogPath("myapp")
//	// Returns: /tmp/myapp.log (on Unix systems)
func LogPath(appName string) string {
	return filepath.Join(os.TempDir(), appName+".log")
}
