package web

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func TestRateLimiter_EvictsIdleKeys(t *testing.T) {
	t.Parallel()

	var nowNanos atomic.Int64
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	nowNanos.Store(base.UnixNano())
	clock := func() time.Time { return time.Unix(0, nowNanos.Load()) }

	keyHeader := func(r *http.Request) string { return r.Header.Get("X-Key") }

	// High rps/burst so Allow never blocks; we only care about map growth.
	rl := RateLimitWithEviction(1000, 1000, keyHeader,
		WithTTL(5*time.Minute),
		withClock(clock),
	)
	defer rl.Stop()
	handler := rl.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create 1000 unique keys.
	for i := range 1000 {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Key", "k"+strconv.Itoa(i))
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}
	if got := rl.LimiterCount(); got != 1000 {
		t.Fatalf("after 1000 unique keys: LimiterCount = %d, want 1000", got)
	}

	// Advance the clock past the TTL so every key is now idle-expired.
	nowNanos.Store(base.Add(6 * time.Minute).UnixNano())
	rl.evictNow()

	if got := rl.LimiterCount(); got != 0 {
		t.Fatalf("after TTL eviction: LimiterCount = %d, want 0", got)
	}
}

func TestRateLimiter_RetainsActiveKeys(t *testing.T) {
	t.Parallel()

	var nowNanos atomic.Int64
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	nowNanos.Store(base.UnixNano())
	clock := func() time.Time { return time.Unix(0, nowNanos.Load()) }

	keyHeader := func(r *http.Request) string { return r.Header.Get("X-Key") }
	rl := RateLimitWithEviction(1000, 1000, keyHeader,
		WithTTL(5*time.Minute),
		withClock(clock),
	)
	defer rl.Stop()
	handler := rl.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	touch := func(key string) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Key", key)
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}

	touch("stale")
	touch("active")

	// Advance 4 minutes (within TTL), re-touch "active" only.
	nowNanos.Store(base.Add(4 * time.Minute).UnixNano())
	touch("active")

	// Advance to 6 minutes total: "stale" last seen at t=0 (idle 6m, expired),
	// "active" last seen at t=4m (idle 2m, retained).
	nowNanos.Store(base.Add(6 * time.Minute).UnixNano())
	rl.evictNow()

	if got := rl.LimiterCount(); got != 1 {
		t.Fatalf("expected only active key retained: LimiterCount = %d, want 1", got)
	}
}

func TestRateLimiter_StopIsIdempotent(t *testing.T) {
	t.Parallel()

	rl := RateLimitWithEviction(1, 1, remoteAddrKey, WithTTL(time.Minute))
	rl.Stop()
	rl.Stop() // must not panic (double-close guarded by sync.Once).
}

func TestRateLimiter_TTLZeroDisablesEviction(t *testing.T) {
	t.Parallel()

	var nowNanos atomic.Int64
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	nowNanos.Store(base.UnixNano())
	clock := func() time.Time { return time.Unix(0, nowNanos.Load()) }

	keyHeader := func(r *http.Request) string { return r.Header.Get("X-Key") }
	rl := RateLimitWithEviction(1000, 1000, keyHeader,
		WithTTL(0), // eviction disabled
		withClock(clock),
	)
	defer rl.Stop()
	handler := rl.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Key", "x")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	nowNanos.Store(base.Add(100 * time.Hour).UnixNano())
	rl.evictNow() // no-op when ttl <= 0

	if got := rl.LimiterCount(); got != 1 {
		t.Fatalf("ttl=0 must retain limiters: LimiterCount = %d, want 1", got)
	}
}

func TestRateLimiter_EvictionRechecksAfterRefresh(t *testing.T) {
	t.Parallel()

	var nowNanos atomic.Int64
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	nowNanos.Store(base.UnixNano())
	clock := func() time.Time { return time.Unix(0, nowNanos.Load()) }

	keyHeader := func(r *http.Request) string { return r.Header.Get("X-Key") }
	var refreshed atomic.Bool
	var rl *RateLimiter
	rl = RateLimitWithEviction(1, 1, keyHeader,
		WithTTL(5*time.Minute),
		withClock(clock),
		withBeforeEvict(func(key string) {
			if key != "active" {
				return
			}
			v, ok := rl.limiters.Load(key)
			if !ok {
				return
			}
			v.(*limiterEntry).lastSeen = base.Add(6 * time.Minute).UnixNano()
			refreshed.Store(true)
		}),
	)
	defer rl.Stop()

	handler := rl.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Key", "active")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	nowNanos.Store(base.Add(6 * time.Minute).UnixNano())
	rl.evictNow()

	if !refreshed.Load() {
		t.Fatal("test hook did not run")
	}
	if got := rl.LimiterCount(); got != 1 {
		t.Fatalf("refreshed key should survive eviction: LimiterCount = %d, want 1", got)
	}
}
