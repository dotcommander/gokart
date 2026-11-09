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
- retries: `web.RetryTransport` (operation-owned; see below)
- rate limiting: `golang.org/x/time/rate`
- templates: templ's native component and handler APIs

## Outbound HTTP clients

`web.NewClient()` returns an `*http.Client` backed by a production-tuned transport: `http.DefaultTransport` cloned, with `MaxIdleConnsPerHost` raised from the Go default of 2 to 100 (~7x throughput on a saturated upstream) and `MaxIdleConns`/`IdleConnTimeout` set for connection reuse. `web.Transport()` exposes the transport for custom clients.

```go
client := web.NewClient() // create once per service, reuse for every request
```

Clients own their connection pool — creating one per request or per goroutine defeats pooling entirely. Never build `&http.Transport{}` from scratch to change one field: the zero value silently drops TLS timeouts, HTTP/2, and proxy settings. `web.Transport()` clones the default so those production-safe settings are inherited.

## Retries (operation-owned)

Retries belong at the operation layer: idempotency depends on the method, body replayability, idempotency keys, and response classification — never automatic retries on every request. `web.RetryTransport` is an explicit `RoundTripper` you opt into per operation:

```go
client := web.NewRetryClient(&web.RetryTransport{
    Next:     web.Transport(),
    Attempts: 3,
    Initial:  100 * time.Millisecond,
    MaxDelay: 2 * time.Second,
})
```

- **Classification:** `web.ShouldRetryHTTP` retries network errors, 408, 429, and 5xx; other 4xx are permanent caller errors. Custom policies implement `web.RetryPolicy`.
- **Jitter (required):** every retry waits `base × (0.5 + uniform(0, 0.25))`, where `base = min(Initial × 2^(attempt-1), MaxDelay)` on attempt `attempt` (first retry waits at least `Initial/2`, at most `3/4 × base`) — without jitter, simultaneous failures retry in lockstep and amplify the outage. The factor hook `JitterRand` overrides the uniform draw when set.
- **Replayability:** the wrapper re-sends the same `*http.Request`; the caller must supply a request whose body can be replayed (e.g. `bytes.NewReader`, `http.NewRequestWithContext` + `GetBody`). Non-replayable bodies must be handled by the operation itself.

## See Also

- [Response helpers](response.md)
- [Validation](validate.md)
