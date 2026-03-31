package web

import (
	"context"
	"log/slog"
	"net/http"
	"sync"

	"golang.org/x/sync/errgroup"
)

// HealthFunc checks the health of a single dependency.
type HealthFunc func(ctx context.Context) error

// HealthCheck pairs a name with its health-check function.
type HealthCheck struct {
	Name string
	Fn   HealthFunc
}

// healthResult holds the response payload returned by both
// HealthHandler and ReadyHandler.
type healthResult struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

// runChecks executes all checks in parallel, recovering from panics.
// Returns the result map and counts of passes and failures.
func runChecks(ctx context.Context, checks []HealthCheck) (map[string]string, int, int) {
	results := make(map[string]string, len(checks))
	passed, failed := 0, 0

	var mu sync.Mutex
	g, gCtx := errgroup.WithContext(ctx)

	for _, c := range checks {
		c := c
		g.Go(func() error {
			ok := true
			func() {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("health check panicked", "check", c.Name, "panic", r)
						ok = false
					}
				}()
				if err := c.Fn(gCtx); err != nil {
					ok = false
					mu.Lock()
					results[c.Name] = err.Error()
					mu.Unlock()
				}
			}()

			mu.Lock()
			defer mu.Unlock()
			if ok {
				results[c.Name] = "ok"
				passed++
			} else if results[c.Name] == "" {
				results[c.Name] = "error"
				failed++
			} else {
				failed++
			}

			return nil
		})
	}
	_ = g.Wait()

	return results, passed, failed
}

// HealthHandler returns an http.HandlerFunc that runs the given health checks.
//
// Response status codes:
//   - 200 with {"status":"ok"}        — all checks pass
//   - 200 with {"status":"degraded"}  — some checks fail
//   - 503 with {"status":"unhealthy"} — all checks fail
//
// Example:
//
//	h := web.HealthHandler(
//	    web.HealthCheck{Name: "database", Fn: db.Ping},
//	    web.HealthCheck{Name: "cache", Fn: cache.Ping},
//	)
//	r.Get("/health", h)
func HealthHandler(checks ...HealthCheck) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		results, passed, failed := runChecks(r.Context(), checks)

		switch {
		case failed == 0:
			JSONStatus(w, http.StatusOK, healthResult{Status: "ok", Checks: results})
		case passed > 0:
			JSONStatus(w, http.StatusOK, healthResult{Status: "degraded", Checks: results})
		default:
			JSONStatus(w, http.StatusServiceUnavailable, healthResult{Status: "unhealthy", Checks: results})
		}
	}
}

// ReadyHandler returns an http.HandlerFunc for readiness probes.
// Returns 200 only if ALL checks pass, 503 otherwise.
//
// Example:
//
//	r.Get("/ready", web.ReadyHandler(
//	    web.HealthCheck{Name: "database", Fn: db.Ping},
//	))
func ReadyHandler(checks ...HealthCheck) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		results, _, failed := runChecks(r.Context(), checks)

		if failed > 0 {
			JSONStatus(w, http.StatusServiceUnavailable, healthResult{Status: "unhealthy", Checks: results})
			return
		}
		JSONStatus(w, http.StatusOK, healthResult{Status: "ok", Checks: results})
	}
}
