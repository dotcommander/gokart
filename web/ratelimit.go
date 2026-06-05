package web

import (
	"log/slog"
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

// defaultLimiterTTL is the duration an untouched per-key limiter survives before
// the background sweeper evicts it. Keys touched within this window are retained.
const defaultLimiterTTL = 10 * time.Minute

// defaultSweepInterval is how often the background goroutine scans for and
// deletes expired limiters.
const defaultSweepInterval = time.Minute

// rateLimitConfig holds eviction tuning resolved from RateLimitOption values.
type rateLimitConfig struct {
	ttl           time.Duration // 0 disables eviction (limiters kept indefinitely).
	sweepInterval time.Duration
	now           func() time.Time // injectable clock for deterministic tests.
	beforeEvict   func(key string) // injectable hook for eviction interleaving tests.
}

// RateLimitOption configures the eviction behavior of a rate limiter.
type RateLimitOption func(*rateLimitConfig)

// WithTTL sets the idle duration after which an untouched limiter is evicted.
// A ttl <= 0 disables eviction (limiters are kept indefinitely).
func WithTTL(ttl time.Duration) RateLimitOption {
	return func(c *rateLimitConfig) { c.ttl = ttl }
}

// WithSweepInterval sets how often the background sweeper scans for expired
// limiters. Ignored when eviction is disabled.
func WithSweepInterval(d time.Duration) RateLimitOption {
	return func(c *rateLimitConfig) {
		if d > 0 {
			c.sweepInterval = d
		}
	}
}

// withClock injects a clock for deterministic testing (unexported).
func withClock(now func() time.Time) RateLimitOption {
	return func(c *rateLimitConfig) { c.now = now }
}

// withBeforeEvict injects a hook after an entry is initially judged stale but
// before deletion. It is unexported because production callers should not alter
// eviction internals.
func withBeforeEvict(fn func(key string)) RateLimitOption {
	return func(c *rateLimitConfig) { c.beforeEvict = fn }
}

// limiterEntry pairs a token-bucket limiter with its last-seen timestamp
// (unix nanos) so the sweeper can decide eviction without deleting a key that
// refreshed after the stale check began.
type limiterEntry struct {
	limiter  *rate.Limiter
	mu       sync.Mutex
	lastSeen int64
}

// RateLimiter is a per-key token-bucket rate limiter with optional TTL
// eviction. Construct via RateLimitWithEviction; call Stop to release the
// background sweeper goroutine when the limiter is discarded.
type RateLimiter struct {
	rps        float64
	burst      int
	retryAfter string
	keyFn      func(r *http.Request) string

	cfg      rateLimitConfig
	limiters sync.Map // key string -> *limiterEntry
	count    atomic.Int64

	stop     chan struct{}
	stopOnce sync.Once
}

// RateLimit returns middleware that limits requests per IP using a token
// bucket. rps is the rate of tokens added per second; burst is the maximum
// burst size. Limiters idle beyond defaultLimiterTTL are evicted by a
// background sweeper so the limiter map cannot grow unbounded.
func RateLimit(rps float64, burst int, opts ...RateLimitOption) func(http.Handler) http.Handler {
	return RateLimitWithKey(rps, burst, remoteAddrKey, opts...)
}

// RateLimitWithKey returns middleware that uses a custom key function instead
// of remote IP for rate limiting. By default idle limiters are evicted after
// defaultLimiterTTL; pass WithTTL(0) to retain them indefinitely.
//
// This wrapper owns the lifecycle of the background sweeper internally. If you
// need to stop the sweeper or observe the live limiter count, use
// RateLimitWithEviction, which returns a *RateLimiter handle.
func RateLimitWithKey(rps float64, burst int, keyFn func(r *http.Request) string, opts ...RateLimitOption) func(http.Handler) http.Handler {
	return RateLimitWithEviction(rps, burst, keyFn, opts...).Middleware()
}

// RateLimitWithEviction builds a *RateLimiter with TTL eviction and starts its
// background sweeper. The caller owns the returned handle: call Stop to release
// the sweeper goroutine, and LimiterCount to observe the live map size.
func RateLimitWithEviction(rps float64, burst int, keyFn func(r *http.Request) string, opts ...RateLimitOption) *RateLimiter {
	if rps <= 0 {
		panic("ratelimit: rps must be positive")
	}

	cfg := rateLimitConfig{
		ttl:           defaultLimiterTTL,
		sweepInterval: defaultSweepInterval,
		now:           time.Now,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.now == nil {
		cfg.now = time.Now
	}

	rl := &RateLimiter{
		rps:        rps,
		burst:      burst,
		retryAfter: strconv.Itoa(int(math.Ceil(1 / rps))),
		keyFn:      keyFn,
		cfg:        cfg,
		stop:       make(chan struct{}),
	}

	// Eviction enabled only when ttl > 0; otherwise behavior matches the old
	// store-forever semantics and no goroutine is started.
	if cfg.ttl > 0 {
		go rl.sweepLoop()
	}

	return rl
}

// Middleware returns the http middleware bound to this RateLimiter.
func (rl *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			limiter := rl.limiterFor(rl.keyFn(r))

			if !limiter.Allow() {
				w.Header().Set("Retry-After", rl.retryAfter)
				Error(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// limiterFor returns the limiter for key, creating it on first use, and stamps
// its last-seen time so the sweeper retains actively-used keys.
func (rl *RateLimiter) limiterFor(key string) *rate.Limiter {
	now := rl.cfg.now().UnixNano()
	for {
		if v, ok := rl.limiters.Load(key); ok {
			e := v.(*limiterEntry)
			e.mu.Lock()
			current, stillStored := rl.limiters.Load(key)
			if !stillStored || current != e {
				e.mu.Unlock()
				continue
			}
			e.lastSeen = now
			limiter := e.limiter
			e.mu.Unlock()
			return limiter
		}

		e := &limiterEntry{limiter: rate.NewLimiter(rate.Limit(rl.rps), rl.burst), lastSeen: now}
		if _, loaded := rl.limiters.LoadOrStore(key, e); loaded {
			continue
		}
		rl.count.Add(1)
		return e.limiter
	}
}

// LimiterCount returns the number of live per-key limiters currently held.
func (rl *RateLimiter) LimiterCount() int {
	return int(rl.count.Load())
}

// Stop terminates the background sweeper goroutine. Safe to call multiple
// times. After Stop, eviction no longer runs but the limiter remains usable.
func (rl *RateLimiter) Stop() {
	rl.stopOnce.Do(func() { close(rl.stop) })
}

// sweepLoop is the background eviction goroutine. It returns when rl.stop is
// closed (its sole exit condition).
func (rl *RateLimiter) sweepLoop() {
	ticker := time.NewTicker(rl.cfg.sweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-rl.stop:
			return
		case <-ticker.C:
			rl.evictNow()
		}
	}
}

// evictNow deletes every limiter untouched for longer than the configured TTL.
// Separated from the ticker so tests can drive eviction deterministically.
func (rl *RateLimiter) evictNow() {
	if rl.cfg.ttl <= 0 {
		return
	}
	cutoff := rl.cfg.now().Add(-rl.cfg.ttl).UnixNano()
	evicted := 0
	rl.limiters.Range(func(k, v any) bool {
		e := v.(*limiterEntry)
		e.mu.Lock()
		if e.lastSeen < cutoff {
			if key, ok := k.(string); ok && rl.cfg.beforeEvict != nil {
				rl.cfg.beforeEvict(key)
			}
		}
		if e.lastSeen < cutoff && rl.limiters.CompareAndDelete(k, e) {
			rl.count.Add(-1)
			evicted++
		}
		e.mu.Unlock()
		return true
	})
	if evicted > 0 {
		slog.Debug("ratelimit: evicted idle limiters", "evicted", evicted, "remaining", rl.count.Load())
	}
}

func remoteAddrKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
