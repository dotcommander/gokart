package web_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dotcommander/gokart/web"
)

func TestNewRouter(t *testing.T) {
	t.Parallel()

	router := web.NewRouter(web.RouterConfig{
		Middleware: web.StandardMiddleware,
		Timeout:    5 * time.Second,
	})

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("expected body 'ok', got %q", rec.Body.String())
	}
}

func ExampleNewRouter() {
	router := web.NewRouter(web.RouterConfig{
		Middleware: web.StandardMiddleware,
		Timeout:    30 * time.Second,
	})

	router.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
