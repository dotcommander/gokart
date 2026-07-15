package gokart_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPhilosophyBoundary(t *testing.T) {
	t.Parallel()

	for _, retired := range []string{"ai/go.mod", "fs/go.mod"} {
		if _, err := os.Stat(retired); !os.IsNotExist(err) {
			t.Fatalf("retired module file %q still exists", retired)
		}
	}

	for _, path := range []string{"go.work", "justfile", "README.md", "doc.go"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		for _, retired := range []string{"./ai", "./fs", "gokart/ai", "gokart/fs"} {
			if strings.Contains(text, retired) {
				t.Fatalf("%s contains retired surface %q", path, retired)
			}
		}
	}

	err := filepath.WalkDir("cmd/gokart/internal/generator/templates", func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(data), "@latest") {
			t.Errorf("generated template %s contains @latest", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestWebSourceAllowlist(t *testing.T) {
	t.Parallel()
	want := map[string]bool{
		"bind.go": true, "bind_test.go": true,
		"httpserver.go": true, "httpserver_test.go": true,
		"response.go": true,
		"validate.go": true,
		"web_test.go": true,
	}
	entries, err := os.ReadDir("web")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".go" && !want[entry.Name()] {
			t.Errorf("web API surface expanded with %s; update the philosophy review explicitly", entry.Name())
		}
	}
}

func TestMigrationInventoryNamesEveryRemovedAPI(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)
	removed := []string{
		"GetString", "GetInt", "GetFloat", "GetBool", "SetOutput", "SetErrOutput", "Output", "ErrOutput", "Fatal", "FatalErr", "Must",
		"Get", "Set", "Delete", "Exists", "Expire", "TTL", "Incr", "IncrBy", "SetNX", "HGet", "HSet", "HGetAll", "HDel", "HIncrBy", "ZAdd", "ZRange", "ZRangeByScore", "ZScore", "ZRem", "ZCard", "SAdd", "SRem", "SMembers", "SIsMember", "LPush", "RPush", "LRange", "LPop", "RPop", "Decr", "DecrBy",
		"NewAssets", "Assets.Path", "Assets.Handler", "APIKeyAuth", "BearerAuth", "CSRFProtect", "SetFlash", "GetFlash", "FlashSuccess", "FlashError", "FlashWarning", "FlashInfo", "HealthHandler", "ReadyHandler", "NewHTTPClient", "NewStandardClient", "WantsJSON", "IsHTMX", "Negotiate", "ParsePage", "NewPagedResponse", "RateLimit", "RateLimiter.Middleware", "RateLimiter.LimiterCount", "RateLimiter.Stop", "Render", "TemplHandler",
	}
	for _, symbol := range removed {
		if !strings.Contains(doc, "`"+symbol+"`") {
			t.Errorf("migration inventory omits %s", symbol)
		}
	}
}
