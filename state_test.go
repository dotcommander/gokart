package gokart_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dotcommander/gokart"
)

type testState struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestSaveAndLoadState(t *testing.T) {
	t.Parallel()

	// Use a unique app name to avoid conflicts
	appName := "gokart-test-" + t.Name()
	filename := "state.json"

	// Clean up after test
	defer func() {
		path := gokart.StatePath(appName, filename)
		os.Remove(path)
		os.Remove(filepath.Dir(path))
	}()

	original := testState{
		Name:  "test",
		Count: 42,
	}

	// Save state
	if err := gokart.SaveState(appName, filename, original); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Load state back
	loaded, err := gokart.LoadState[testState](appName, filename)
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	if loaded.Name != original.Name {
		t.Errorf("Name mismatch: got %q, want %q", loaded.Name, original.Name)
	}
	if loaded.Count != original.Count {
		t.Errorf("Count mismatch: got %d, want %d", loaded.Count, original.Count)
	}
}

func TestLoadState_NotFound(t *testing.T) {
	t.Parallel()

	_, err := gokart.LoadState[testState]("nonexistent-app-xyz", "missing.json")
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected os.ErrNotExist, got %v", err)
	}
}

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
