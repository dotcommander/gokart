package web_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/dotcommander/gokart/web"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestShouldRetryHTTP(t *testing.T) {
	t.Parallel()
	if !web.ShouldRetryHTTP(nil, errors.New("boom")) {
		t.Error("network error should retry")
	}
	for _, code := range []int{408, 429, 500, 502, 503, 504} {
		if !web.ShouldRetryHTTP(&http.Response{StatusCode: code}, nil) {
			t.Errorf("status %d should retry", code)
		}
	}
	for _, code := range []int{200, 400, 401, 403, 404, 422} {
		if web.ShouldRetryHTTP(&http.Response{StatusCode: code}, nil) {
			t.Errorf("status %d should not retry", code)
		}
	}
}

func TestRetryTransportRetriesTransient(t *testing.T) {
	t.Parallel()
	calls := 0
	next := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls < 3 {
			return &http.Response{StatusCode: 503, Body: io.NopCloser(strings.NewReader(""))}, nil
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok"))}, nil
	})
	rt := &web.RetryTransport{Next: next, Attempts: 3, Initial: time.Millisecond, MaxDelay: 5 * time.Millisecond,
		Sleep: func(context.Context, time.Duration) error { return nil }}
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3", calls)
	}
}

func TestRetryTransportDoesNotRetry4xx(t *testing.T) {
	t.Parallel()
	calls := 0
	next := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	rt := &web.RetryTransport{Next: next, Attempts: 3, Initial: time.Millisecond,
		Sleep: func(context.Context, time.Duration) error { return nil }}
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 404 {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestRetryTransportHonorsContextCancel(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	next := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 503, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	rt := &web.RetryTransport{Next: next, Attempts: 3, Initial: time.Millisecond}
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	req = req.WithContext(ctx)
	_, err := rt.RoundTrip(req)
	if err == nil {
		t.Fatal("want context error")
	}
}
