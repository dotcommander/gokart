# Response

JSON response helpers for writing consistent HTTP API responses. Provides functions for success responses, error responses, and common status codes.

## Installation

```bash
go get github.com/dotcommander/gokart
```

## Quick Start

```go
import "github.com/dotcommander/gokart"

func handleUser(w http.ResponseWriter, r *http.Request) {
    user := User{ID: 1, Name: "Alice"}

    // Success response (200)
    gokart.JSON(w, user)

    // Error response (400)
    gokart.Error(w, http.StatusBadRequest, "Invalid user ID")

    // Custom status (201)
    gokart.JSONStatus(w, http.StatusCreated, user)
}
```

---

## Response Functions

### JSON

Writes a JSON response with status 200 OK.

```go
func JSON(w http.ResponseWriter, data any)
```

```go
type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

func handleUser(w http.ResponseWriter, r *http.Request) {
    user := User{ID: 1, Name: "Alice"}
    gokart.JSON(w, user)
    // Response: 200 {"id":1,"name":"Alice"}
}
```

**Parameters:**
- `w` - `http.ResponseWriter` to write to
- `data` - Any value that can be JSON-encoded

---

### JSONStatus

Writes a JSON response with a custom status code.

```go
func JSONStatus(w http.ResponseWriter, status int, data any)
```

```go
func handleCreate(w http.ResponseWriter, r *http.Request) {
    user := User{ID: 1, Name: "Alice"}
    gokart.JSONStatus(w, http.StatusCreated, user)
    // Response: 201 {"id":1,"name":"Alice"}
}
```

**Parameters:**
- `w` - `http.ResponseWriter` to write to
- `status` - HTTP status code
- `data` - Any value that can be JSON-encoded

**Common uses:**
- `201 Created` - Resource created successfully
- `202 Accepted` - Request accepted for processing
- `206 Partial Content` - Partial response for range requests

---

### JSONStatusE

Writes a JSON response with a custom status code and returns an error if encoding fails.

```go
func JSONStatusE(w http.ResponseWriter, status int, data any) error
```

```go
func handleWithError(w http.ResponseWriter, r *http.Request) error {
    user := User{ID: 1, Name: "Alice"}
    if err := gokart.JSONStatusE(w, http.StatusOK, user); err != nil {
        return fmt.Errorf("failed to write response: %w", err)
    }
    return nil
}
```

**Parameters:**
- `w` - `http.ResponseWriter` to write to
- `status` - HTTP status code
- `data` - Any value that can be JSON-encoded

**Returns:** Error if JSON encoding fails

**Use when:** You need to handle encoding errors (e.g., logging, middleware error handling)

---

### Error

Writes a JSON error response with a message.

```go
func Error(w http.ResponseWriter, status int, message string)
```

```go
func handleDelete(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")

    if id == "" {
        gokart.Error(w, http.StatusBadRequest, "Missing user ID")
        // Response: 400 {"error":"Missing user ID"}
        return
    }

    // ... delete logic
}
```

**Parameters:**
- `w` - `http.ResponseWriter` to write to
- `status` - HTTP error status code
- `message` - Error message describing the problem

**Response format:**
```json
{
  "error": "Error message here"
}
```

**Common status codes:**
- `400 Bad Request` - Invalid input
- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Insufficient permissions
- `404 Not Found` - Resource not found
- `409 Conflict` - Conflicting state
- `422 Unprocessable Entity` - Validation errors
- `500 Internal Server Error` - Server error

---

### NoContent

Writes a 204 No Content response with an empty body.

```go
func NoContent(w http.ResponseWriter)
```

```go
func handleDelete(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")

    if err := deleteResource(id); err != nil {
        gokart.Error(w, http.StatusInternalServerError, "Failed to delete")
        return
    }

    gokart.NoContent(w)
    // Response: 204 (no body)
}
```

**Use when:**
- Resource deleted successfully
- Request accepted but no response needed
- Update with no return value

---

## Example Handlers

### REST API Handler

```go
func handleUser(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")

    user, err := db.GetUser(r.Context(), id)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            gokart.Error(w, http.StatusNotFound, "User not found")
        } else {
            gokart.Error(w, http.StatusInternalServerError, "Failed to fetch user")
        }
        return
    }

    gokart.JSON(w, user)
}
```

### Create Handler

```go
func handleCreate(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        gokart.Error(w, http.StatusBadRequest, "Invalid request body")
        return
    }

    if err := validator.Struct(&req); err != nil {
        gokart.Error(w, http.StatusUnprocessableEntity, "Validation failed")
        return
    }

    user, err := db.CreateUser(r.Context(), req)
    if err != nil {
        gokart.Error(w, http.StatusInternalServerError, "Failed to create user")
        return
    }

    gokart.JSONStatus(w, http.StatusCreated, user)
}
```

### Delete Handler

```go
func handleDelete(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")

    if err := db.DeleteUser(r.Context(), id); err != nil {
        gokart.Error(w, http.StatusInternalServerError, "Failed to delete user")
        return
    }

    gokart.NoContent(w)
}
```

### Error Handling Middleware

```go
func errorHandler(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                gokart.Error(w, http.StatusInternalServerError, "Internal server error")
            }
        }()
        next.ServeHTTP(w, r)
    })
}
```

---

## Response Patterns

### Success with Data

```go
gokart.JSON(w, user)
// 200 {"id":1,"name":"Alice"}
```

### Success No Data

```go
gokart.NoContent(w)
// 204 (no body)
```

### Client Error

```go
gokart.Error(w, http.StatusBadRequest, "Invalid input")
// 400 {"error":"Invalid input"}
```

### Server Error

```go
gokart.Error(w, http.StatusInternalServerError, "Database error")
// 500 {"error":"Database error"}
```

---

## Reference

### Functions

| Function | Returns | Description |
|----------|---------|-------------|
| `JSON` | - | Write 200 JSON response |
| `JSONStatus` | - | Write JSON response with custom status |
| `JSONStatusE` | `error` | Write JSON response, return error on failure |
| `Error` | - | Write JSON error response |
| `NoContent` | - | Write 204 No Content response |

### See Also

- [Validator](/components/validate) - Request validation
- [Templ](/components/templ) - HTML rendering
- [HTTP router](/api/gokart#http-router) - Request routing
