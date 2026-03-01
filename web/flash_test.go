package web_test

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dotcommander/gokart/web"
)

func TestSetFlash(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	web.SetFlash(rec, web.FlashSuccess, "Item created")

	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected flash cookie to be set")
	}

	cookie := cookies[0]
	if cookie.Name != "_flash" {
		t.Errorf("cookie name = %q, want _flash", cookie.Name)
	}
	if cookie.Path != "/" {
		t.Errorf("cookie path = %q, want /", cookie.Path)
	}
	if !cookie.HttpOnly {
		t.Error("expected HttpOnly cookie")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want Lax", cookie.SameSite)
	}
}

func TestGetFlash(t *testing.T) {
	t.Parallel()

	t.Run("reads and clears", func(t *testing.T) {
		t.Parallel()
		// Set the flash
		setRec := httptest.NewRecorder()
		web.SetFlash(setRec, web.FlashError, "Something failed")
		setCookies := setRec.Result().Cookies()

		// Read the flash
		r := httptest.NewRequest("GET", "/", nil)
		for _, c := range setCookies {
			r.AddCookie(c)
		}
		getRec := httptest.NewRecorder()
		flash := web.GetFlash(getRec, r)

		if flash == nil {
			t.Fatal("expected flash message")
		}
		if flash.Level != web.FlashError {
			t.Errorf("level = %q, want error", flash.Level)
		}
		if flash.Message != "Something failed" {
			t.Errorf("message = %q, want 'Something failed'", flash.Message)
		}

		// Verify cookie was cleared
		clearCookies := getRec.Result().Cookies()
		found := false
		for _, c := range clearCookies {
			if c.Name == "_flash" && c.MaxAge == -1 {
				found = true
			}
		}
		if !found {
			t.Error("expected flash cookie to be cleared with MaxAge=-1")
		}
	})

	t.Run("nil when no cookie", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		flash := web.GetFlash(rec, r)
		if flash != nil {
			t.Errorf("expected nil, got %+v", flash)
		}
	})

	t.Run("corrupted base64 returns nil", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/", nil)
		r.AddCookie(&http.Cookie{Name: "_flash", Value: "!!!not-base64!!!"})
		rec := httptest.NewRecorder()
		flash := web.GetFlash(rec, r)
		if flash != nil {
			t.Errorf("expected nil for corrupted base64, got %+v", flash)
		}
	})

	t.Run("corrupted JSON returns nil", func(t *testing.T) {
		t.Parallel()
		encoded := base64.RawURLEncoding.EncodeToString([]byte("{not json"))
		r := httptest.NewRequest("GET", "/", nil)
		r.AddCookie(&http.Cookie{Name: "_flash", Value: encoded})
		rec := httptest.NewRecorder()
		flash := web.GetFlash(rec, r)
		if flash != nil {
			t.Errorf("expected nil for corrupted JSON, got %+v", flash)
		}
	})
}

func TestFlashRoundTrip(t *testing.T) {
	t.Parallel()

	levels := []web.FlashLevel{web.FlashSuccess, web.FlashError, web.FlashWarning, web.FlashInfo}

	for _, level := range levels {
		t.Run(string(level), func(t *testing.T) {
			t.Parallel()
			rec := httptest.NewRecorder()
			web.SetFlash(rec, level, "test message")

			r := httptest.NewRequest("GET", "/", nil)
			for _, c := range rec.Result().Cookies() {
				r.AddCookie(c)
			}
			getRec := httptest.NewRecorder()
			flash := web.GetFlash(getRec, r)

			if flash == nil {
				t.Fatal("expected flash message")
			}
			if flash.Level != level {
				t.Errorf("level = %q, want %q", flash.Level, level)
			}
			if flash.Message != "test message" {
				t.Errorf("message = %q, want 'test message'", flash.Message)
			}
		})
	}
}

func TestFlashMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("injects flash into context", func(t *testing.T) {
		t.Parallel()
		// Set flash
		setRec := httptest.NewRecorder()
		web.SetFlash(setRec, web.FlashInfo, "Welcome back")

		// Build request with flash cookie
		r := httptest.NewRequest("GET", "/", nil)
		for _, c := range setRec.Result().Cookies() {
			r.AddCookie(c)
		}

		var gotFlash *web.FlashMessage
		handler := web.FlashMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotFlash = web.FlashFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		}))

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, r)

		if gotFlash == nil {
			t.Fatal("expected flash in context")
		}
		if gotFlash.Level != web.FlashInfo {
			t.Errorf("level = %q, want info", gotFlash.Level)
		}
		if gotFlash.Message != "Welcome back" {
			t.Errorf("message = %q, want 'Welcome back'", gotFlash.Message)
		}
	})

	t.Run("passes through when no flash", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/", nil)
		var gotFlash *web.FlashMessage
		handler := web.FlashMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotFlash = web.FlashFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		}))

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, r)

		if gotFlash != nil {
			t.Errorf("expected nil flash, got %+v", gotFlash)
		}
		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rec.Code)
		}
	})
}

func TestFlashFromContext(t *testing.T) {
	t.Parallel()

	t.Run("nil when missing", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		flash := web.FlashFromContext(ctx)
		if flash != nil {
			t.Errorf("expected nil, got %+v", flash)
		}
	})
}
