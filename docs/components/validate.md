# Validator

Struct validation with clear error messages and common validation rules.
Built on [go-playground/validator/v10](https://github.com/go-playground/validator).

## Installation

```bash
go get github.com/dotcommander/gokart
```

## Quick Start

```go
import "github.com/dotcommander/gokart"

// Create validator with default settings
v := gokart.NewValidator(gokart.ValidatorConfig{})

type User struct {
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"gte=0,lte=130"`
}

user := User{Email: "invalid", Age: 150}
if err := v.Struct(user); err != nil {
    for field, msg := range gokart.ValidationErrors(err) {
        fmt.Printf("%s %s\n", field, msg)
    }
    // Output:
    // email must be a valid email
    // age must be less than or equal to 130
}
```

---

## Creating a Validator

### NewValidator

Creates a configured validator instance.

```go
v := gokart.NewValidator(gokart.ValidatorConfig{
    UseJSONNames: true,  // Use json tag names in errors (default)
})
```

**Signature:**

```go
func NewValidator(cfg ValidatorConfig) *validator.Validate
```

**Parameters:**

| Parameter | Type              | Default                 | Description            |
|-----------|-------------------|-------------------------|------------------------|
| `cfg`     | `ValidatorConfig` | `{UseJSONNames: true}`  | Validator configuration |

### NewStandardValidator

Creates a validator with default settings. Convenience wrapper for
`NewValidator(ValidatorConfig{})`.

```go
v := gokart.NewStandardValidator()
```

**Signature:**

```go
func NewStandardValidator() *validator.Validate
```

---

## Configuration

### ValidatorConfig Struct

| Field          | Type  | Default | Description                                             |
|----------------|-------|---------|---------------------------------------------------------|
| `UseJSONNames` | `bool` | `true`  | Use `json` tag names in errors instead of struct fields |

**Default Behavior:**

When `UseJSONNames` is `true` (default), error messages use JSON field names:

```go
type User struct {
    Email string `json:"email" validate:"required,email"`
    //      ^^^^^ struct field name
    //             ^^^^^ json tag name (used in errors)
}

err := v.Struct(user)
// errors["email"] = "is required"
//               ^^^^^ uses json tag
```

When `UseJSONNames` is `false`, struct field names are used:

```go
v := gokart.NewValidator(gokart.ValidatorConfig{UseJSONNames: false})
// errors["Email"] = "is required"
//               ^^^^^ uses struct field
```

---

## Validation Tags

### Common Tags

| Tag | Description | Example |
|-----|-------------|---------|
| `required` | Field must be non-zero | `validate:"required"` |
| `notblank` | Field must be non-zero and not whitespace-only | `validate:"notblank"` |
| `email` | Valid email format | `validate:"email"` |
| `url` | Valid URL format | `validate:"url"` |
| `uuid` | Valid UUID format | `validate:"uuid"` |
| `min` | Minimum length (strings/arrays) or value (numbers) | `validate:"min=3"` |
| `max` | Maximum length (strings/arrays) or value (numbers) | `validate:"max=100"` |
| `gte` | Greater than or equal to | `validate:"gte=0"` |
| `lte` | Less than or equal to | `validate:"lte=130"` |
| `oneof` | Value must be one of the listed options | `validate:"oneof=red green blue"` |
| `eq` | Equal to | `validate:"eq=5"` |
| `ne` | Not equal to | `validate:"ne=0"` |
| `alpha` | Contains only letters | `validate:"alpha"` |
| `alphanum` | Contains only alphanumeric characters | `validate:"alphanum"` |
| `numeric` | Valid numeric string | `validate:"numeric"` |

### Examples

```go
type Registration struct {
    Username string `json:"username" validate:"required,min=3,max=20,alphanum"`
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
    Age      int    `json:"age" validate:"gte=18,lte=120"`
    Role     string `json:"role" validate:"oneof=user admin moderator"`
    Website  string `json:"website" validate:"omitempty,url"`
}
```

### Custom Validator: `notblank`

GoKart registers a custom `notblank` validator that rejects whitespace-only
strings:

```go
type Comment struct {
    Content string `json:"content" validate:"notblank"`
}

// These fail:
Comment{Content: ""}        // empty
Comment{Content: "   "}     // whitespace only

// These pass:
Comment{Content: "hello"}   // non-whitespace
```

**Difference from `required`:**

| Validator | Empty String | Whitespace Only | Non-Empty |
|-----------|--------------|-----------------|------------|
| `required` | ❌ Fails     | ✅ Passes       | ✅ Passes  |
| `notblank` | ❌ Fails     | ❌ Fails        | ✅ Passes  |

### Combining Tags

Chain multiple validators with commas:

```go
type Product struct {
    SKU      string  `json:"sku" validate:"required,uppercase,min=3,max=10"`
    Price    float64 `json:"price" validate:"required,gte=0.01"`
    Stock    int     `json:"stock" validate:"gte=0"`
    IsActive bool    `json:"is_active" validate:"required"`
}
```

---

## Validating

### Validating a Struct

```go
v := gokart.NewValidator(gokart.ValidatorConfig{})

user := User{Email: "not-an-email", Age: -5}
if err := v.Struct(user); err != nil {
    // Handle validation errors
}
```

### Conditional Validation with `omitempty`

Use `omitempty` to skip validation when field is empty:

```go
type Profile struct {
    Bio     string `json:"bio" validate:"omitempty,max=500"`
    Website string `json:"website" validate:"omitempty,url"`
}

// These pass (empty fields skip validation):
Profile{}
Profile{Bio: "", Website: ""}

// These pass (non-empty fields validate):
Profile{Bio: "Short bio", Website: "https://example.com"}
```

---

## Error Handling

### ValidationErrors

Extracts field-level errors as a `map[string]string`:

```go
if err := v.Struct(user); err != nil {
    errors := gokart.ValidationErrors(err)
    for field, message := range errors {
        fmt.Printf("%s: %s\n", field, message)
    }
}
```

**Signature:**

```go
func ValidationErrors(err error) map[string]string
```

**Return value:**

- `map[string]string` on validation errors
- `nil` if `err` is not a `validator.ValidationErrors`

### Error Messages

Default messages for common validators:

| Tag       | Message                                            |
|-----------|----------------------------------------------------|
| `required` | "is required"                                      |
| `notblank` | "cannot be blank"                                  |
| `email`    | "must be a valid email"                            |
| `url`      | "must be a valid URL"                              |
| `uuid`     | "must be a valid UUID"                             |
| `min`      | "is too short"                                     |
| `max`      | "is too long"                                      |
| `gte`      | "must be greater than or equal to {param}"         |
| `lte`      | "must be less than or equal to {param}"            |
| `oneof`    | "must be one of: {param}"                          |
| other      | "failed {tag} validation"                          |

### Custom Error Handling

For more control, type-assert to `validator.ValidationErrors`:

```go
import "github.com/go-playground/validator/v10"

if err := v.Struct(user); err != nil {
    if validationErrors, ok := err.(validator.ValidationErrors); ok {
        for _, fe := range validationErrors {
            fmt.Printf("Field: %s\n", fe.Field())
            fmt.Printf("Tag: %s\n", fe.Tag())
            fmt.Printf("Param: %s\n", fe.Param())
            fmt.Printf("Value: %v\n", fe.Value())
        }
    }
}
```

---

## HTTP Integration

### API Error Responses

Combine with HTTP handlers for validation errors:

```go
func handleCreateUser(w http.ResponseWriter, r *http.Request) {
    var user User
    if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
        http.Error(w, "invalid JSON", http.StatusBadRequest)
        return
    }

    if err := v.Struct(user); err != nil {
        errors := gokart.ValidationErrors(err)
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusUnprocessableEntity)
        json.NewEncoder(w).Encode(map[string]any{
            "error": "validation failed",
            "fields": errors,
        })
        return
    }

    // Process valid user...
}
```

**Response example:**

```json
{
  "error": "validation failed",
  "fields": {
    "email": "must be a valid email",
    "age": "must be greater than or equal to 0"
  }
}
```

---

## Best Practices

### Use JSON Tags for API Responses

Always set `UseJSONNames: true` for HTTP APIs:

```go
v := gokart.NewValidator(gokart.ValidatorConfig{UseJSONNames: true})

type Request struct {
    FirstName string `json:"first_name" validate:"required"`
    // Response uses "first_name", not "FirstName"
}
```

### Prefer `notblank` for User Input

Use `notblank` instead of `required` for text fields:

```go
// Good - rejects whitespace-only input
type Comment struct {
    Body string `validate:"notblank,max=5000"`
}

// Avoid - accepts "   " as valid
type Comment struct {
    Body string `validate:"required,max=5000"`
}
```

### Validate Early

Validate as soon as possible in request handlers:

```go
func handleCreate(w http.ResponseWriter, r *http.Request) {
    var req CreateRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid JSON", http.StatusBadRequest)
        return
    }

    // Validate immediately after decoding
    if err := v.Struct(req); err != nil {
        // Handle validation errors
        return
    }

    // Continue with business logic...
}
```

### Reuse Validator Instances

Create validator once, reuse everywhere:

```go
var v = gokart.NewStandardValidator()

func main() {
    // Use v throughout application
}

func handleRequest() {
    if err := v.Struct(request); err != nil {
        // ...
    }
}
```

---

## Reference

### Functions

| Function                | Returns             | Description               |
|-------------------------|---------------------|---------------------------|
| `NewValidator`          | `*validator.Validate` | Creates configured validator |
| `NewStandardValidator`  | `*validator.Validate` | Creates validator with defaults |
| `ValidationErrors`      | `map[string]string` | Extracts field errors     |

### Types

| Type              | Description                      |
|-------------------|----------------------------------|
| `ValidatorConfig` | Validator configuration options |

### See Also

- [go-playground/validator documentation](https://github.com/go-playground/validator#usage)
- [Validation tags reference](https://github.com/go-playground/validator/blob/main/README.md#tags)
- [Response helpers](/components/response) - HTTP error responses
