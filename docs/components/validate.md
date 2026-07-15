# Validate a request

```go
type CreateUser struct {
    Email string `json:"email" validate:"required,email"`
    Name  string `json:"name" validate:"notblank"`
}

v := web.NewStandardValidator()
var input CreateUser
fields, err := web.BindAndValidate(r, v, &input)
if err != nil {
    var tooLarge *http.MaxBytesError
    if errors.As(err, &tooLarge) {
        web.Error(w, http.StatusRequestEntityTooLarge, "request too large")
    } else {
        web.Error(w, http.StatusBadRequest, "invalid JSON")
    }
    return
}
if len(fields) > 0 {
    web.JSONStatus(w, http.StatusUnprocessableEntity, map[string]any{"errors": fields})
    return
}
```

Validation uses `go-playground/validator/v10` and returns short field messages suitable for JSON APIs.

## Install

```bash
go get github.com/dotcommander/gokart/web@v0.13.0
```

## Create a validator

```go
v := web.NewValidator(web.ValidatorConfig{})
```

The zero configuration and `NewStandardValidator` use JSON tag names. `ValidatorConfig{UseJSONNames:false}` is also the zero value, so it cannot explicitly select struct field names; use the upstream validator directly when that distinction is required.

GoKart registers `notblank`, which rejects empty and whitespace-only strings and zero values of other supported kinds.

## Format errors

```go
if err := v.Struct(input); err != nil {
    for field, message := range web.ValidationErrors(err) {
        fmt.Printf("%s %s\n", field, message)
    }
}
```

`ValidationErrors` returns `nil` for errors that are not `validator.ValidationErrors`. It supplies dedicated messages for `required`, `notblank`, `email`, `url`, `uuid`, `min`, `max`, `gte`, `lte`, and `oneof`; other tags use `failed <tag> validation`.

## Bind and validate

`BindAndValidate` first calls the bounded JSON binder, then validates the destination. A syntax or size error is returned as `err`; field failures are returned in the map.

## See also

- [Web](web.md)
- [Responses](response.md)
