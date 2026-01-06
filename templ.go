package gokart

import (
	"context"
	"net/http"

	"github.com/a-h/templ"
)

// Render renders a templ component to an http.ResponseWriter.
//
// Sets Content-Type to text/html and handles errors.
//
// Example:
//
//	func handleHome(w http.ResponseWriter, r *http.Request) {
//	    gokart.Render(w, r, views.HomePage("Welcome"))
//	}
func Render(w http.ResponseWriter, r *http.Request, component templ.Component) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return component.Render(r.Context(), w)
}

// RenderWithStatus renders a templ component with a custom status code.
//
// Example:
//
//	func handleNotFound(w http.ResponseWriter, r *http.Request) {
//	    gokart.RenderWithStatus(w, r, http.StatusNotFound, views.NotFoundPage())
//	}
func RenderWithStatus(w http.ResponseWriter, r *http.Request, status int, component templ.Component) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	return component.Render(r.Context(), w)
}

// RenderCtx renders a templ component with a custom context.
//
// Example:
//
//	ctx := context.WithValue(r.Context(), "user", currentUser)
//	gokart.RenderCtx(ctx, w, views.Dashboard(data))
func RenderCtx(ctx context.Context, w http.ResponseWriter, component templ.Component) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return component.Render(ctx, w)
}

// TemplHandler creates an http.Handler from a templ component.
//
// Useful for static pages or when you don't need request data.
//
// Example:
//
//	router.Get("/about", gokart.TemplHandler(views.AboutPage()))
func TemplHandler(component templ.Component) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := component.Render(r.Context(), w); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})
}

// TemplHandlerFunc creates an http.HandlerFunc from a function that returns a component.
//
// Useful when the component needs data from the request.
//
// Example:
//
//	router.Get("/user/{id}", gokart.TemplHandlerFunc(func(r *http.Request) templ.Component {
//	    id := chi.URLParam(r, "id")
//	    user := getUser(id)
//	    return views.UserPage(user)
//	}))
func TemplHandlerFunc(fn func(r *http.Request) templ.Component) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		component := fn(r)
		if err := component.Render(r.Context(), w); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}

// TemplHandlerFuncE creates an http.HandlerFunc from a function that can return an error.
//
// Example:
//
//	router.Get("/dashboard", gokart.TemplHandlerFuncE(func(r *http.Request) (templ.Component, error) {
//	    data, err := loadDashboardData(r.Context())
//	    if err != nil {
//	        return nil, err
//	    }
//	    return views.Dashboard(data), nil
//	}))
func TemplHandlerFuncE(fn func(r *http.Request) (templ.Component, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		component, err := fn(r)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := component.Render(r.Context(), w); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}
