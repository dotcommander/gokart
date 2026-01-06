package gokart

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

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
//	router := gokart.NewRouter(gokart.RouterConfig{
//	    Middleware: gokart.StandardMiddleware,
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
