package web

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
)

// DefaultMaxRequestBodyBytes caps JSON request bodies decoded by BindJSON and
// BindAndValidate to keep a single request from exhausting server memory.
// 10 MiB is generous for typical JSON payloads while still bounded; callers
// needing a different cap should use BindJSONWithLimit.
const DefaultMaxRequestBodyBytes int64 = 10 * 1024 * 1024

// BindJSON decodes a JSON request body into dst, applying
// DefaultMaxRequestBodyBytes to bound r.Body. When the cap is exceeded the
// returned error wraps *http.MaxBytesError, which callers can detect with
// errors.As to send 413 instead of 400.
func BindJSON(r *http.Request, dst any) error {
	return BindJSONWithLimit(r, dst, DefaultMaxRequestBodyBytes)
}

// BindJSONWithLimit is BindJSON with an explicit byte cap. Use this for
// endpoints whose payload shape demands a tighter or looser limit than the
// package default. A non-positive limit disables the cap (caller takes
// responsibility for sizing).
func BindJSONWithLimit(r *http.Request, dst any, limit int64) error {
	if limit > 0 {
		// http.MaxBytesReader tolerates a nil ResponseWriter — it just
		// skips the auto-413 wiring — so BindJSON callers that don't have
		// a writer in scope (rare) still get the body cap.
		r.Body = http.MaxBytesReader(nil, r.Body, limit)
	}
	return json.NewDecoder(r.Body).Decode(dst)
}

// BindAndValidate decodes a JSON request body and validates it.
//
// Returns:
//   - (nil, decodeErr) — malformed JSON or oversized body (caller sends 400/413)
//   - (fieldErrors, nil) — validation failed (caller sends 422)
//   - (nil, nil) — success
//
// The body is bounded by DefaultMaxRequestBodyBytes. Detect the oversized
// case with errors.As(err, new(*http.MaxBytesError)) to differentiate 413
// from 400.
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
	if err := BindJSON(r, dst); err != nil {
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
