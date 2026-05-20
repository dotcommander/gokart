package gokart_test

import (
	"errors"
	"os"
	"path/filepath"
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
		_ = os.Remove(path)
		_ = os.Remove(filepath.Dir(path))
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

// TestState_DocSignatures verifies SaveState and LoadState signatures are documented
func TestState_DocSignatures(t *testing.T) {
	t.Parallel()

	// This test verifies the signatures work as documented
	type TestState struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	appName := "test-signatures-" + t.Name()
	filename := "test.json"

	// Clean up
	defer func() {
		path := gokart.StatePath(appName, filename)
		_ = os.Remove(path)
		_ = os.Remove(filepath.Dir(path))
	}()

	// Test SaveState signature: func SaveState[T any](appName, filename string, data T) error
	err := gokart.SaveState(appName, filename, TestState{
		Name:  "test",
		Count: 42,
	})
	if err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Test LoadState signature: func LoadState[T any](appName, filename string) (T, error)
	state, err := gokart.LoadState[TestState](appName, filename)
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	if state.Name != "test" || state.Count != 42 {
		t.Errorf("LoadState returned unexpected data: got %+v, want Name=test, Count=42", state)
	}
}
