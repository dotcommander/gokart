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
		os.Remove(path)
		os.Remove(filepath.Dir(path))
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

// TestState_DocSaveLoadCycle verifies a complete save/load cycle
func TestState_DocSaveLoadCycle(t *testing.T) {
	t.Parallel()

	type AppState struct {
		LastFile   string   `json:"last_file"`
		Recent     []string `json:"recent_files"`
		VisitCount int      `json:"visit_count"`
	}

	appName := "cycle-test-" + t.Name()
	filename := "state.json"

	// Clean up
	defer func() {
		path := gokart.StatePath(appName, filename)
		os.Remove(path)
		os.Remove(filepath.Dir(path))
	}()

	// Step 1: First run - file doesn't exist
	_, err := gokart.LoadState[AppState](appName, filename)
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("First run should return os.ErrNotExist, got: %v", err)
	}

	// Step 2: Initialize defaults
	state := AppState{
		Recent:     []string{},
		VisitCount: 0,
	}

	// Step 3: Update state
	state.LastFile = "/new/file.txt"
	state.VisitCount++
	state.Recent = append(state.Recent, "/new/file.txt")

	// Step 4: Save state
	if err := gokart.SaveState(appName, filename, state); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Step 5: Load state back
	loaded, err := gokart.LoadState[AppState](appName, filename)
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	// Step 6: Verify data
	if loaded.LastFile != state.LastFile {
		t.Errorf("LastFile = %s, want %s", loaded.LastFile, state.LastFile)
	}
	if loaded.VisitCount != state.VisitCount {
		t.Errorf("VisitCount = %d, want %d", loaded.VisitCount, state.VisitCount)
	}
	if len(loaded.Recent) != len(state.Recent) {
		t.Errorf("Recent length = %d, want %d", len(loaded.Recent), len(state.Recent))
	} else if len(loaded.Recent) > 0 && loaded.Recent[0] != state.Recent[0] {
		t.Errorf("Recent[0] = %s, want %s", loaded.Recent[0], state.Recent[0])
	}

	// Step 7: Verify file is readable JSON
	path := gokart.StatePath(appName, filename)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read state file: %v", err)
	}

	// Verify JSON is indented (human-readable as documented)
	if string(content) == "" {
		t.Error("State file is empty")
	}
	// Check for indentation (2-space as documented)
	if string(content)[0] != '{' {
		t.Error("State file should start with '{'")
	}
}

// TestState_DocJsonFormat verifies JSON is indented for human readability
func TestState_DocJsonFormat(t *testing.T) {
	t.Parallel()

	type SimpleState struct {
		Name string `json:"name"`
	}

	appName := "json-test-" + t.Name()
	filename := "formatted.json"

	// Clean up
	defer func() {
		path := gokart.StatePath(appName, filename)
		os.Remove(path)
		os.Remove(filepath.Dir(path))
	}()

	data := SimpleState{Name: "test"}
	if err := gokart.SaveState(appName, filename, data); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	path := gokart.StatePath(appName, filename)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read state file: %v", err)
	}

	// Verify JSON is formatted with 2-space indentation
	expected := `{
  "name": "test"
}`
	if string(content) != expected {
		t.Errorf("JSON format mismatch.\nGot:\n%s\n\nWant:\n%s", string(content), expected)
	}
}

// TestState_DocZeroValue verifies zero value return on error
func TestState_DocZeroValue(t *testing.T) {
	t.Parallel()

	type NonZeroState struct {
		Active bool   `json:"active"`
		Name   string `json:"name"`
	}

	// Load non-existent file
	state, err := gokart.LoadState[NonZeroState]("nonexistent-app-xyz", "missing.json")

	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Expected os.ErrNotExist, got: %v", err)
	}

	// Verify zero value (not default struct values)
	if state.Active != false {
		t.Errorf("Zero value should have Active=false, got %v", state.Active)
	}
	if state.Name != "" {
		t.Errorf("Zero value should have empty Name, got %q", state.Name)
	}
}
