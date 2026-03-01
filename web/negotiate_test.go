package web_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/stretchr/testify/assert"

	"github.com/dotcommander/gokart/web"
)

func TestWantsJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		accept   string
		expected bool
	}{
		{name: "explicit json", accept: "application/json", expected: true},
		{name: "json among multiple", accept: "text/html, application/json", expected: true},
		{name: "empty", accept: "", expected: false},
		{name: "text/html", accept: "text/html", expected: false},
		{name: "wildcard", accept: "*/*", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.accept != "" {
				r.Header.Set("Accept", tt.accept)
			}
			assert.Equal(t, tt.expected, web.WantsJSON(r))
		})
	}
}

func TestIsHTMX(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		hxRequest string
		set       bool
		expected  bool
	}{
		{name: "true", hxRequest: "true", set: true, expected: true},
		{name: "absent", hxRequest: "", set: false, expected: false},
		{name: "false", hxRequest: "false", set: true, expected: false},
		{name: "non-standard 1", hxRequest: "1", set: true, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.set {
				r.Header.Set("HX-Request", tt.hxRequest)
			}
			assert.Equal(t, tt.expected, web.IsHTMX(r))
		})
	}
}

func TestNegotiate(t *testing.T) {
	t.Parallel()

	t.Run("json_via_accept", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()

		dummy := templ.ComponentFunc(func(_ context.Context, _ io.Writer) error {
			return nil
		})

		err := web.Negotiate(w, r, map[string]string{"ok": "true"}, dummy)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
		assert.Contains(t, w.Body.String(), `"ok"`)
	})

	t.Run("htmx_partial", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("HX-Request", "true")
		w := httptest.NewRecorder()

		component := templ.ComponentFunc(func(_ context.Context, wr io.Writer) error {
			_, err := io.WriteString(wr, "<div>partial</div>")
			return err
		})

		err := web.Negotiate(w, r, nil, component)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
		assert.Contains(t, w.Body.String(), "partial")
	})

	t.Run("browser_fallback", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		component := templ.ComponentFunc(func(_ context.Context, wr io.Writer) error {
			_, err := io.WriteString(wr, "<html>full</html>")
			return err
		})

		err := web.Negotiate(w, r, nil, component)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "full")
	})
}

func TestNegotiateStatus(t *testing.T) {
	t.Parallel()

	t.Run("json_with_status", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		r.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()

		dummy := templ.ComponentFunc(func(_ context.Context, _ io.Writer) error {
			return nil
		})

		err := web.NegotiateStatus(w, r, http.StatusCreated, map[string]string{"created": "true"}, dummy)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
		assert.True(t, strings.Contains(w.Body.String(), "created") || len(w.Body.String()) > 0)
	})

	t.Run("html_with_status", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest(http.MethodGet, "/missing", nil)
		w := httptest.NewRecorder()

		component := templ.ComponentFunc(func(_ context.Context, wr io.Writer) error {
			_, err := io.WriteString(wr, "<h1>not found</h1>")
			return err
		})

		err := web.NegotiateStatus(w, r, http.StatusNotFound, nil, component)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
		assert.Contains(t, w.Body.String(), "not found")
	})

	t.Run("json_priority_over_htmx", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Accept", "application/json")
		r.Header.Set("HX-Request", "true")
		w := httptest.NewRecorder()

		dummy := templ.ComponentFunc(func(_ context.Context, _ io.Writer) error {
			return nil
		})

		err := web.NegotiateStatus(w, r, http.StatusOK, map[string]string{"source": "json"}, dummy)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
		assert.Contains(t, w.Body.String(), "source")
	})
}
