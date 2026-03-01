package web

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
)

// BindJSON decodes a JSON request body into dst.
func BindJSON(r *http.Request, dst any) error {
	return json.NewDecoder(r.Body).Decode(dst)
}

// BindAndValidate decodes a JSON request body and validates it.
//
// Returns:
//   - (nil, decodeErr) — malformed JSON (caller sends 400)
//   - (fieldErrors, nil) — validation failed (caller sends 422)
//   - (nil, nil) — success
//
// Example:
//
//	v := web.NewStandardValidator()
//	var req CreateUserRequest
//	if fields, err := web.BindAndValidate(r, v, &req); err != nil {
//	    web.Error(w, http.StatusBadRequest, "invalid JSON")
//	    return
//	} else if fields != nil {
//	    web.JSONStatus(w, http.StatusUnprocessableEntity, map[string]any{"errors": fields})
//	    return
//	}
func BindAndValidate(r *http.Request, v *validator.Validate, dst any) (map[string]string, error) {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return nil, err
	}

	if err := v.Struct(dst); err != nil {
		if fields := ValidationErrors(err); fields != nil {
			return fields, nil
		}
		return nil, err
	}

	return nil, nil
}
