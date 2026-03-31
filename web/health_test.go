package web_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dotcommander/gokart/web"
)

func TestHealthHandler(t *testing.T) {
	t.Parallel()

	t.Run("AllPass", func(t *testing.T) {
		t.Parallel()

		h := web.HealthHandler(
			web.HealthCheck{Name: "database", Fn: func(context.Context) error { return nil }},
			web.HealthCheck{Name: "cache", Fn: func(context.Context) error { return nil }},
		)

		rec := httptest.NewRecorder()
		h(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}

		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["status"] != "ok" {
			t.Errorf("status = %v, want ok", body["status"])
		}
		checks := body["checks"].(map[string]any)
		if checks["database"] != "ok" {
			t.Errorf("database = %v, want ok", checks["database"])
		}
		if checks["cache"] != "ok" {
			t.Errorf("cache = %v, want ok", checks["cache"])
		}
	})

	t.Run("SomeFail", func(t *testing.T) {
		t.Parallel()

		h := web.HealthHandler(
			web.HealthCheck{Name: "database", Fn: func(context.Context) error { return nil }},
			web.HealthCheck{Name: "cache", Fn: func(context.Context) error { return errors.New("connection refused") }},
		)

		rec := httptest.NewRecorder()
		h(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}

		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["status"] != "degraded" {
			t.Errorf("status = %v, want degraded", body["status"])
		}
		checks := body["checks"].(map[string]any)
		if checks["cache"] != "connection refused" {
			t.Errorf("cache = %v, want connection refused", checks["cache"])
		}
	})

	t.Run("AllFail", func(t *testing.T) {
		t.Parallel()

		h := web.HealthHandler(
			web.HealthCheck{Name: "database", Fn: func(context.Context) error { return errors.New("timeout") }},
			web.HealthCheck{Name: "cache", Fn: func(context.Context) error { return errors.New("connection refused") }},
		)

		rec := httptest.NewRecorder()
		h(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", rec.Code)
		}

		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["status"] != "unhealthy" {
			t.Errorf("status = %v, want unhealthy", body["status"])
		}
	})

	t.Run("PanicRecovery", func(t *testing.T) {
		t.Parallel()

		h := web.HealthHandler(
			web.HealthCheck{Name: "stable", Fn: func(context.Context) error { return nil }},
			web.HealthCheck{Name: "panicker", Fn: func(context.Context) error { panic("boom") }},
		)

		rec := httptest.NewRecorder()
		h(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

		// Handler must not propagate the panic — it should return a response.
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (degraded)", rec.Code)
		}

		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["status"] != "degraded" {
			t.Errorf("status = %v, want degraded", body["status"])
		}
		checks := body["checks"].(map[string]any)
		if checks["stable"] != "ok" {
			t.Errorf("stable = %v, want ok", checks["stable"])
		}
		if checks["panicker"] != "error" {
			t.Errorf("panicker = %v, want error", checks["panicker"])
		}
	})

	t.Run("EmptyChecks", func(t *testing.T) {
		t.Parallel()

		h := web.HealthHandler()
		rec := httptest.NewRecorder()
		h(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}

		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["status"] != "ok" {
			t.Errorf("status = %v, want ok", body["status"])
		}
	})
}

func TestReadyHandler(t *testing.T) {
	t.Parallel()

	t.Run("AllPass", func(t *testing.T) {
		t.Parallel()

		h := web.ReadyHandler(
			web.HealthCheck{Name: "database", Fn: func(context.Context) error { return nil }},
		)

		rec := httptest.NewRecorder()
		h(rec, httptest.NewRequest(http.MethodGet, "/ready", nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}

		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["status"] != "ok" {
			t.Errorf("status = %v, want ok", body["status"])
		}
	})

	t.Run("SomeFail", func(t *testing.T) {
		t.Parallel()

		h := web.ReadyHandler(
			web.HealthCheck{Name: "database", Fn: func(context.Context) error { return nil }},
			web.HealthCheck{Name: "cache", Fn: func(context.Context) error { return errors.New("connection refused") }},
		)

		rec := httptest.NewRecorder()
		h(rec, httptest.NewRequest(http.MethodGet, "/ready", nil))

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", rec.Code)
		}

		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["status"] != "unhealthy" {
			t.Errorf("status = %v, want unhealthy", body["status"])
		}
	})

	t.Run("EmptyChecks", func(t *testing.T) {
		t.Parallel()

		h := web.ReadyHandler()
		rec := httptest.NewRecorder()
		h(rec, httptest.NewRequest(http.MethodGet, "/ready", nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}

		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["status"] != "ok" {
			t.Errorf("status = %v, want ok", body["status"])
		}
	})

	t.Run("PanicTreatedAsFailure", func(t *testing.T) {
		t.Parallel()

		h := web.ReadyHandler(
			web.HealthCheck{Name: "panicker", Fn: func(context.Context) error { panic("boom") }},
		)

		rec := httptest.NewRecorder()
		h(rec, httptest.NewRequest(http.MethodGet, "/ready", nil))

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", rec.Code)
		}
	})

	t.Run("ContentType", func(t *testing.T) {
		t.Parallel()

		h := web.ReadyHandler()
		rec := httptest.NewRecorder()
		h(rec, httptest.NewRequest(http.MethodGet, "/ready", nil))

		ct := rec.Header().Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
	})
}
