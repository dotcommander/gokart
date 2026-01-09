// Package gokart provides thin wrappers around battle-tested packages.
// This file provides HTTP server with graceful shutdown.
package gokart

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// ListenAndServe starts an HTTP server with graceful shutdown.
// Blocks until SIGINT or SIGTERM is received, then gracefully shuts down
// with a 30 second timeout.
func ListenAndServe(addr string, handler http.Handler) error {
	return ListenAndServeWithTimeout(addr, handler, 30*time.Second)
}

// ListenAndServeWithTimeout starts an HTTP server with graceful shutdown
// and a custom shutdown timeout.
func ListenAndServeWithTimeout(addr string, handler http.Handler, timeout time.Duration) error {
	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		slog.Info("server starting", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		slog.Info("shutting down", "signal", sig.String())
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "err", err)
		return err
	}

	slog.Info("server stopped")
	return nil
}
