package web

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const jsonContentType = "application/json"

func writeJSON(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", jsonContentType)
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		return fmt.Errorf("json encode: %w", err)
	}
	return nil
}

// JSON writes a JSON response with status 200.
func JSON(w http.ResponseWriter, data any) {
	JSONStatus(w, http.StatusOK, data)
}

// JSONStatus writes a JSON response with the given status code.
func JSONStatus(w http.ResponseWriter, status int, data any) {
	_ = writeJSON(w, status, data)
}

// JSONStatusE writes a JSON response with the given status code.
// Returns an error if JSON encoding fails.
func JSONStatusE(w http.ResponseWriter, status int, data any) error {
	return writeJSON(w, status, data)
}

// Error writes a JSON error response with the given status code.
func Error(w http.ResponseWriter, status int, message string) {
	JSONStatus(w, status, map[string]string{"error": message})
}

// NoContent writes a 204 No Content response.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
