package web

import (
	"fmt"
	"net/http"
)

// CSRFProtect returns a chi-compatible middleware that guards against
// cross-site request forgery using the stdlib CrossOriginProtection.
//
// It uses [http.NewCrossOriginProtection], which inspects Sec-Fetch-Site
// and Origin headers set by modern browsers. Non-browser clients (those
// that omit these headers) are allowed through unchanged.
//
// Example:
//
//	router := web.NewRouter(web.RouterConfig{
//	    Middleware: append(web.StandardMiddleware, web.CSRFProtect()),
//	})
func CSRFProtect() func(http.Handler) http.Handler {
	p := http.NewCrossOriginProtection()
	return func(next http.Handler) http.Handler {
		return p.Handler(next)
	}
}

// CSRFProtectWithOrigins is like [CSRFProtect] but also registers additional
// trusted origins. Requests from a trusted origin are allowed even when the
// Sec-Fetch-Site header indicates a cross-site context.
//
// Each origin must be a valid URL origin (scheme + host + optional port),
// e.g. "https://app.example.com". An error is returned if any origin string
// is malformed.
//
// Example:
//
//	mw, err := web.CSRFProtectWithOrigins("https://app.example.com", "https://cdn.example.com")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	router := web.NewRouter(web.RouterConfig{
//	    Middleware: append(web.StandardMiddleware, mw),
//	})
func CSRFProtectWithOrigins(origins ...string) (func(http.Handler) http.Handler, error) {
	p := http.NewCrossOriginProtection()
	for _, origin := range origins {
		if err := p.AddTrustedOrigin(origin); err != nil {
			return nil, fmt.Errorf("csrf: invalid origin %q: %w", origin, err)
		}
	}
	return func(next http.Handler) http.Handler {
		return p.Handler(next)
	}, nil
}
