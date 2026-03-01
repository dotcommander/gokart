package web

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
)

// FlashLevel represents the severity of a flash message.
type FlashLevel string

const (
	FlashSuccess FlashLevel = "success"
	FlashError   FlashLevel = "error"
	FlashWarning FlashLevel = "warning"
	FlashInfo    FlashLevel = "info"
)

// FlashMessage is a one-time notification displayed after a redirect.
type FlashMessage struct {
	Level   FlashLevel `json:"level"`
	Message string     `json:"message"`
}

type flashContextKey struct{}

const flashCookieName = "_flash"

// SetFlash sets a flash message cookie that will be consumed on the next request.
//
// Example:
//
//	web.SetFlash(w, web.FlashSuccess, "Item created successfully")
//	http.Redirect(w, r, "/items", http.StatusSeeOther)
func SetFlash(w http.ResponseWriter, level FlashLevel, message string) {
	msg := FlashMessage{Level: level, Message: message}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	encoded := base64.RawURLEncoding.EncodeToString(data)
	http.SetCookie(w, &http.Cookie{
		Name:     flashCookieName,
		Value:    encoded,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// GetFlash reads and clears the flash message cookie.
// Returns nil if no flash cookie exists or if the cookie is corrupted.
//
// Example:
//
//	if flash := web.GetFlash(w, r); flash != nil {
//	    fmt.Printf("[%s] %s\n", flash.Level, flash.Message)
//	}
func GetFlash(w http.ResponseWriter, r *http.Request) *FlashMessage {
	cookie, err := r.Cookie(flashCookieName)
	if err != nil {
		return nil
	}

	// Clear the cookie immediately
	http.SetCookie(w, &http.Cookie{
		Name:     flashCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	data, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return nil
	}

	var msg FlashMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil
	}

	return &msg
}

// FlashMiddleware reads flash messages and injects them into the request context.
// Use FlashFromContext to retrieve the flash in handlers.
//
// Example:
//
//	r.Use(web.FlashMiddleware)
//	r.Get("/dashboard", func(w http.ResponseWriter, r *http.Request) {
//	    if flash := web.FlashFromContext(r.Context()); flash != nil {
//	        // render flash message
//	    }
//	})
func FlashMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flash := GetFlash(w, r)
		if flash != nil {
			ctx := context.WithValue(r.Context(), flashContextKey{}, flash)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

// FlashFromContext retrieves a flash message from the request context.
// Returns nil if no flash message is present.
func FlashFromContext(ctx context.Context) *FlashMessage {
	flash, _ := ctx.Value(flashContextKey{}).(*FlashMessage)
	return flash
}
