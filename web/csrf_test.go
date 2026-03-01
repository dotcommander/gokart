package web_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dotcommander/gokart/web"
)

// okHandler is a simple handler that always returns 200 with body "ok".
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok")) //nolint:errcheck
})

func TestCSRFProtect(t *testing.T) {
	t.Parallel()

	mw := web.CSRFProtect()
	handler := mw(okHandler)

	tests := []struct {
		name           string
		method         string
		secFetchSite   string
		wantStatusCode int
	}{
		{
			name:           "GET request is always allowed",
			method:         http.MethodGet,
			secFetchSite:   "cross-site",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "POST same-origin is allowed",
			method:         http.MethodPost,
			secFetchSite:   "same-origin",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "POST cross-site is rejected",
			method:         http.MethodPost,
			secFetchSite:   "cross-site",
			wantStatusCode: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(tt.method, "/", nil)
			if tt.secFetchSite != "" {
				req.Header.Set("Sec-Fetch-Site", tt.secFetchSite)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatusCode {
				t.Errorf("expected status %d, got %d", tt.wantStatusCode, rec.Code)
			}
		})
	}
}

func TestCSRFProtectWithOrigins(t *testing.T) {
	t.Parallel()

	trustedOrigin := "https://app.example.com"

	tests := []struct {
		name           string
		origins        []string
		method         string
		secFetchSite   string
		origin         string
		wantBuildErr   bool
		wantStatusCode int
	}{
		{
			name:           "POST from trusted origin is allowed",
			origins:        []string{trustedOrigin},
			method:         http.MethodPost,
			secFetchSite:   "cross-site",
			origin:         trustedOrigin,
			wantBuildErr:   false,
			wantStatusCode: http.StatusOK,
		},
		{
			name:         "invalid origin string returns build error",
			origins:      []string{"not a valid origin %%"},
			wantBuildErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mw, err := web.CSRFProtectWithOrigins(tt.origins...)
			if tt.wantBuildErr {
				if err == nil {
					t.Fatal("expected error from CSRFProtectWithOrigins, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error from CSRFProtectWithOrigins: %v", err)
			}

			handler := mw(okHandler)

			req := httptest.NewRequest(tt.method, "/", nil)
			if tt.secFetchSite != "" {
				req.Header.Set("Sec-Fetch-Site", tt.secFetchSite)
			}
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatusCode {
				t.Errorf("expected status %d, got %d", tt.wantStatusCode, rec.Code)
			}
		})
	}
}

func TestCSRFProtect_SafeMethods(t *testing.T) {
	t.Parallel()

	mw := web.CSRFProtect()
	handler := mw(okHandler)

	safeMethods := []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodOptions,
	}

	for _, method := range safeMethods {
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(method, "/", nil)
			req.Header.Set("Sec-Fetch-Site", "cross-site")
			req.Header.Set("Origin", "https://evil.com")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("%s with cross-site headers: expected 200, got %d", method, rec.Code)
			}
		})
	}
}
