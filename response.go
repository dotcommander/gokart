// Package gokart provides thin wrappers around battle-tested packages.
// This file provides HTTP response helper functions.
package gokart

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// JSON writes a JSON response with status 200.
func JSON(w http.ResponseWriter, data any) {
	JSONStatus(w, http.StatusOK, data)
}

// JSONStatus writes a JSON response with the given status code.
func JSONStatus(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// JSONStatusE writes a JSON response with the given status code.
// Returns an error if JSON encoding fails.
func JSONStatusE(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		return fmt.Errorf("json encode: %w", err)
	}
	return nil
}

// Error writes a JSON error response with the given status code.
func Error(w http.ResponseWriter, status int, message string) {
	JSONStatus(w, status, map[string]string{"error": message})
}

// NoContent writes a 204 No Content response.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
