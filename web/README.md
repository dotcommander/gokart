# Serve JSON over HTTP

```go
router := web.NewRouter(web.RouterConfig{Middleware: web.StandardMiddleware})
router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
    web.JSON(w, map[string]string{"status": "ok"})
})

if err := web.Serve(ctx, ":8080", router, web.DefaultServerConfig()); err != nil {
    return err
}
```

The module owns chi router/server construction, JSON responses, bounded request binding, and validator setup. The detailed guides are in [the web documentation](../docs/components/web.md).

## Bind safely

```go
var input CreateUser
if err := web.BindJSON(r, &input); err != nil {
    web.Error(w, http.StatusBadRequest, "invalid request")
    return
}
```

`BindJSON` limits request bodies to 10 MiB. Use `BindJSONWithLimit` for a smaller endpoint-specific bound.

## Removed in v0.11

| Removed policy surface | Direct replacement |
|---|---|
| static assets | `http.FileServer` |
| auth, sessions, CSRF, flash | caller-selected middleware |
| health orchestration | caller-owned handlers |
| retry clients | `net/http` or `go-retryablehttp` |
| content negotiation and pagination | caller-owned request/response policy |
| rate limiting | `golang.org/x/time/rate` |
| templ adapters | templ's native API |

Historical `v0.10.3` tags remain the compatibility path.
