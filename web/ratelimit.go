package web

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"

	"golang.org/x/time/rate"
)

// RateLimit returns middleware that limits requests per IP using a token bucket.
// rps is the rate of tokens added per second; burst is the maximum burst size.
func RateLimit(rps float64, burst int) func(http.Handler) http.Handler {
	return RateLimitWithKey(rps, burst, remoteAddrKey)
}

// RateLimitWithKey returns middleware that uses a custom key function
// instead of remote IP for rate limiting.
//
// Note: per-key limiters are stored indefinitely. For high-cardinality keys,
// consider an external rate limiter (e.g., Redis-based).
func RateLimitWithKey(rps float64, burst int, keyFn func(r *http.Request) string) func(http.Handler) http.Handler {
	if rps <= 0 {
		panic("ratelimit: rps must be positive")
	}

	retryAfter := strconv.Itoa(int(math.Ceil(1 / rps)))
	var limiters sync.Map

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFn(r)

			lim, _ := limiters.LoadOrStore(key, rate.NewLimiter(rate.Limit(rps), burst))
			limiter := lim.(*rate.Limiter)

			if !limiter.Allow() {
				w.Header().Set("Retry-After", retryAfter)
				Error(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func remoteAddrKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
