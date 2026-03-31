package web_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dotcommander/gokart/web"
)

// authOKHandler returns 200 with body "ok".
var authOKHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok")) //nolint:errcheck
})

// validAPIKey is the only key accepted by the test keyFn.
const validAPIKey = "test-key"

// validBearerToken is the only token accepted by the test tokenFn.
const validBearerToken = "test-token"

type contextKey string

const userKey contextKey = "user"

func testKeyFn(_ context.Context, key string) (context.Context, error) {
	if key != validAPIKey {
		return nil, fmt.Errorf("invalid")
	}
	return context.WithValue(context.Background(), userKey, "api-user"), nil
}

func testTokenFn(_ context.Context, token string) (context.Context, error) {
	if token != validBearerToken {
		return nil, fmt.Errorf("invalid")
	}
	return context.WithValue(context.Background(), userKey, "bearer-user"), nil
}

func TestAPIKeyAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		header         string
		query          string
		wantStatus     int
		wantBody       string
		wantContextVal any // non-nil means handler should see this in context
	}{
		{
			name:       "ValidHeader",
			header:     validAPIKey,
			wantStatus: http.StatusOK,
			wantBody:   "ok",
		},
		{
			name:       "ValidQuery",
			query:      validAPIKey,
			wantStatus: http.StatusOK,
			wantBody:   "ok",
		},
		{
			name:       "MissingKey",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "InvalidKey",
			header:     "wrong-key",
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := web.APIKeyAuth(testKeyFn)(authOKHandler)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("X-API-Key", tt.header)
			}
			if tt.query != "" {
				q := req.URL.Query()
				q.Set("api_key", tt.query)
				req.URL.RawQuery = q.Encode()
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestAPIKeyAuth_ContextPropagation(t *testing.T) {
	t.Parallel()

	var gotVal any

	checkHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotVal = r.Context().Value(userKey)
		w.WriteHeader(http.StatusOK)
	})

	mw := web.APIKeyAuth(testKeyFn)(checkHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", validAPIKey)
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if gotVal != "api-user" {
		t.Errorf("expected context value %q, got %v", "api-user", gotVal)
	}
}

func TestBearerAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		authHeader    string
		wantStatus    int
		wantErrorBody string // expected error message in JSON body, empty means no check
	}{
		{
			name:       "ValidToken",
			authHeader: "Bearer " + validBearerToken,
			wantStatus: http.StatusOK,
		},
		{
			name:       "MissingToken",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:          "MalformedHeader",
			authHeader:    "Token xxx",
			wantStatus:    http.StatusUnauthorized,
			wantErrorBody: "invalid authorization header",
		},
		{
			name:       "InvalidToken",
			authHeader: "Bearer wrong-token",
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := web.BearerAuth(testTokenFn)(authOKHandler)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}

			if tt.wantErrorBody != "" {
				var body map[string]string
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("failed to decode JSON body: %v", err)
				}
				if body["error"] != tt.wantErrorBody {
					t.Errorf("expected error %q, got %q", tt.wantErrorBody, body["error"])
				}
			}
		})
	}
}

func TestBearerAuth_ContextPropagation(t *testing.T) {
	t.Parallel()

	var gotVal any

	checkHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotVal = r.Context().Value(userKey)
		w.WriteHeader(http.StatusOK)
	})

	mw := web.BearerAuth(testTokenFn)(checkHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+validBearerToken)
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if gotVal != "bearer-user" {
		t.Errorf("expected context value %q, got %v", "bearer-user", gotVal)
	}
}
