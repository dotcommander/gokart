package web

import (
	"net/http"
	"time"
)

// Transport returns a production-tuned *http.Transport ready for an
// outbound HTTP client.
//
// It clones http.DefaultTransport and patches only what the Go
// defaults get wrong for high-concurrency callers:
// MaxIdleConnsPerHost 2 -> 100 (the default forces connection churn
// beyond 2 concurrent connections and is ~7x slower on a saturated
// upstream), MaxIdleConns 100, IdleConnTimeout 90s.
//
// Never construct &http.Transport{} from scratch to change one field:
// the zero value silently drops TLS handshake timeouts, HTTP/2,
// proxy-from-env, and the default dialer. Clone inherits them; only
// the connection-pool fields are overridden.
func Transport() *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 100
	transport.MaxIdleConnsPerHost = 100
	transport.IdleConnTimeout = 90 * time.Second
	return transport
}

// NewClient returns an *http.Client backed by Transport().
//
// Clients are safe for concurrent use and own their connection pool.
// Create one client per service at startup and reuse it; creating a
// new client per request or per goroutine defeats pooling entirely.
func NewClient() *http.Client {
	return &http.Client{Transport: Transport()}
}
