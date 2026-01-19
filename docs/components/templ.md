# Templ

Type-safe HTML rendering helpers for [templ](https://github.com/a-h/templ) components. Provides thin wrappers around templ's rendering API with sensible defaults for HTTP handlers.

## Installation

```bash
go get github.com/dotcommander/gokart
go get github.com/a-h/templ
```

## Quick Start

```go
import "github.com/dotcommander/gokart"

func handleHome(w http.ResponseWriter, r *http.Request) {
    // Simple render with 200 status
    gokart.Render(w, r, views.HomePage("Welcome"))

    // Render with custom status
    gokart.RenderWithStatus(w, r, http.StatusNotFound, views.NotFoundPage())

    // Handler adapter for static pages
    router.Get("/about", gokart.TemplHandler(views.AboutPage()))

    // Handler adapter for pages needing request data
    router.Get("/user/{id}", gokart.TemplHandlerFunc(func(r *http.Request) templ.Component {
        user := getUser(chi.URLParam(r, "id"))
        return views.UserPage(user)
    }))
}
```

---

## Render Functions

### Render

Renders a templ component to an http.ResponseWriter with status 200 OK.

```go
func Render(w http.ResponseWriter, r *http.Request, component templ.Component) error
```

```go
func handleHome(w http.ResponseWriter, r *http.Request) {
    if err := gokart.Render(w, r, views.HomePage("Welcome")); err != nil {
        log.Printf("render error: %v", err)
    }
}
```

**Parameters:**
- `w` - `http.ResponseWriter` to write to
- `r` - `*http.Request` providing the context
- `component` - `templ.Component` to render

**Headers set:**
- `Content-Type: text/html; charset=utf-8`

**Returns:** Error if rendering fails

**Use when:** Standard page render with 200 OK status

---

### RenderWithStatus

Renders a templ component with a custom HTTP status code.

```go
func RenderWithStatus(w http.ResponseWriter, r *http.Request, status int, component templ.Component) error
```

```go
func handleNotFound(w http.ResponseWriter, r *http.Request) {
    gokart.RenderWithStatus(w, r, http.StatusNotFound, views.NotFoundPage())
}

func handleUnauthorized(w http.ResponseWriter, r *http.Request) {
    gokart.RenderWithStatus(w, r, http.StatusUnauthorized, views.UnauthorizedPage())
}
```

**Parameters:**
- `w` - `http.ResponseWriter` to write to
- `r` - `*http.Request` providing the context
- `status` - HTTP status code to send
- `component` - `templ.Component` to render

**Headers set:**
- `Content-Type: text/html; charset=utf-8`

**Returns:** Error if rendering fails

**Common status codes:**
- `200 OK` - Successful response (use `Render` instead)
- `201 Created` - Resource created
- `400 Bad Request` - Invalid input
- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Insufficient permissions
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Server error

---

### RenderCtx

Renders a templ component with a custom context.

```go
func RenderCtx(ctx context.Context, w http.ResponseWriter, component templ.Component) error
```

```go
func handleDashboard(w http.ResponseWriter, r *http.Request) {
    user := middleware.GetUser(r.Context())

    // Add user to context for component access
    ctx := context.WithValue(r.Context(), "user", user)

    if err := gokart.RenderCtx(ctx, w, views.Dashboard()); err != nil {
        log.Printf("render error: %v", err)
    }
}
```

**Parameters:**
- `ctx` - `context.Context` to pass to the component
- `w` - `http.ResponseWriter` to write to
- `component` - `templ.Component` to render

**Headers set:**
- `Content-Type: text/html; charset=utf-8`

**Returns:** Error if rendering fails

**Use when:** Component needs access to custom context values (user, request ID, tracing, etc.)

---

## Handler Adapters

### TemplHandler

Creates an `http.Handler` from a static templ component.

```go
func TemplHandler(component templ.Component) http.Handler
```

```go
func setupRoutes(r chi.Router) {
    // Static page - no request data needed
    r.Get("/about", gokart.TemplHandler(views.AboutPage()))
    r.Get("/contact", gokart.TemplHandler(views.ContactPage()))
    r.Get("/privacy", gokart.TemplHandler(views.PrivacyPage()))
}
```

**Parameters:**
- `component` - `templ.Component` to render

**Returns:** `http.Handler` that renders the component

**Use when:** Page doesn't need data from the request (static content, simple pages)

---

### TemplHandlerFunc

Creates an `http.HandlerFunc` from a function that returns a component based on the request.

```go
func TemplHandlerFunc(fn func(r *http.Request) templ.Component) http.HandlerFunc
```

```go
func setupRoutes(r chi.Router) {
    // Page needs data from request
    r.Get("/user/{id}", gokart.TemplHandlerFunc(func(r *http.Request) templ.Component {
        id := chi.URLParam(r, "id")
        user := getUser(id)
        return views.UserPage(user)
    }))

    r.Get("/search", gokart.TemplHandlerFunc(func(r *http.Request) templ.Component {
        query := r.URL.Query().Get("q")
        results := search(query)
        return views.SearchResults(query, results)
    }))
}
```

**Parameters:**
- `fn` - Function receiving `*http.Request` and returning `templ.Component`

**Returns:** `http.HandlerFunc`

**Use when:** Component needs data extracted from the request (URL params, query string, headers)

**Error handling:** Returns 500 Internal Server Error if rendering fails

---

### TemplHandlerFuncE

Creates an `http.HandlerFunc` from a function that can return an error.

```go
func TemplHandlerFuncE(fn func(r *http.Request) (templ.Component, error)) http.HandlerFunc
```

```go
func setupRoutes(r chi.Router) {
    // Page needs data fetching with error handling
    r.Get("/dashboard", gokart.TemplHandlerFuncE(func(r *http.Request) (templ.Component, error) {
        data, err := loadDashboardData(r.Context())
        if err != nil {
            return nil, fmt.Errorf("load dashboard: %w", err)
        }
        return views.Dashboard(data), nil
    }))

    r.Get("/post/{id}", gokart.TemplHandlerFuncE(func(r *http.Request) (templ.Component, error) {
        id := chi.URLParam(r, "id")
        post, err := db.GetPost(r.Context(), id)
        if err != nil {
            if errors.Is(err, pgx.ErrNoRows) {
                return nil, fmt.Errorf("post not found: %w", err)
            }
            return nil, fmt.Errorf("get post: %w", err)
        }
        return views.PostPage(post), nil
    }))
}
```

**Parameters:**
- `fn` - Function receiving `*http.Request` and returning `(templ.Component, error)`

**Returns:** `http.HandlerFunc`

**Use when:** Component requires data fetching that can fail (database queries, API calls)

**Error handling:**
- Returns 500 Internal Server Error if function returns error
- Returns 500 Internal Server Error if rendering fails

---

## Example Handlers

### Basic Page Handler

```go
func handleHome(w http.ResponseWriter, r *http.Request) {
    title := "Welcome to My App"
    if err := gokart.Render(w, r, views.HomePage(title)); err != nil {
        log.Printf("render error: %v", err)
    }
}
```

### Data-Driven Page Handler

```go
func handleUserProfile(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")

    user, err := db.GetUser(r.Context(), id)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            gokart.RenderWithStatus(w, r, http.StatusNotFound, views.NotFoundPage())
            return
        }
        gokart.RenderWithStatus(w, r, http.StatusInternalServerError, views.ErrorPage())
        return
    }

    gokart.Render(w, r, views.UserProfilePage(user))
}
```

### Handler with Context Data

```go
func handleDashboard(w http.ResponseWriter, r *http.Request) {
    user := middleware.GetUser(r.Context())

    data, err := loadDashboardData(r.Context(), user.ID)
    if err != nil {
        log.Printf("load dashboard: %v", err)
        gokart.RenderWithStatus(w, r, http.StatusInternalServerError, views.ErrorPage())
        return
    }

    // Pass user context to component
    ctx := context.WithValue(r.Context(), "user", user)
    if err := gokart.RenderCtx(ctx, w, views.Dashboard(data)); err != nil {
        log.Printf("render error: %v", err)
    }
}
```

### Static Routes with Handler Adapter

```go
func setupRoutes(r chi.Router) {
    // Public pages
    r.Get("/", gokart.TemplHandler(views.HomePage()))
    r.Get("/about", gokart.TemplHandler(views.AboutPage()))
    r.Get("/contact", gokart.TemplHandler(views.ContactPage()))

    // Protected pages
    r.Group(func(r chi.Router) {
        r.Use(middleware.RequireAuth)
        r.Get("/dashboard", gokart.TemplHandlerFunc(dashboardHandler))
        r.Get("/settings", gokart.TemplHandlerFunc(settingsHandler))
    })
}

func dashboardHandler(r *http.Request) templ.Component {
    user := middleware.GetUser(r.Context())
    data := getDashboardData(user.ID)
    return views.Dashboard(data)
}

func settingsHandler(r *http.Request) templ.Component {
    user := middleware.GetUser(r.Context())
    settings := getSettings(user.ID)
    return views.SettingsPage(settings)
}
```

### Handler with Error Handling

```go
func handlePost(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")

    post, err := db.GetPost(r.Context(), id)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            gokart.RenderWithStatus(w, r, http.StatusNotFound, views.NotFoundPage())
        } else {
            log.Printf("get post: %v", err)
            gokart.RenderWithStatus(w, r, http.StatusInternalServerError, views.ErrorPage())
        }
        return
    }

    comments, err := db.GetComments(r.Context(), id)
    if err != nil {
        log.Printf("get comments: %v", err)
        // Show post without comments rather than error page
    }

    gokart.Render(w, r, views.PostPage(post, comments))
}
```

---

## Error Handling Patterns

### Graceful Degradation

```go
func handlePage(w http.ResponseWriter, r *http.Request) {
    primary, err := loadPrimaryData(r.Context())
    if err != nil {
        log.Printf("load primary: %v", err)
        gokart.RenderWithStatus(w, r, http.StatusInternalServerError, views.ErrorPage())
        return
    }

    secondary, err := loadSecondaryData(r.Context())
    if err != nil {
        log.Printf("load secondary: %v", err)
        // Render with empty secondary rather than error page
        gokart.Render(w, r, views.Page(primary, nil))
        return
    }

    gokart.Render(w, r, views.Page(primary, secondary))
}
```

### Error Page with Status

```go
func handleNotFound(w http.ResponseWriter, r *http.Request) {
    gokart.RenderWithStatus(w, r, http.StatusNotFound, views.NotFoundPage())
}

func handleUnauthorized(w http.ResponseWriter, r *http.Request) {
    gokart.RenderWithStatus(w, r, http.StatusUnauthorized, views.UnauthorizedPage())
}

func handleServerError(w http.ResponseWriter, r *http.Request) {
    gokart.RenderWithStatus(w, r, http.StatusInternalServerError, views.ServerErrorPage())
}
```

### Logging Errors

```go
func handlePage(w http.ResponseWriter, r *http.Request) {
    if err := gokart.Render(w, r, views.Page()); err != nil {
        log.Printf("render error: %v", err)
        // Render already failed, can't send another response
    }
}
```

---

## Reference

### Render Functions

| Function | Returns | Description |
|----------|---------|-------------|
| `Render` | `error` | Render component with 200 status |
| `RenderWithStatus` | `error` | Render component with custom status |
| `RenderCtx` | `error` | Render component with custom context |

### Handler Adapters

| Function | Returns | Description |
|----------|---------|-------------|
| `TemplHandler` | `http.Handler` | Handler for static components |
| `TemplHandlerFunc` | `http.HandlerFunc` | Handler adapter for request-based components |
| `TemplHandlerFuncE` | `http.HandlerFunc` | Handler adapter with error handling |

### See Also

- [Response helpers](/components/response) - JSON response helpers
- [HTTP router](/api/gokart#http-router) - Request routing
- [templ documentation](https://github.com/a-h/templ)
