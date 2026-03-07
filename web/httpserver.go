package web

import (
	"context"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// ServerConfig configures HTTP server behavior.
type ServerConfig struct {
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
	ShutdownTimeout   time.Duration
}

// DefaultServerConfig returns production-ready server defaults.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
		ShutdownTimeout:   30 * time.Second,
	}
}

// RouterConfig configures HTTP router behavior.
type RouterConfig struct {
	Middleware []func(http.Handler) http.Handler
	Timeout    time.Duration // request timeout (default: none)
}

// StandardMiddleware provides production-ready middleware stack:
//   - RequestID: Injects request ID for tracing
//   - RealIP: Extracts real client IP from proxies
//   - Logger: Structured request/response logging
//   - Recoverer: Panic recovery
var StandardMiddleware = []func(http.Handler) http.Handler{
	middleware.RequestID,
	middleware.RealIP,
	middleware.Logger,
	middleware.Recoverer,
}

// NewRouter creates a new chi router with configured middleware.
//
// Example:
//
//	router := web.NewRouter(web.RouterConfig{
//	    Middleware: web.StandardMiddleware,
//	    Timeout:    30 * time.Second,
//	})
//
//	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
//	    w.WriteHeader(http.StatusOK)
//	})
//
//	http.ListenAndServe(":8080", router)
func NewRouter(cfg RouterConfig) chi.Router {
	r := chi.NewRouter()

	// Apply middleware
	for _, mw := range cfg.Middleware {
		r.Use(mw)
	}

	// Apply timeout if configured
	if cfg.Timeout > 0 {
		r.Use(middleware.Timeout(cfg.Timeout))
	}

	return r
}

// Serve starts an HTTP server that shuts down when ctx is cancelled.
func Serve(ctx context.Context, addr string, handler http.Handler, cfg ServerConfig) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("server starting", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("shutting down")
	}

	shutdownTimeout := cfg.ShutdownTimeout
	if shutdownTimeout == 0 {
		shutdownTimeout = 30 * time.Second
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "err", err)
		return err
	}

	slog.Info("server stopped")
	return nil
}

// ListenAndServe starts an HTTP server with graceful shutdown.
// Blocks until SIGINT or SIGTERM is received, then gracefully shuts down
// with a 30-second timeout.
func ListenAndServe(addr string, handler http.Handler) error {
	return ListenAndServeWithTimeout(addr, handler, 30*time.Second)
}

// ListenAndServeWithTimeout starts an HTTP server with graceful shutdown
// and a custom shutdown timeout.
func ListenAndServeWithTimeout(addr string, handler http.Handler, timeout time.Duration) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := DefaultServerConfig()
	cfg.ShutdownTimeout = timeout
	return Serve(ctx, addr, handler, cfg)
}
