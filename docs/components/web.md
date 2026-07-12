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

## See Also

- [Response helpers](response.md)
- [Validation](validate.md)
