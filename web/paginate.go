package web

import (
	"net/http"
	"strconv"
)

// PageConfig configures pagination defaults.
type PageConfig struct {
	// DefaultPerPage is the default number of items per page.
	// Default: 20
	DefaultPerPage int

	// MaxPerPage is the maximum allowed items per page.
	// Default: 100
	MaxPerPage int
}

// Page represents parsed pagination parameters.
type Page struct {
	Number  int // 1-based page number
	PerPage int // items per page
	Offset  int // (Number-1) * PerPage, for SQL OFFSET
}

// ParsePage extracts pagination from query params with default config.
//
// Reads "page" and "per_page" query parameters. Invalid or missing values
// use defaults. Never returns an error — pagination is best-effort.
//
// Example:
//
//	// GET /users?page=2&per_page=25
//	p := web.ParsePage(r)
//	// p.Number=2, p.PerPage=25, p.Offset=25
func ParsePage(r *http.Request) Page {
	return ParsePageWithConfig(r, PageConfig{})
}

// ParsePageWithConfig extracts pagination with custom configuration.
//
// Example:
//
//	cfg := web.PageConfig{DefaultPerPage: 50, MaxPerPage: 200}
//	p := web.ParsePageWithConfig(r, cfg)
func ParsePageWithConfig(r *http.Request, cfg PageConfig) Page {
	if cfg.DefaultPerPage <= 0 {
		cfg.DefaultPerPage = 20
	}
	if cfg.MaxPerPage <= 0 {
		cfg.MaxPerPage = 100
	}

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	perPage, err := strconv.Atoi(r.URL.Query().Get("per_page"))
	if err != nil || perPage < 1 {
		perPage = cfg.DefaultPerPage
	}
	if perPage > cfg.MaxPerPage {
		perPage = cfg.MaxPerPage
	}

	return Page{
		Number:  page,
		PerPage: perPage,
		Offset:  (page - 1) * perPage,
	}
}

// PagedResponse is a generic paginated response envelope.
//
// Example:
//
//	users, total := fetchUsers(p.Offset, p.PerPage)
//	resp := web.NewPagedResponse(users, p, total)
//	web.JSON(w, resp)
type PagedResponse[T any] struct {
	Data       []T `json:"data"`
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// NewPagedResponse creates a paginated response envelope.
// Nil data is replaced with an empty slice so JSON serializes as [] not null.
func NewPagedResponse[T any](data []T, page Page, total int) PagedResponse[T] {
	if data == nil {
		data = []T{}
	}

	totalPages := 0
	if page.PerPage > 0 {
		totalPages = (total + page.PerPage - 1) / page.PerPage
	}

	return PagedResponse[T]{
		Data:       data,
		Page:       page.Number,
		PerPage:    page.PerPage,
		Total:      total,
		TotalPages: totalPages,
	}
}
