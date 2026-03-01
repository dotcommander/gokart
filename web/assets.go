package web

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
)

// AssetConfig configures static asset serving.
type AssetConfig struct {
	FS     fs.FS  // required: embedded filesystem
	Prefix string // URL path prefix, default "/assets"
	MaxAge int    // Cache-Control max-age seconds, default 31536000 (1 year)
}

// Assets serves static files with content-hash cache busting.
type Assets struct {
	fs           fs.FS
	prefix       string
	maxAge       int
	cacheControl string
	hashes       map[string]string // path -> first 12 hex chars of SHA256
	etags        map[string]string // path -> pre-quoted ETag value
}

// NewAssets creates an Assets server by walking the filesystem and computing
// content hashes for every file. Returns an error if cfg.FS is nil or if the
// filesystem walk fails.
func NewAssets(cfg AssetConfig) (*Assets, error) {
	if cfg.FS == nil {
		return nil, fmt.Errorf("web: AssetConfig.FS is required")
	}

	prefix := cfg.Prefix
	if prefix == "" {
		prefix = "/assets"
	}
	prefix = ensureLeadingSlash(prefix)
	prefix = strings.TrimRight(prefix, "/")

	maxAge := cfg.MaxAge
	if maxAge == 0 {
		maxAge = 31536000
	}

	hashes := make(map[string]string)
	err := fs.WalkDir(cfg.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		f, err := cfg.FS.Open(path)
		if err != nil {
			return fmt.Errorf("web: open %s: %w", path, err)
		}
		defer f.Close()

		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			return fmt.Errorf("web: hash %s: %w", path, err)
		}

		// Store with leading "/" so it matches request paths.
		hashes["/"+path] = hex.EncodeToString(h.Sum(nil)[:6])
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("web: walk FS: %w", err)
	}

	cacheControl := fmt.Sprintf("public, max-age=%d", maxAge)

	etags := make(map[string]string, len(hashes))
	for p, hash := range hashes {
		etags[p] = `"` + hash + `"`
	}

	return &Assets{
		fs:           cfg.FS,
		prefix:       prefix,
		maxAge:       maxAge,
		cacheControl: cacheControl,
		hashes:       hashes,
		etags:        etags,
	}, nil
}

// Path returns the versioned URL for the given asset path.
// If the asset is known, appends "?v=<hash>". Otherwise returns the bare URL.
// name may or may not have a leading slash.
func (a *Assets) Path(name string) string {
	name = ensureLeadingSlash(name)
	if hash, ok := a.hashes[name]; ok {
		return a.prefix + name + "?v=" + hash
	}
	return a.prefix + name
}

// Handler returns an http.Handler that serves the embedded filesystem.
// It handles ETag / If-None-Match negotiation and sets Cache-Control headers.
func (a *Assets) Handler() http.Handler {
	fileServer := http.FileServerFS(a.fs)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// r.URL.Path already has the prefix stripped; ensure leading slash.
		path := ensureLeadingSlash(r.URL.Path)

		if etag, ok := a.etags[path]; ok {
			w.Header().Set("ETag", etag)

			if r.Header.Get("If-None-Match") == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		w.Header().Set("Cache-Control", a.cacheControl)
		fileServer.ServeHTTP(w, r)
	})

	return http.StripPrefix(a.prefix, inner)
}

func ensureLeadingSlash(s string) string {
	if strings.HasPrefix(s, "/") {
		return s
	}
	return "/" + s
}
