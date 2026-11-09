package web

import (
	"context"
	"errors"
	"io"
	"math"
	"math/rand"
	"net/http"
	"time"
)

// RetryPolicy decides whether an HTTP operation may be retried.
//
// Retries belong at the operation layer: idempotency depends on the
// method, body replayability, idempotency keys, and response
// classification — never configure automatic retries on every request.
type RetryPolicy interface {
	ShouldRetry(resp *http.Response, err error) bool
}

// ShouldRetryHTTP returns true for network errors, 408, 429, and
// 5xx responses. Other 4xx are permanent caller errors.
func ShouldRetryHTTP(resp *http.Response, err error) bool {
	if err != nil {
		return true
	}
	if resp == nil {
		return false
	}
	switch resp.StatusCode {
	case http.StatusRequestTimeout, http.StatusTooManyRequests,
		http.StatusInternalServerError, http.StatusBadGateway,
		http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	return false
}

// RetryTransport is an explicit RoundTripper that retries using the
// client request as-is. The caller must supply a replayable request
// (clone per attempt) — this wrapper does not buffer bodies.
type RetryTransport struct {
	Next       http.RoundTripper
	Attempts   int
	Initial    time.Duration
	MaxDelay   time.Duration
	Sleep      func(ctx context.Context, d time.Duration) error
	JitterRand func() float64
}

func (t *RetryTransport) next() http.RoundTripper {
	if t.Next != nil {
		return t.Next
	}
	return Transport()
}

func (t *RetryTransport) sleep(ctx context.Context, d time.Duration) error {
	if t.Sleep != nil {
		return t.Sleep(ctx, d)
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// RoundTrip executes the request, retrying transient failures with
// jittered exponential backoff: base/2 + uniform(0, base/2) where
// base = min(Initial * 2^attempt, MaxDelay).
func (t *RetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	attempts := t.Attempts
	if attempts < 1 {
		return nil, errors.New("retry transport: attempts must be positive")
	}
	next := t.next()
	ctx := req.Context()
	var err error
	for attempt := 1; attempt <= attempts; attempt++ {
		var resp *http.Response
		resp, err = next.RoundTrip(req)
		if err == nil && resp != nil && !ShouldRetryHTTP(resp, nil) {
			return resp, nil
		}
		if attempt == attempts {
			return resp, err
		}
		if resp != nil && resp.Body != nil {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
			_ = resp.Body.Close()
		}
		delay := t.Initial
		if delay < 1 {
			delay = 100 * time.Millisecond
		}
		maxDelay := t.MaxDelay
		if maxDelay < delay {
			maxDelay = delay
		}
		base := math.Min(float64(delay)*math.Pow(2, float64(attempt-1)), float64(maxDelay))
		var jitter float64
		if t.JitterRand != nil {
			jitter = t.JitterRand()
		} else {
			jitter = 0.5 + rand.Float64()*0.25 //nolint:gosec // jitter, not security
		}
		if err := t.sleep(ctx, time.Duration(base*jitter)); err != nil {
			return nil, err
		}
	}
	return nil, err
}

// NewRetryClient returns an *http.Client whose transport retries
// transient failures. Wrap Transport() explicitly; do not rely on
// transport-level retries hiding caller errors.
func NewRetryClient(rt *RetryTransport) *http.Client {
	return &http.Client{Transport: rt}
}
