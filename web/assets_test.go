package web_test

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/dotcommander/gokart/web"
)

func testFS() fs.FS {
	return fstest.MapFS{
		"css/app.css": &fstest.MapFile{Data: []byte("body{color:red}")},
		"js/main.js":  &fstest.MapFile{Data: []byte("console.log('hello')")},
	}
}

func TestNewAssets(t *testing.T) {
	t.Parallel()

	t.Run("defaults", func(t *testing.T) {
		t.Parallel()

		a, err := web.NewAssets(web.AssetConfig{FS: testFS()})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify defaults via Path output (prefix should be /assets).
		p := a.Path("css/app.css")
		if !strings.HasPrefix(p, "/assets/") {
			t.Errorf("expected path prefix /assets/, got %q", p)
		}

		// max-age default verified via Handler test; here we just confirm no error.
	})

	t.Run("custom config", func(t *testing.T) {
		t.Parallel()

		a, err := web.NewAssets(web.AssetConfig{
			FS:     testFS(),
			Prefix: "/static",
			MaxAge: 3600,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		p := a.Path("css/app.css")
		if !strings.HasPrefix(p, "/static/") {
			t.Errorf("expected path prefix /static/, got %q", p)
		}
	})

	t.Run("nil FS returns error", func(t *testing.T) {
		t.Parallel()

		_, err := web.NewAssets(web.AssetConfig{})
		if err == nil {
			t.Fatal("expected error for nil FS, got nil")
		}
	})
}

func TestAssetsPath(t *testing.T) {
	t.Parallel()

	a, err := web.NewAssets(web.AssetConfig{FS: testFS()})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	tests := []struct {
		name       string
		input      string
		wantHash   bool
		wantPrefix string
	}{
		{
			name:       "known file with leading slash",
			input:      "/css/app.css",
			wantHash:   true,
			wantPrefix: "/assets/css/app.css?v=",
		},
		{
			name:       "known file without leading slash",
			input:      "js/main.js",
			wantHash:   true,
			wantPrefix: "/assets/js/main.js?v=",
		},
		{
			name:       "unknown file no hash",
			input:      "images/logo.png",
			wantHash:   false,
			wantPrefix: "/assets/images/logo.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := a.Path(tt.input)

			if !strings.HasPrefix(p, tt.wantPrefix) {
				t.Errorf("Path(%q) = %q, want prefix %q", tt.input, p, tt.wantPrefix)
			}

			if tt.wantHash {
				_, hash, ok := strings.Cut(p, "?v=")
				if !ok {
					t.Fatalf("Path(%q) = %q: missing ?v= param", tt.input, p)
				}
				if len(hash) != 12 {
					t.Errorf("hash length = %d, want 12; hash=%q", len(hash), hash)
				}
				for _, c := range hash {
					if !strings.ContainsRune("0123456789abcdef", c) {
						t.Errorf("hash %q contains non-hex character %q", hash, c)
						break
					}
				}

				// Deterministic: second call returns same value.
				if p2 := a.Path(tt.input); p != p2 {
					t.Errorf("Path not deterministic: %q != %q", p, p2)
				}
			} else {
				if strings.Contains(p, "?v=") {
					t.Errorf("Path(%q) = %q: unknown file should not have hash", tt.input, p)
				}
			}
		})
	}
}

func TestAssetsPath_CustomPrefix(t *testing.T) {
	t.Parallel()

	a, err := web.NewAssets(web.AssetConfig{
		FS:     testFS(),
		Prefix: "/static",
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	p := a.Path("css/app.css")
	if !strings.HasPrefix(p, "/static/css/app.css") {
		t.Errorf("expected /static/css/app.css prefix, got %q", p)
	}
}

func TestAssetsHandler(t *testing.T) {
	t.Parallel()

	a, err := web.NewAssets(web.AssetConfig{FS: testFS()})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	srv := httptest.NewServer(a.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/assets/css/app.css")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	if cc := resp.Header.Get("Cache-Control"); cc == "" {
		t.Error("Cache-Control header missing")
	}

	if etag := resp.Header.Get("ETag"); etag == "" {
		t.Error("ETag header missing")
	}
}

func TestAssetsHandler_CacheHeaders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		maxAge  int
		wantCC  string
	}{
		{
			name:   "default max-age",
			maxAge: 0, // triggers default of 31536000
			wantCC: "public, max-age=31536000",
		},
		{
			name:   "custom max-age",
			maxAge: 3600,
			wantCC: "public, max-age=3600",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a, err := web.NewAssets(web.AssetConfig{
				FS:     testFS(),
				MaxAge: tt.maxAge,
			})
			if err != nil {
				t.Fatalf("setup: %v", err)
			}

			srv := httptest.NewServer(a.Handler())
			defer srv.Close()

			resp, err := http.Get(srv.URL + "/assets/css/app.css")
			if err != nil {
				t.Fatalf("GET: %v", err)
			}
			defer resp.Body.Close()

			cc := resp.Header.Get("Cache-Control")
			if cc != tt.wantCC {
				t.Errorf("Cache-Control = %q, want %q", cc, tt.wantCC)
			}
		})
	}
}

func TestAssetsHandler_ETag304(t *testing.T) {
	t.Parallel()

	a, err := web.NewAssets(web.AssetConfig{FS: testFS()})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	srv := httptest.NewServer(a.Handler())
	defer srv.Close()

	// First request to get the ETag.
	resp, err := http.Get(srv.URL + "/assets/js/main.js")
	if err != nil {
		t.Fatalf("initial GET: %v", err)
	}
	resp.Body.Close()

	etag := resp.Header.Get("ETag")
	if etag == "" {
		t.Fatal("no ETag in initial response")
	}

	// Second request with If-None-Match should return 304.
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/assets/js/main.js", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("If-None-Match", etag)

	client := &http.Client{}
	resp2, err := client.Do(req)
	if err != nil {
		t.Fatalf("conditional GET: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusNotModified {
		t.Errorf("status = %d, want 304", resp2.StatusCode)
	}

}
