package web_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dotcommander/gokart/web"
)

func TestJSONStatusUnencodableReturns500(t *testing.T) {
	rec := httptest.NewRecorder()
	web.JSONStatus(rec, http.StatusOK, func() {})
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := rec.Body.String(); !strings.Contains(got, "json encode failed") {
		t.Fatalf("body = %q, want json encode failed fallback", got)
	}
}

func TestJSONStatusEUnencodableReturnsError(t *testing.T) {
	rec := httptest.NewRecorder()
	err := web.JSONStatusE(rec, http.StatusOK, make(chan int))
	if err == nil {
		t.Fatal("JSONStatusE with unencodable value error = nil, want error")
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestJSONStatusEncodableUnchanged(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	web.JSONStatus(rec, http.StatusCreated, map[string]string{"id": "1"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
	if got := rec.Body.String(); got != "{\"id\":\"1\"}\n" {
		t.Fatalf("body = %q", got)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q", ct)
	}
}

func TestPlainResponseHelpers(t *testing.T) {
	t.Parallel()
	t.Run("JSON", func(t *testing.T) {
		t.Parallel()
		rec := httptest.NewRecorder()
		web.JSON(rec, map[string]string{"ok": "true"})
		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `"ok":"true"`) {
			t.Errorf("body = %q", rec.Body.String())
		}
	})
	t.Run("Error", func(t *testing.T) {
		t.Parallel()
		rec := httptest.NewRecorder()
		web.Error(rec, http.StatusBadRequest, "nope")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `"error":"nope"`) {
			t.Errorf("body = %q", rec.Body.String())
		}
	})
	t.Run("NoContent", func(t *testing.T) {
		t.Parallel()
		rec := httptest.NewRecorder()
		web.NoContent(rec)
		if rec.Code != http.StatusNoContent {
			t.Errorf("status = %d, want 204", rec.Code)
		}
		if rec.Body.Len() != 0 {
			t.Errorf("body length = %d, want 0", rec.Body.Len())
		}
	})
}
