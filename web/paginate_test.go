package web_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotcommander/gokart/web"
)

func TestParsePage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		query       string
		wantNumber  int
		wantPerPage int
		wantOffset  int
	}{
		{
			name:        "defaults no params",
			query:       "/items",
			wantNumber:  1,
			wantPerPage: 20,
			wantOffset:  0,
		},
		{
			name:        "custom page and per_page",
			query:       "/items?page=3&per_page=25",
			wantNumber:  3,
			wantPerPage: 25,
			wantOffset:  50,
		},
		{
			name:        "page zero clamps to 1",
			query:       "/items?page=0",
			wantNumber:  1,
			wantPerPage: 20,
			wantOffset:  0,
		},
		{
			name:        "negative page clamps to 1",
			query:       "/items?page=-5",
			wantNumber:  1,
			wantPerPage: 20,
			wantOffset:  0,
		},
		{
			name:        "per_page over max clamps to 100",
			query:       "/items?per_page=500",
			wantNumber:  1,
			wantPerPage: 100,
			wantOffset:  0,
		},
		{
			name:        "per_page zero uses default",
			query:       "/items?per_page=0",
			wantNumber:  1,
			wantPerPage: 20,
			wantOffset:  0,
		},
		{
			name:        "non-numeric page uses default",
			query:       "/items?page=abc",
			wantNumber:  1,
			wantPerPage: 20,
			wantOffset:  0,
		},
		{
			name:        "non-numeric per_page uses default",
			query:       "/items?per_page=xyz",
			wantNumber:  1,
			wantPerPage: 20,
			wantOffset:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest("GET", tt.query, nil)
			p := web.ParsePage(r)
			if p.Number != tt.wantNumber {
				t.Errorf("Number = %d, want %d", p.Number, tt.wantNumber)
			}
			if p.PerPage != tt.wantPerPage {
				t.Errorf("PerPage = %d, want %d", p.PerPage, tt.wantPerPage)
			}
			if p.Offset != tt.wantOffset {
				t.Errorf("Offset = %d, want %d", p.Offset, tt.wantOffset)
			}
		})
	}
}

func TestParsePageWithConfig(t *testing.T) {
	t.Parallel()

	cfg := web.PageConfig{DefaultPerPage: 50, MaxPerPage: 200}

	t.Run("uses custom defaults", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/items", nil)
		p := web.ParsePageWithConfig(r, cfg)
		if p.PerPage != 50 {
			t.Errorf("PerPage = %d, want 50", p.PerPage)
		}
	})

	t.Run("clamps to custom max", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("GET", "/items?per_page=300", nil)
		p := web.ParsePageWithConfig(r, cfg)
		if p.PerPage != 200 {
			t.Errorf("PerPage = %d, want 200", p.PerPage)
		}
	})
}

func TestNewPagedResponse(t *testing.T) {
	t.Parallel()

	t.Run("exact pages", func(t *testing.T) {
		t.Parallel()
		page := web.Page{Number: 1, PerPage: 10, Offset: 0}
		data := make([]string, 10)
		resp := web.NewPagedResponse(data, page, 30)
		if resp.TotalPages != 3 {
			t.Errorf("TotalPages = %d, want 3", resp.TotalPages)
		}
		if resp.Total != 30 {
			t.Errorf("Total = %d, want 30", resp.Total)
		}
		if resp.Page != 1 {
			t.Errorf("Page = %d, want 1", resp.Page)
		}
	})

	t.Run("partial last page", func(t *testing.T) {
		t.Parallel()
		page := web.Page{Number: 2, PerPage: 10, Offset: 10}
		data := make([]string, 5)
		resp := web.NewPagedResponse(data, page, 15)
		if resp.TotalPages != 2 {
			t.Errorf("TotalPages = %d, want 2", resp.TotalPages)
		}
	})

	t.Run("zero total", func(t *testing.T) {
		t.Parallel()
		page := web.Page{Number: 1, PerPage: 10, Offset: 0}
		resp := web.NewPagedResponse([]string{}, page, 0)
		if resp.TotalPages != 0 {
			t.Errorf("TotalPages = %d, want 0", resp.TotalPages)
		}
		if resp.Total != 0 {
			t.Errorf("Total = %d, want 0", resp.Total)
		}
	})

	t.Run("nil data becomes empty slice in JSON", func(t *testing.T) {
		t.Parallel()
		page := web.Page{Number: 1, PerPage: 10, Offset: 0}
		resp := web.NewPagedResponse[string](nil, page, 0)
		b, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(b, &raw); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if string(raw["data"]) != "[]" {
			t.Errorf("data = %s, want []", string(raw["data"]))
		}
	})
}
