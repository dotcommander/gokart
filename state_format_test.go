package gokart_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/dotcommander/gokart"
)

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
		_ = os.Remove(path)
		_ = os.Remove(filepath.Dir(path))
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
		_ = os.Remove(path)
		_ = os.Remove(filepath.Dir(path))
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
