package gokart_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dotcommander/gokart"
)

func TestNewLogger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config gokart.LogConfig
	}{
		{
			name:   "defaults",
			config: gokart.LogConfig{},
		},
		{
			name:   "debug json",
			config: gokart.LogConfig{Level: "debug", Format: "json"},
		},
		{
			name:   "warn text",
			config: gokart.LogConfig{Level: "warn", Format: "text"},
		},
		{
			name:   "custom output",
			config: gokart.LogConfig{Output: &bytes.Buffer{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := gokart.NewLogger(tt.config)
			if log == nil {
				t.Fatal("expected logger, got nil")
			}
			// Verify logger works
			log.Info("test message", "key", "value")
		})
	}
}

func TestNewRouter(t *testing.T) {
	t.Parallel()

	router := gokart.NewRouter(gokart.RouterConfig{
		Middleware: gokart.StandardMiddleware,
		Timeout:    5 * time.Second,
	})

	// Add a test route
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Test the route
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
		config gokart.HTTPConfig
	}{
		{
			name:   "defaults",
			config: gokart.HTTPConfig{},
		},
		{
			name: "custom timeout",
			config: gokart.HTTPConfig{
				Timeout:   10 * time.Second,
				RetryMax:  5,
				RetryWait: 2 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := gokart.NewHTTPClient(tt.config)
			if client == nil {
				t.Fatal("expected client, got nil")
			}
		})
	}
}

func TestNewStandardClient(t *testing.T) {
	t.Parallel()

	client := gokart.NewStandardClient()
	if client == nil {
		t.Fatal("expected client, got nil")
	}

	// Test it can be used as http.Client
	var _ *http.Client = client
}

func ExampleNewLogger() {
	log := gokart.NewLogger(gokart.LogConfig{
		Level:  "info",
		Format: "json",
	})
	log.Info("server started", "port", 8080)
}

func ExampleNewRouter() {
	router := gokart.NewRouter(gokart.RouterConfig{
		Middleware: gokart.StandardMiddleware,
		Timeout:    30 * time.Second,
	})

	router.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// http.ListenAndServe(":8080", router)
}

func ExampleNewHTTPClient() {
	client := gokart.NewHTTPClient(gokart.HTTPConfig{
		Timeout:   10 * time.Second,
		RetryMax:  3,
		RetryWait: 1 * time.Second,
	})

	_ = client // Use client.Get(), client.Post(), etc.
}
