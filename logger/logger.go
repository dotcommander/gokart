// Package logger provides structured logging utilities wrapping log/slog.
package logger

import (
	"fmt"
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
	if appName == "" || appName == "." || appName == ".." || appName != filepath.Base(appName) || strings.ContainsAny(appName, `/\\`) || strings.ContainsRune(appName, 0) {
		return nil, func() {}, fmt.Errorf("invalid app name %q", appName)
	}
	path := Path(appName)

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, func() {}, fmt.Errorf("open log file %s: %w", path, err)
	}
	// Tighten permissions on pre-existing files (created 0644 by older
	// versions): logs can carry tokens and user data, so keep them
	// owner-only even when the file already exists.
	if err := file.Chmod(0600); err != nil {
		_ = file.Close()
		return nil, func() {}, fmt.Errorf("secure log file %s: %w", path, err)
	}

	handler := slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)

	cleanup := func() {
		if err := file.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "logger: close log file %s: %v\n", path, err)
		}
	}

	return logger, cleanup, nil
}

// SetTUIHandler installs a non-stderr slog default handler for
// terminal-owned UIs (bubbletea, tview, tcell, termui).
//
// Go's slog default handler writes to stderr, so a single slog.Warn
// or slog.Info at runtime splatters log text across the rendered
// frame. Call SetTUIHandler before starting the TUI and keep it
// installed for the TUI's lifetime. It opens the same file as
// NewFile (os.TempDir()/appName.log, 0600) and installs it as the
// default via slog.SetDefault, so library and third-party code
// calling slog.Default() is redirected too. If the file cannot be
// opened, it installs an io.Discard handler instead of leaving
// slog pointed at stderr.
//
// Call only in interactive/TUI mode; headless (--print,
// --non-interactive) runs should keep stderr so operators see errors.
// The returned cleanup function closes the file and must be deferred
// immediately after checking the error.
//
// Example:
//
//	cleanup, err := logger.SetTUIHandler("myapp")
//	if err != nil {
//	    fmt.Fprintln(os.Stderr, "warn: logging disabled:", err)
//	}
//	defer cleanup()
func SetTUIHandler(appName string) (func(), error) {
	log, cleanup, err := NewFile(appName)
	if err != nil {
		slog.SetDefault(slog.New(slog.NewJSONHandler(io.Discard, nil)))
		return func() {}, err
	}
	slog.SetDefault(log)
	return cleanup, nil
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
