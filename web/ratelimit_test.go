package web_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dotcommander/gokart/web"
)

func TestRateLimit_AllowsUnderBurst(t *testing.T) {
	t.Parallel()

	handler := web.RateLimit(1, 2)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Two requests should succeed (burst = 2).
	for range 2 {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	}
}

func TestRateLimit_BlocksOverBurst(t *testing.T) {
	t.Parallel()

	handler := web.RateLimit(1, 2)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Drain the burst.
	for range 2 {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		handler.ServeHTTP(rec, req)
	}

	// Third request should be rate limited.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
}

func TestRateLimit_429HasRetryAfter(t *testing.T) {
	t.Parallel()

	handler := web.RateLimit(1, 1)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Drain burst.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	// Second request should have Retry-After header.
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header to be set")
	}
}

func TestRateLimit_429JSON(t *testing.T) {
	t.Parallel()

	handler := web.RateLimit(1, 1)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Drain burst.
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	// Second request should return JSON error.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if body["error"] != "rate limit exceeded" {
		t.Fatalf("expected error 'rate limit exceeded', got %q", body["error"])
	}
}

func TestRateLimit_PerIPIsolation(t *testing.T) {
	t.Parallel()

	handler := web.RateLimit(1, 1)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request from IP A drains its bucket.
	reqA := httptest.NewRequest(http.MethodGet, "/", nil)
	reqA.RemoteAddr = "1.2.3.4:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, reqA)
	if rec.Code != http.StatusOK {
		t.Fatalf("IP A first request: expected 200, got %d", rec.Code)
	}

	// Second request from same IP A should be limited.
	reqA = httptest.NewRequest(http.MethodGet, "/", nil)
	reqA.RemoteAddr = "1.2.3.4:1234"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, reqA)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("IP A second request: expected 429, got %d", rec.Code)
	}

	// Request from IP B should still succeed (separate bucket).
	reqB := httptest.NewRequest(http.MethodGet, "/", nil)
	reqB.RemoteAddr = "5.6.7.8:5678"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, reqB)
	if rec.Code != http.StatusOK {
		t.Fatalf("IP B request: expected 200, got %d", rec.Code)
	}
}

func TestRateLimitWithKey_CustomKey(t *testing.T) {
	t.Parallel()

	keyFn := func(r *http.Request) string {
		return r.Header.Get("X-API-Key")
	}

	handler := web.RateLimitWithKey(1, 1, keyFn)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Drain burst for key "a".
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "a")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("key 'a' first request: expected 200, got %d", rec.Code)
	}

	// Second request for key "a" should be limited.
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "a")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("key 'a' second request: expected 429, got %d", rec.Code)
	}

	// Request for key "b" should succeed.
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "b")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("key 'b' request: expected 200, got %d", rec.Code)
	}
}
