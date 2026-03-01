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

func TestNewHTTPClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config web.HTTPConfig
	}{
		{
			name:   "defaults",
			config: web.HTTPConfig{},
		},
		{
			name: "custom timeout",
			config: web.HTTPConfig{
				Timeout:   10 * time.Second,
				RetryMax:  5,
				RetryWait: 2 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := web.NewHTTPClient(tt.config)
			if client == nil {
				t.Fatal("expected client, got nil")
			}
		})
	}
}

func TestNewStandardClient(t *testing.T) {
	t.Parallel()

	client := web.NewStandardClient()
	if client == nil {
		t.Fatal("expected client, got nil")
	}

	var _ *http.Client = client
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

func ExampleNewHTTPClient() {
	client := web.NewHTTPClient(web.HTTPConfig{
		Timeout:   10 * time.Second,
		RetryMax:  3,
		RetryWait: 1 * time.Second,
	})

	_ = client
}
