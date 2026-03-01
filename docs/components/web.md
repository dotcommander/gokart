# Web

Full HTTP toolkit for Go web applications and hypermedia-driven services. The `web` package collects everything a server-side rendered or API service needs: router, response helpers, HTML rendering, input binding, content negotiation, flash messages, CSRF protection, pagination, and static asset serving. Built on [chi](https://github.com/go-chi/chi) and [a-h/templ](https://github.com/a-h/templ).

## Installation

```bash
go get github.com/dotcommander/gokart/web
```

## Quick start

```go
import "github.com/dotcommander/gokart/web"

router := web.NewRouter(web.RouterConfig{
    Middleware: web.StandardMiddleware,
    Timeout:    30 * time.Second,
})

router.Get("/api/users", func(w http.ResponseWriter, r *http.Request) {
    users := db.ListUsers(r.Context())
    web.JSON(w, users)
})

router.Get("/users", func(w http.ResponseWriter, r *http.Request) {
    users := db.ListUsers(r.Context())
    web.Negotiate(w, r, users, views.UsersPage(users))
})

web.ListenAndServe(":8080", router)
```

---

## Router

### NewRouter

Creates a chi router with configured middleware.

```go
func NewRouter(cfg RouterConfig) chi.Router
```

```go
router := web.NewRouter(web.RouterConfig{
    Middleware: web.StandardMiddleware,
    Timeout:    30 * time.Second,
})
```

### RouterConfig

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Middleware` | `[]func(http.Handler) http.Handler` | `nil` | Middleware stack applied in order |
| `Timeout` | `time.Duration` | none | Per-request timeout (0 disables) |

### StandardMiddleware

Production-ready middleware stack:

```go
web.StandardMiddleware = []func(http.Handler) http.Handler{
    middleware.RequestID,  // injects X-Request-ID
    middleware.RealIP,     // extracts client IP from proxy headers
    middleware.Logger,     // logs requests with timing
    middleware.Recoverer,  // recovers panics with 500 response
}
```

---

## Server

### ListenAndServe

Starts the server and blocks until SIGINT or SIGTERM, then shuts down with a 30-second drain.

```go
func ListenAndServe(addr string, handler http.Handler) error
```

```go
if err := web.ListenAndServe(":8080", router); err != nil {
    log.Fatal(err)
}
```

### ListenAndServeWithTimeout

Same as `ListenAndServe` with a custom shutdown timeout.

```go
func ListenAndServeWithTimeout(addr string, handler http.Handler, timeout time.Duration) error
```

---

## HTTP Client

Wraps [hashicorp/go-retryablehttp](https://github.com/hashicorp/go-retryablehttp) to provide an HTTP client with automatic retries and exponential backoff.

### NewStandardClient

Returns a `*http.Client` with default configuration — drop-in replacement for `http.DefaultClient`.

```go
func NewStandardClient() *http.Client
```

```go
client := web.NewStandardClient()
resp, err := client.Get("https://api.example.com/data")
```

### HTTPConfig

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Timeout` | `time.Duration` | `30s` | Per-request timeout |
| `RetryMax` | `int` | `3` | Maximum retry attempts |
| `RetryWait` | `time.Duration` | `1s` | Base wait between retries; max wait is 10x this value |

### NewHTTPClient

Creates a retryable client with custom configuration. Returns `*retryablehttp.Client`, which exposes the full go-retryablehttp surface (custom retry policies, request logging, etc.).

```go
func NewHTTPClient(cfg HTTPConfig) *retryablehttp.Client
```

```go
client := web.NewHTTPClient(web.HTTPConfig{
    Timeout:   10 * time.Second,
    RetryMax:  5,
    RetryWait: 2 * time.Second,
})
resp, err := client.Get("https://api.example.com/data")
```

The client retries automatically on network errors and 5xx responses. Wait time between retries grows exponentially from `RetryWait` up to `RetryWait * 10`.

---

## Response helpers

Convenience functions for JSON success responses, error payloads, and 204 No Content. Covers `JSON`, `JSONStatus`, `JSONStatusE`, `Error`, and `NoContent`.

See [Response Helpers](response.md) for the full API reference and examples.

---

## HTML rendering (templ)

Thin wrappers around the templ rendering API: `Render`, `RenderWithStatus`, `RenderCtx`, and handler adapters (`TemplHandler`, `TemplHandlerFunc`, `TemplHandlerFuncE`) for wiring components directly to chi routes.

See [Templ](templ.md) for the full API reference and examples.

---

## Content negotiation

Negotiate selects the response format based on the `Accept` header. Clients sending `Accept: application/json` get JSON; all others get the templ component rendered as HTML.

```go
func WantsJSON(r *http.Request) bool
func IsHTMX(r *http.Request) bool
func Negotiate(w http.ResponseWriter, r *http.Request, jsonData any, component templ.Component) error
func NegotiateStatus(w http.ResponseWriter, r *http.Request, status int, jsonData any, component templ.Component) error
```

### WantsJSON

Returns true when the request `Accept` header contains `application/json`.

```go
if web.WantsJSON(r) {
    web.JSON(w, data)
} else {
    web.Render(w, r, views.Page(data))
}
```

### IsHTMX

Returns true when the request carries an `HX-Request: true` header (HTMX).

```go
if web.IsHTMX(r) {
    web.Render(w, r, views.Fragment(data))
} else {
    web.Render(w, r, views.FullPage(data))
}
```

### Negotiate

Picks JSON or HTML automatically using `WantsJSON`. Returns 200.

```go
func handleUsers(w http.ResponseWriter, r *http.Request) {
    users := db.ListUsers(r.Context())
    web.Negotiate(w, r, users, views.UsersPage(users))
}
```

The same handler responds correctly to both `curl` with `-H "Accept: application/json"` and a browser.

### NegotiateStatus

Same as `Negotiate` with a custom status code.

```go
func handleCreate(w http.ResponseWriter, r *http.Request) {
    user := db.CreateUser(r.Context(), req)
    web.NegotiateStatus(w, r, http.StatusCreated, user, views.UserCreated(user))
}
```

---

## Input binding and validation

### BindJSON

Decodes a JSON request body into dst.

```go
func BindJSON(r *http.Request, dst any) error
```

```go
var req CreateUserRequest
if err := web.BindJSON(r, &req); err != nil {
    web.Error(w, http.StatusBadRequest, "invalid JSON")
    return
}
```

### BindAndValidate

Decodes and validates in one call. Returns field errors separately from decode errors so you can respond with the right status code.

```go
func BindAndValidate(r *http.Request, v *validator.Validate, dst any) (map[string]string, error)
```

Return semantics:

| fields | err | Meaning |
|--------|-----|---------|
| `nil` | `nil` | Success |
| `nil` | non-nil | Malformed JSON — send 400 |
| non-nil | `nil` | Validation failed — send 422 |

```go
v := web.NewStandardValidator()
var req CreateUserRequest

fields, err := web.BindAndValidate(r, v, &req)
if err != nil {
    web.Error(w, http.StatusBadRequest, "invalid JSON")
    return
}
if fields != nil {
    web.JSONStatus(w, http.StatusUnprocessableEntity, map[string]any{"errors": fields})
    return
}

// req is valid — proceed
```

### Validator

`NewValidator`, `NewStandardValidator`, and `ValidationErrors` provide struct validation backed by go-playground/validator. The `notblank` custom tag rejects whitespace-only strings (unlike `required`).

See [Validator](validate.md) for the full API reference, tag listing, and error message table.

---

## Flash messages

One-time notifications stored in a cookie and consumed on the next request. Designed for the post-redirect-get pattern.

```go
func SetFlash(w http.ResponseWriter, level FlashLevel, message string)
func GetFlash(w http.ResponseWriter, r *http.Request) *FlashMessage
func FlashMiddleware(next http.Handler) http.Handler
func FlashFromContext(ctx context.Context) *FlashMessage
```

### FlashLevel

| Level | Constant |
|-------|----------|
| `"success"` | `web.FlashSuccess` |
| `"error"` | `web.FlashError` |
| `"warning"` | `web.FlashWarning` |
| `"info"` | `web.FlashInfo` |

### Post-redirect-get pattern

```go
func handleCreate(w http.ResponseWriter, r *http.Request) {
    // ... create item ...
    web.SetFlash(w, web.FlashSuccess, "Item created")
    http.Redirect(w, r, "/items", http.StatusSeeOther)
}
```

### FlashMiddleware

Add to your router to make flash available via `FlashFromContext`:

```go
router.Use(web.FlashMiddleware)

router.Get("/items", func(w http.ResponseWriter, r *http.Request) {
    flash := web.FlashFromContext(r.Context()) // nil if no flash
    items := db.ListItems(r.Context())
    web.Render(w, r, views.Items(items, flash))
})
```

`FlashMiddleware` reads and clears the cookie in one step — the message displays once, then disappears.

---

## CSRF protection

Uses Go 1.23's `http.NewCrossOriginProtection`, which inspects `Sec-Fetch-Site` and `Origin` headers. Non-browser clients that omit these headers pass through unchanged.

```go
func CSRFProtect() func(http.Handler) http.Handler
func CSRFProtectWithOrigins(origins ...string) (func(http.Handler) http.Handler, error)
```

```go
// Basic protection
router := web.NewRouter(web.RouterConfig{
    Middleware: append(web.StandardMiddleware, web.CSRFProtect()),
})

// With trusted cross-origin frontends
mw, err := web.CSRFProtectWithOrigins("https://app.example.com")
if err != nil {
    log.Fatal(err)
}
router := web.NewRouter(web.RouterConfig{
    Middleware: append(web.StandardMiddleware, mw),
})
```

---

## Pagination

Parses `page` and `per_page` query parameters. Never returns an error — invalid values fall back to defaults.

```go
func ParsePage(r *http.Request) Page
func ParsePageWithConfig(r *http.Request, cfg PageConfig) Page
func NewPagedResponse[T any](data []T, page Page, total int) PagedResponse[T]
```

### Page

| Field | Description |
|-------|-------------|
| `Number` | 1-based page number |
| `PerPage` | items per page |
| `Offset` | `(Number-1) * PerPage` — use as SQL OFFSET |

### PageConfig

| Field | Default | Description |
|-------|---------|-------------|
| `DefaultPerPage` | 20 | Per-page when `per_page` is missing or invalid |
| `MaxPerPage` | 100 | Hard cap on per-page |

```go
func handleUsers(w http.ResponseWriter, r *http.Request) {
    p := web.ParsePage(r)
    users, total := db.ListUsers(r.Context(), p.Offset, p.PerPage)
    web.JSON(w, web.NewPagedResponse(users, p, total))
}
```

Response shape:

```json
{
  "data": [...],
  "page": 2,
  "per_page": 25,
  "total": 143,
  "total_pages": 6
}
```

---

## Static assets

`Assets` serves embedded files with content-hash cache busting and ETag negotiation.

```go
func NewAssets(cfg AssetConfig) (*Assets, error)
func (a *Assets) Path(name string) string
func (a *Assets) Handler() http.Handler
```

### AssetConfig

| Field | Default | Description |
|-------|---------|-------------|
| `FS` | required | Embedded filesystem |
| `Prefix` | `"/assets"` | URL path prefix |
| `MaxAge` | 31536000 (1 year) | `Cache-Control max-age` seconds |

```go
//go:embed static
var staticFS embed.FS

assets, err := web.NewAssets(web.AssetConfig{FS: staticFS})
if err != nil {
    log.Fatal(err)
}

router.Handle("/assets/*", assets.Handler())
```

In templates, use `assets.Path` to get the versioned URL:

```go
// returns "/assets/app.css?v=a3f2b1"
url := assets.Path("app.css")
```

The hash changes when file content changes. Browsers cache the file for one year, then fetch a new URL when you deploy updated content.

---

## Reference

Quick lookup for all exports in the `web` package. For detailed usage, see [Response Helpers](response.md), [Templ](templ.md), and [Validator](validate.md).

### Router and server

| Function | Description |
|----------|-------------|
| `NewRouter(cfg)` | Create chi router with middleware |
| `ListenAndServe(addr, handler)` | Start server with graceful shutdown (30s) |
| `ListenAndServeWithTimeout(addr, handler, timeout)` | Start server with custom shutdown timeout |

### Response

| Function | Returns | Description |
|----------|---------|-------------|
| `JSON(w, data)` | — | 200 JSON response |
| `JSONStatus(w, status, data)` | — | JSON response with status |
| `JSONStatusE(w, status, data)` | `error` | JSON response, return error on failure |
| `Error(w, status, message)` | — | `{"error":"..."}` response |
| `NoContent(w)` | — | 204 response |

### HTML rendering

| Function | Returns | Description |
|----------|---------|-------------|
| `Render(w, r, component)` | `error` | Render component, 200 |
| `RenderWithStatus(w, r, status, component)` | `error` | Render component with status |
| `RenderCtx(ctx, w, component)` | `error` | Render component with custom context |
| `TemplHandler(component)` | `http.Handler` | Handler for static components |
| `TemplHandlerFunc(fn)` | `http.HandlerFunc` | Handler adapter for dynamic components |
| `TemplHandlerFuncE(fn)` | `http.HandlerFunc` | Handler adapter with error return |

### Negotiation

| Function | Returns | Description |
|----------|---------|-------------|
| `WantsJSON(r)` | `bool` | True if `Accept` contains `application/json` |
| `IsHTMX(r)` | `bool` | True if `HX-Request: true` |
| `Negotiate(w, r, json, component)` | `error` | JSON or HTML at 200 |
| `NegotiateStatus(w, r, status, json, component)` | `error` | JSON or HTML with status |

### Binding and validation

| Function | Returns | Description |
|----------|---------|-------------|
| `BindJSON(r, dst)` | `error` | Decode JSON body |
| `BindAndValidate(r, v, dst)` | `(map[string]string, error)` | Decode and validate |
| `NewValidator(cfg)` | `*validator.Validate` | Configured validator |
| `NewStandardValidator()` | `*validator.Validate` | Validator with defaults |
| `ValidationErrors(err)` | `map[string]string` | Extract field errors |

### Flash

| Function | Returns | Description |
|----------|---------|-------------|
| `SetFlash(w, level, message)` | — | Write flash cookie |
| `GetFlash(w, r)` | `*FlashMessage` | Read and clear flash cookie |
| `FlashMiddleware` | `http.Handler` | Inject flash into context |
| `FlashFromContext(ctx)` | `*FlashMessage` | Read flash from context |

### CSRF

| Function | Returns | Description |
|----------|---------|-------------|
| `CSRFProtect()` | middleware | CSRF middleware |
| `CSRFProtectWithOrigins(origins...)` | `(middleware, error)` | CSRF with trusted origins |

### Pagination

| Function | Returns | Description |
|----------|---------|-------------|
| `ParsePage(r)` | `Page` | Parse `page` and `per_page` params |
| `ParsePageWithConfig(r, cfg)` | `Page` | Parse with custom defaults and max |
| `NewPagedResponse[T](data, page, total)` | `PagedResponse[T]` | Build paginated JSON envelope |

### Assets

| Function | Returns | Description |
|----------|---------|-------------|
| `NewAssets(cfg)` | `(*Assets, error)` | Build asset server from embedded FS |
| `(a *Assets) Path(name)` | `string` | Versioned URL for an asset |
| `(a *Assets) Handler()` | `http.Handler` | HTTP handler for embedded files |

### HTTP Client

| Function | Returns | Description |
|----------|---------|-------------|
| `NewStandardClient()` | `*http.Client` | Retryable client with default config, standard library interface |
| `NewHTTPClient(cfg)` | `*retryablehttp.Client` | Retryable client with custom timeout, retries, and wait |
| `HTTPConfig` | struct | Config for `NewHTTPClient`: `Timeout`, `RetryMax`, `RetryWait` |

### See also

- [chi documentation](https://github.com/go-chi/chi)
- [templ documentation](https://github.com/a-h/templ)
- [go-playground/validator](https://github.com/go-playground/validator)
- [Migrations](/components/migrate) - Database schema versioning
- [PostgreSQL](/components/postgres) - Database integration
- [SQLite](/components/sqlite) - Embedded database integration
