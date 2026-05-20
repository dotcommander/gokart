package gokart_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dotcommander/gokart"
)

func TestStatePath(t *testing.T) {
	t.Parallel()

	path := gokart.StatePath("myapp", "state.json")
	if path == "" {
		t.Fatal("StatePath returned empty string")
	}

	// Should contain the app name
	if !strings.Contains(path, "myapp") {
		t.Errorf("path should contain app name: %s", path)
	}

	// Should end with filename
	if filepath.Base(path) != "state.json" {
		t.Errorf("path should end with filename: %s", path)
	}
}

// TestState_DocPlatformPaths verifies platform-specific paths are explained
func TestState_DocPlatformPaths(t *testing.T) {
	t.Parallel()

	// This test verifies StatePath returns platform-specific paths
	tests := []struct {
		appName  string
		filename string
	}{
		{"myapp", "state.json"},
		{"cli-tool", "preferences.json"},
		{"test-app", "cache.json"},
	}

	for _, tt := range tests {
		t.Run(tt.appName+"_"+tt.filename, func(t *testing.T) {
			t.Parallel()

			path := gokart.StatePath(tt.appName, tt.filename)

			// Verify path is not empty
			if path == "" {
				t.Error("StatePath returned empty string")
			}

			// Verify path uses platform-specific config directory
			configDir, err := os.UserConfigDir()
			if err != nil {
				t.Fatalf("Failed to get user config dir: %v", err)
			}

			expectedDir := filepath.Join(configDir, tt.appName)
			if filepath.Dir(path) != expectedDir {
				t.Errorf("StatePath path dir = %s, want %s", filepath.Dir(path), expectedDir)
			}

			// Verify filename matches
			if filepath.Base(path) != tt.filename {
				t.Errorf("StatePath filename = %s, want %s", filepath.Base(path), tt.filename)
			}

			// Verify path is absolute
			if !filepath.IsAbs(path) {
				t.Error("StatePath should return absolute path")
			}
		})
	}
}
