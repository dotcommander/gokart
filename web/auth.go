package web

import (
	"context"
	"net/http"
	"strings"
)

// authMiddleware returns middleware that extracts a credential from the
// request, validates it, and optionally enriches the context.
// Missing credential → 401, validation error → 403.
func authMiddleware(extract func(r *http.Request) (cred string, ok bool), validate func(ctx context.Context, cred string) (context.Context, error), missingMsg string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cred, ok := extract(r)
			if !ok {
				Error(w, http.StatusUnauthorized, missingMsg)
				return
			}

			ctx, err := validate(r.Context(), cred)
			if err != nil {
				Error(w, http.StatusForbidden, err.Error())
				return
			}

			if ctx != nil {
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// APIKeyAuth returns middleware that validates API keys from the X-API-Key
// header or api_key query parameter. keyFn validates the key and optionally
// enriches the request context. Missing key → 401, keyFn error → 403.
func APIKeyAuth(keyFn func(ctx context.Context, key string) (context.Context, error)) func(http.Handler) http.Handler {
	return authMiddleware(
		func(r *http.Request) (string, bool) {
			key := r.Header.Get("X-API-Key")
			if key == "" {
				key = r.URL.Query().Get("api_key")
			}
			return key, key != ""
		},
		keyFn,
		"missing api key",
	)
}

// BearerAuth returns middleware that validates Bearer tokens from the
// Authorization header. tokenFn validates the token and optionally enriches
// the request context. Missing/malformed → 401, tokenFn error → 403.
func BearerAuth(tokenFn func(ctx context.Context, token string) (context.Context, error)) func(http.Handler) http.Handler {
	return authMiddleware(
		func(r *http.Request) (string, bool) {
			header := r.Header.Get("Authorization")
			token, ok := strings.CutPrefix(header, "Bearer ")
			return token, ok && token != ""
		},
		tokenFn,
		"invalid authorization header",
	)
}
