package generator

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func TestApplyRejectsSymlinkDestination(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"templates/basic/main.go.tmpl": {Data: []byte("package main\n")},
	}

	targetDir := t.TempDir()
	outsideFile := filepath.Join(t.TempDir(), "outside.go")
	if err := os.WriteFile(outsideFile, []byte("package outside\n"), 0644); err != nil {
		t.Fatalf("seed outside file: %v", err)
	}

	symlinkPath := filepath.Join(targetDir, "main.go")
	if err := os.Symlink(outsideFile, symlinkPath); err != nil {
		t.Skipf("symlink creation not supported in this environment: %v", err)
	}

	_, err := Apply(fsys, "templates/basic", targetDir, TemplateData{}, ApplyOptions{ExistingFilePolicy: ExistingFilePolicyOverwrite})
	if err == nil {
		t.Fatal("expected symlink destination error, got nil")
	}

	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink error, got: %v", err)
	}
}

func TestApplyRejectsSymlinkParentPath(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"templates/basic/nested/main.go.tmpl": {Data: []byte("package main\n")},
	}

	targetDir := t.TempDir()
	outsideDir := t.TempDir()
	symlinkParent := filepath.Join(targetDir, "nested")
	if err := os.Symlink(outsideDir, symlinkParent); err != nil {
		t.Skipf("symlink creation not supported in this environment: %v", err)
	}

	_, err := Apply(fsys, "templates/basic", targetDir, TemplateData{}, ApplyOptions{DryRun: true})
	if err == nil {
		t.Fatal("expected symlink parent error, got nil")
	}

	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink parent error, got: %v", err)
	}
}

func TestApplyFailsWhenLockFileExists(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"templates/basic/main.go.tmpl": {Data: []byte("package main\n")},
	}

	targetDir := t.TempDir()
	lockPath := filepath.Join(targetDir, ".gokart.lock")
	if err := os.WriteFile(lockPath, []byte(fmt.Sprintf("pid=%d\n", os.Getpid())), 0600); err != nil {
		t.Fatalf("seed lock file: %v", err)
	}

	_, err := Apply(fsys, "templates/basic", targetDir, TemplateData{}, ApplyOptions{ExistingFilePolicy: ExistingFilePolicyOverwrite})
	if err == nil {
		t.Fatal("expected lock contention error, got nil")
	}

	if !strings.Contains(err.Error(), "already running") {
		t.Fatalf("expected lock contention error, got: %v", err)
	}

	var lockErr *ApplyLockError
	if !errors.As(err, &lockErr) {
		t.Fatalf("expected ApplyLockError, got %T (%v)", err, err)
	}
}

func TestApplyReclaimsStaleLockFile(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"templates/basic/main.go.tmpl": {Data: []byte("package main\n")},
	}

	targetDir := t.TempDir()
	lockPath := filepath.Join(targetDir, ".gokart.lock")
	if err := os.WriteFile(lockPath, []byte("pid=999999\n"), 0600); err != nil {
		t.Fatalf("seed stale lock file: %v", err)
	}

	result, err := Apply(fsys, "templates/basic", targetDir, TemplateData{}, ApplyOptions{ExistingFilePolicy: ExistingFilePolicyOverwrite})
	if err != nil {
		t.Fatalf("Apply() should reclaim stale lock, got error: %v", err)
	}

	if len(result.Created) != 1 || result.Created[0] != "main.go" {
		t.Fatalf("unexpected created result: %#v", result.Created)
	}
}

func TestShouldRetainOldLockWhenPIDIsRunning(t *testing.T) {
	t.Parallel()
	lockPath := filepath.Join(t.TempDir(), ".gokart.lock")
	metadata := applyLockMetadata{
		PID:        os.Getpid(),
		CreatedAt:  time.Now().Add(-10 * time.Second),
		StaleAfter: 1,
	}

	if err := os.WriteFile(lockPath, metadata.encode(), 0600); err != nil {
		t.Fatalf("seed lock metadata: %v", err)
	}

	old := time.Now().Add(-5 * time.Second)
	if err := os.Chtimes(lockPath, old, old); err != nil {
		t.Fatalf("set stale lock mtime: %v", err)
	}

	isStale, reason, err := shouldReclaimStaleLock(lockPath)
	if err != nil {
		t.Fatalf("shouldReclaimStaleLock() error = %v", err)
	}
	if isStale {
		t.Fatalf("expected live owner lock to be retained, got isStale=%v reason=%q", isStale, reason)
	}
}

func TestProcessIsRunningRecognizesCurrentAndMissingPID(t *testing.T) {
	t.Parallel()

	running, err := processIsRunning(os.Getpid())
	if err != nil {
		t.Fatalf("check current pid: %v", err)
	}
	if !running {
		t.Fatal("current process reported as not running")
	}

	running, err = processIsRunning(999999)
	if err != nil {
		t.Fatalf("check missing pid: %v", err)
	}
	if running {
		t.Fatal("missing process reported as running")
	}
}
