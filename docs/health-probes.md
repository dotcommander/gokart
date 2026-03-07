# Health Probes

Example pattern for Kubernetes-style health and readiness probes using GoKart's web package.

## Usage

```go
package main

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

    // Liveness: is the process alive?
    r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("ok"))
    })

    // Readiness: is the service ready to accept traffic?
    r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
        if !ready.Load() {
            http.Error(w, "not ready", http.StatusServiceUnavailable)
            return
        }
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("ok"))
    })

    // Mark ready after initialization
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
