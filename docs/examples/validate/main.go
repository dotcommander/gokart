//go:build ignore

// Example: Bounded JSON binding and validation with gokart/web.
//
// BindAndValidate limits the body (10 MiB default) and separates transport
// errors (err) from field failures (the returned map). No port is opened:
// httptest drives the handler, the pattern used in handler tests.
//
// Run with: go run docs/examples/validate/main.go
package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/dotcommander/gokart/web"
	"github.com/go-playground/validator/v10"
)

type CreateUser struct {
	Email string `json:"email" validate:"required,email"`
	Name  string `json:"name" validate:"notblank"`
}

func handler(v *validator.Validate) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		web.JSONStatus(w, http.StatusCreated, input)
	}
}

func main() {
	h := handler(web.NewStandardValidator())

	for _, body := range []string{
		`{"email":"ada@example.com","name":"Ada"}`, // 201
		`{"email":"not-an-email","name":"  "}`,     // 422 field errors
		`{"email":`,                                // 400 syntax error
	} {
		req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		fmt.Printf("%s -> %d %s\n", body, rec.Code, strings.TrimSpace(rec.Body.String()))
	}
}
