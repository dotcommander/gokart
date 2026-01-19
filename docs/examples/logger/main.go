// Example: Structured logging with GoKart logger.
//
// This example demonstrates:
//   - Creating loggers with custom configurations
//   - JSON vs text formatting
//   - Log levels (debug, info, warn, error)
//   - Structured logging with key-value pairs
package main

import (
	"log/slog"
	"os"

	"github.com/dotcommander/gokart/logger"
)

func main() {
	// Example 1: Default logger (JSON format, info level)
	// This is what you'd use in production
	defaultLog := logger.NewDefault()
	defaultLog.Info("Application started", "version", "1.0.0")

	// Example 2: Text format logger (useful for development)
	devLog := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
	})

	devLog.Debug("Detailed debug info", "user_id", 123, "action", "login")
	devLog.Info("Processing request", "endpoint", "/api/users")
	devLog.Warn("High memory usage", "usage_mb", 450)
	devLog.Error("Database connection failed", "err", "connection timeout")

	// Example 3: JSON format logger (for log aggregation systems)
	jsonLog := logger.New(logger.Config{
		Level:  "info",
		Format: "json",
	})

	jsonLog.Info("User created", "user_id", 456, "email", "user@example.com")
	jsonLog.Error("Payment failed", "order_id", 789, "reason", "insufficient funds")

	// Example 4: File logger (useful for CLI apps with TUIs)
	// Creates a temp file and returns a cleanup function
	fileLog, cleanup, err := logger.NewFile("myapp")
	if err != nil {
		slog.Error("Failed to create file logger", "err", err)
		os.Exit(1)
	}
	defer cleanup() // Close file when done

	fileLog.Info("This goes to a file", "path", logger.Path("myapp"))
	fileLog.Debug("File logs keep stdout clean for spinners and tables")

	// Example 5: Using slog directly with GoKart
	// Since GoKart returns *slog.Logger, you can use all slog features
	attr := slog.Group("request",
		slog.String("method", "GET"),
		slog.String("path", "/api/users"),
		slog.Int("status", 200),
	)
	defaultLog.Info("Request completed", attr)
}
