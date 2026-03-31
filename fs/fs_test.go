package fs_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/dotcommander/gokart/fs"
)

func TestWriteFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := []byte("hello world")

	if err := fs.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "hello world" {
		t.Fatalf("content mismatch: got %q, want %q", got, content)
	}
}

func TestWriteFile_CreatesDirs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "c", "test.txt")

	if err := fs.WriteFile(path, []byte("nested"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "nested" {
		t.Fatalf("content mismatch: got %q", got)
	}
}

func TestWriteFile_Atomic(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "atomic.txt")

	// First write
	if err := fs.WriteFile(path, []byte("v1"), 0o644); err != nil {
		t.Fatalf("WriteFile v1: %v", err)
	}

	// Overwrite
	if err := fs.WriteFile(path, []byte("v2"), 0o644); err != nil {
		t.Fatalf("WriteFile v2: %v", err)
	}

	// Verify no .tmp files left behind
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if slices.Contains([]string{".tmp"}, filepath.Ext(e.Name())) {
			t.Fatalf("temp file left behind: %s", e.Name())
		}
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "v2" {
		t.Fatalf("content mismatch: got %q, want %q", got, "v2")
	}
}

func TestReadOrCreate_Existing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "existing.txt")
	original := []byte("already here")

	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatalf("setup WriteFile: %v", err)
	}

	got, err := fs.ReadOrCreate(path, []byte("default"))
	if err != nil {
		t.Fatalf("ReadOrCreate: %v", err)
	}
	if string(got) != "already here" {
		t.Fatalf("should return existing content, got %q", got)
	}
}

func TestReadOrCreate_Creates(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "new.txt")
	defaultContent := []byte("default content")

	got, err := fs.ReadOrCreate(path, defaultContent)
	if err != nil {
		t.Fatalf("ReadOrCreate: %v", err)
	}
	if string(got) != "default content" {
		t.Fatalf("content mismatch: got %q, want %q", got, defaultContent)
	}

	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(onDisk) != "default content" {
		t.Fatalf("on-disk mismatch: got %q", onDisk)
	}
}

func TestConfigDir(t *testing.T) {
	t.Parallel()
	appName := "gokart-fs-test-9x7k"

	dir, err := fs.ConfigDir(appName)
	if err != nil {
		t.Fatalf("ConfigDir: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("expected directory, got %v", info.Mode())
	}

	if filepath.Base(dir) != appName {
		t.Fatalf("path should end with %q, got %q", appName, dir)
	}

	t.Cleanup(func() { os.RemoveAll(dir) })
}
