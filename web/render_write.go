package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/a-h/templ"
)

const (
	htmlContentType = "text/html; charset=utf-8"
	jsonContentType = "application/json"
)

func setContentType(w http.ResponseWriter, contentType string) {
	w.Header().Set("Content-Type", contentType)
}

func renderComponent(ctx context.Context, w http.ResponseWriter, status int, component templ.Component) error {
	setContentType(w, htmlContentType)
	if status > 0 {
		w.WriteHeader(status)
	}
	return component.Render(ctx, w)
}

func writeJSON(w http.ResponseWriter, status int, data any) error {
	setContentType(w, jsonContentType)
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		return fmt.Errorf("json encode: %w", err)
	}
	return nil
}
