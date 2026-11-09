# Web

The `web` module owns only recurring HTTP setup: chi router/server construction, JSON responses, bounded JSON binding, and validation.

```go
router := web.NewRouter(web.RouterConfig{})
router.Post("/users", func(w http.ResponseWriter, r *http.Request) {
    var input CreateUser
    if err := web.BindJSON(r, &input); err != nil {
        web.Error(w, http.StatusBadRequest, "invalid request")
        return
    }
    web.JSONStatus(w, http.StatusCreated, input)
})
```

Use upstream facilities directly for removed policy surfaces:

- static assets: `http.FileServer`
- auth, sessions, and CSRF: caller-selected middleware
- pagination: caller-owned request and response types
- retries: `net/http` or `go-retryablehttp`
- rate limiting: `golang.org/x/time/rate`
- templates: templ's native component and handler APIs

## Outbound HTTP clients

`web.NewClient()` returns an `*http.Client` backed by a production-tuned transport: `http.DefaultTransport` cloned, with `MaxIdleConnsPerHost` raised from the Go default of 2 to 100 (~7x throughput on a saturated upstream) and `MaxIdleConns`/`IdleConnTimeout` set for connection reuse. `web.Transport()` exposes the transport for custom clients.

```go
client := web.NewClient() // create once per service, reuse for every request
```

Clients own their connection pool — creating one per request or per goroutine defeats pooling entirely. Never build `&http.Transport{}` from scratch to change one field: the zero value silently drops TLS timeouts, HTTP/2, and proxy settings. `web.Transport()` clones the default so those production-safe settings are inherited.

## See Also

- [Response helpers](response.md)
- [Validation](validate.md)
