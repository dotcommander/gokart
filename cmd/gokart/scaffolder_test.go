package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestApplyDryRunDoesNotWriteFiles(t *testing.T) {
	fsys := fstest.MapFS{
		"templates/basic/main.go.tmpl": {Data: []byte("package main\n")},
	}

	targetDir := t.TempDir()
	result, err := Apply(fsys, "templates/basic", targetDir, TemplateData{}, ApplyOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if len(result.Created) != 1 || result.Created[0] != "main.go" {
		t.Fatalf("unexpected created result: %#v", result.Created)
	}

	if _, err := os.Stat(filepath.Join(targetDir, "main.go")); !os.IsNotExist(err) {
		t.Fatalf("expected file to not exist after dry run, stat err = %v", err)
	}
}

func TestApplyFailsWhenExistingFileConflicts(t *testing.T) {
	fsys := fstest.MapFS{
		"templates/basic/main.go.tmpl": {Data: []byte("package main\n")},
	}

	targetDir := t.TempDir()
	dest := filepath.Join(targetDir, "main.go")
	if err := os.WriteFile(dest, []byte("old\n"), 0644); err != nil {
		t.Fatalf("seed destination file: %v", err)
	}

	_, err := Apply(fsys, "templates/basic", targetDir, TemplateData{}, ApplyOptions{ExistingFilePolicy: ExistingFilePolicyFail})
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
}

func TestApplyReportsAllExistingFileConflicts(t *testing.T) {
	fsys := fstest.MapFS{
		"templates/basic/main.go.tmpl":   {Data: []byte("package main\n")},
		"templates/basic/README.md.tmpl": {Data: []byte("# demo\n")},
	}

	targetDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(targetDir, "main.go"), []byte("old\n"), 0644); err != nil {
		t.Fatalf("seed main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("old\n"), 0644); err != nil {
		t.Fatalf("seed README.md: %v", err)
	}

	_, err := Apply(fsys, "templates/basic", targetDir, TemplateData{}, ApplyOptions{ExistingFilePolicy: ExistingFilePolicyFail})
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}

	var conflictErr *ExistingFileConflictError
	if !errors.As(err, &conflictErr) {
		t.Fatalf("expected ExistingFileConflictError, got %T (%v)", err, err)
	}

	if len(conflictErr.Paths) != 2 {
		t.Fatalf("expected 2 conflicts, got %d (%v)", len(conflictErr.Paths), conflictErr.Paths)
	}

	conflicts := map[string]bool{}
	for _, path := range conflictErr.Paths {
		conflicts[path] = true
	}

	if !conflicts["main.go"] || !conflicts["README.md"] {
		t.Fatalf("missing expected conflicts: %v", conflictErr.Paths)
	}
}

func TestApplySkipsExistingWithSkipPolicy(t *testing.T) {
	fsys := fstest.MapFS{
		"templates/basic/main.go.tmpl": {Data: []byte("package main\n")},
	}

	targetDir := t.TempDir()
	dest := filepath.Join(targetDir, "main.go")
	if err := os.WriteFile(dest, []byte("old\n"), 0644); err != nil {
		t.Fatalf("seed destination file: %v", err)
	}

	result, err := Apply(fsys, "templates/basic", targetDir, TemplateData{}, ApplyOptions{ExistingFilePolicy: ExistingFilePolicySkip})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if len(result.Skipped) != 1 || result.Skipped[0] != "main.go" {
		t.Fatalf("unexpected skipped result: %#v", result.Skipped)
	}

	content, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read destination file: %v", err)
	}
	if string(content) != "old\n" {
		t.Fatalf("destination file changed unexpectedly: %q", string(content))
	}
}

func TestApplyOverwritesExistingWithForcePolicy(t *testing.T) {
	fsys := fstest.MapFS{
		"templates/basic/main.go.tmpl": {Data: []byte("package main\n")},
	}

	targetDir := t.TempDir()
	dest := filepath.Join(targetDir, "main.go")
	if err := os.WriteFile(dest, []byte("old\n"), 0644); err != nil {
		t.Fatalf("seed destination file: %v", err)
	}

	result, err := Apply(fsys, "templates/basic", targetDir, TemplateData{}, ApplyOptions{ExistingFilePolicy: ExistingFilePolicyOverwrite})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if len(result.Overwritten) != 1 || result.Overwritten[0] != "main.go" {
		t.Fatalf("unexpected overwritten result: %#v", result.Overwritten)
	}

	content, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read destination file: %v", err)
	}
	if string(content) != "package main\n" {
		t.Fatalf("destination file not overwritten: %q", string(content))
	}
}

func TestApplyMarksUnchangedFiles(t *testing.T) {
	fsys := fstest.MapFS{
		"templates/basic/main.go.tmpl": {Data: []byte("package main\n")},
	}

	targetDir := t.TempDir()
	dest := filepath.Join(targetDir, "main.go")
	if err := os.WriteFile(dest, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("seed destination file: %v", err)
	}

	result, err := Apply(fsys, "templates/basic", targetDir, TemplateData{}, ApplyOptions{ExistingFilePolicy: ExistingFilePolicyFail})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if len(result.Unchanged) != 1 || result.Unchanged[0] != "main.go" {
		t.Fatalf("unexpected unchanged result: %#v", result.Unchanged)
	}
}
