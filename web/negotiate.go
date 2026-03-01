package web

import (
	"net/http"
	"strings"

	"github.com/a-h/templ"
)

// WantsJSON returns true if the request prefers a JSON response.
func WantsJSON(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "application/json")
}

// IsHTMX returns true if the request originated from HTMX.
func IsHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// Negotiate writes a JSON or HTML response based on the Accept header.
func Negotiate(w http.ResponseWriter, r *http.Request, jsonData any, component templ.Component) error {
	return NegotiateStatus(w, r, http.StatusOK, jsonData, component)
}

// NegotiateStatus writes a JSON or HTML response with the given status code.
func NegotiateStatus(w http.ResponseWriter, r *http.Request, status int, jsonData any, component templ.Component) error {
	if WantsJSON(r) {
		JSONStatus(w, status, jsonData)
		return nil
	}
	return RenderWithStatus(w, r, status, component)
}
