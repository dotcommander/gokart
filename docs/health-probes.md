# Health Probes

Kubernetes-style health and readiness probes using GoKart's built-in helpers.

## Built-in Helpers

Use `web.HealthHandler` and `web.ReadyHandler` for dependency-aware probes with parallel checks and automatic JSON responses.

```go
package main

import (
    "context"

    "github.com/dotcommander/gokart/web"
)

func main() {
    r := web.NewRouter(web.RouterConfig{
        Middleware: web.StandardMiddleware,
    })

    // Liveness: is the process alive?
    r.Get("/healthz", web.HealthHandler())

    // Readiness: checks all dependencies in parallel
    r.Get("/readyz", web.ReadyHandler(
        web.HealthCheck{Name: "database", Fn: pool.Ping},
        web.HealthCheck{Name: "cache", Fn: func(ctx context.Context) error {
            return redisClient.Ping(ctx).Err()
        }},
    ))

    web.ListenAndServe(":8080", r)
}
```

`web.ReadyHandler` runs all checks in parallel. If any check fails, the endpoint returns `503` with a JSON body identifying the failing dependency.

## Manual Pattern

If you need full control over the response format:

```go
import (
    "net/http"
    "sync/atomic"

    "github.com/dotcommander/gokart/web"
)

func main() {
    var ready atomic.Bool

    r := web.NewRouter(web.RouterConfig{
        Middleware: web.StandardMiddleware,
    })

    r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("ok"))
    })

    r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
        if !ready.Load() {
            http.Error(w, "not ready", http.StatusServiceUnavailable)
            return
        }
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("ok"))
    })

    ready.Store(true)
    web.ListenAndServe(":8080", r)
}
```

## Kubernetes Manifests

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /readyz
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```
