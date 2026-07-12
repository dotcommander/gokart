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

func TestSaveState_UsesPrivatePermissions(t *testing.T) {
	appName := "gokart-test-permissions-" + t.Name()
	filename := "state.json"
	path := gokart.StatePath(appName, filename)
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(path)) })

	if err := gokart.SaveState(appName, filename, testState{Name: "private"}); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("permissions = %o, want 600", got)
	}
}

func TestSaveState_FailedPublicationPreservesExistingState(t *testing.T) {
	appName := "gokart-test-atomic-" + t.Name()
	filename := "state.json"
	path := gokart.StatePath(appName, filename)
	dir := filepath.Dir(path)
	t.Cleanup(func() {
		_ = os.Chmod(dir, 0o755)
		_ = os.RemoveAll(dir)
	})

	if err := gokart.SaveState(appName, filename, testState{Name: "original", Count: 1}); err != nil {
		t.Fatalf("initial SaveState: %v", err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	if err := gokart.SaveState(appName, filename, testState{Name: "replacement", Count: 2}); err == nil {
		t.Skip("environment permits writes to a non-writable directory")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "{\n  \"name\": \"original\",\n  \"count\": 1\n}" {
		t.Fatalf("existing state changed after failed publication: %s", content)
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
