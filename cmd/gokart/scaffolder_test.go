package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
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

func TestApplyRejectsSymlinkDestination(t *testing.T) {
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

func TestShouldReclaimStaleLockWhenPIDRunningButLockExpired(t *testing.T) {
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
	if !isStale {
		t.Fatalf("expected stale lock reclaim, got isStale=%v reason=%q", isStale, reason)
	}
	if !strings.Contains(reason, "older than") {
		t.Fatalf("unexpected stale reason: %q", reason)
	}
}

func TestApplyFinalizesJournalAfterInProcessRollback(t *testing.T) {
	failingFS := fstest.MapFS{
		"templates/basic/a.tmpl":       {Data: []byte("file-a\n")},
		"templates/basic/a/b.txt.tmpl": {Data: []byte("file-b\n")},
	}

	targetDir := t.TempDir()
	if _, err := Apply(failingFS, "templates/basic", targetDir, TemplateData{}, ApplyOptions{ExistingFilePolicy: ExistingFilePolicyOverwrite}); err == nil {
		t.Fatal("expected Apply() to fail for conflicting file/directory writes")
	}

	if _, err := os.Stat(filepath.Join(targetDir, "a", "b.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected rolled-back file to be absent, stat err = %v", err)
	}

	txDir := filepath.Join(targetDir, ".gokart", "tx")
	entries, err := os.ReadDir(txDir)
	if err == nil && len(entries) > 0 {
		t.Fatalf("expected no pending journal entries, found %d", len(entries))
	}
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("read tx directory: %v", err)
	}

	healthyFS := fstest.MapFS{
		"templates/basic/main.go.tmpl": {Data: []byte("package main\n")},
	}

	result, err := Apply(healthyFS, "templates/basic", targetDir, TemplateData{}, ApplyOptions{ExistingFilePolicy: ExistingFilePolicyOverwrite})
	if err != nil {
		t.Fatalf("Apply() after rollback should succeed, got error: %v", err)
	}
	if len(result.Created) != 1 || result.Created[0] != "main.go" {
		t.Fatalf("unexpected created result after rollback recovery: %#v", result.Created)
	}
}

func TestRecoverPendingJournalsRestoresIncompleteTransaction(t *testing.T) {
	targetDir := t.TempDir()
	targetRoot, err := filepath.Abs(targetDir)
	if err != nil {
		t.Fatalf("resolve target root: %v", err)
	}

	mainPath := filepath.Join(targetDir, "main.go")
	originalMain := []byte("package original\n")
	if err := os.WriteFile(mainPath, originalMain, 0644); err != nil {
		t.Fatalf("seed main.go: %v", err)
	}

	journal, err := beginApplyJournal(targetRoot)
	if err != nil {
		t.Fatalf("beginApplyJournal() error = %v", err)
	}

	mainRollbackIndex, err := journal.appendAction(rollbackAction{
		Kind:           rollbackRestore,
		Root:           targetRoot,
		Path:           mainPath,
		Content:        originalMain,
		Mode:           0644,
		ExpectedExists: true,
		ExpectedHash:   sha256Hex([]byte("package mutated\n")),
		ExpectedSize:   int64(len([]byte("package mutated\n"))),
		ExpectedMode:   0644,
	})
	if err != nil {
		t.Fatalf("append overwrite rollback action: %v", err)
	}

	if err := os.WriteFile(mainPath, []byte("package mutated\n"), 0644); err != nil {
		t.Fatalf("mutate main.go: %v", err)
	}

	if err := journal.markActionApplied(mainRollbackIndex); err != nil {
		t.Fatalf("mark overwrite rollback action applied: %v", err)
	}

	readmePath := filepath.Join(targetDir, "README.md")
	readmeRollbackIndex, err := journal.appendAction(rollbackAction{
		Kind:           rollbackRemove,
		Root:           targetRoot,
		Path:           readmePath,
		ExpectedExists: true,
		ExpectedHash:   sha256Hex([]byte("# generated\n")),
		ExpectedSize:   int64(len([]byte("# generated\n"))),
		ExpectedMode:   0644,
	})
	if err != nil {
		t.Fatalf("append create rollback action: %v", err)
	}

	if err := os.WriteFile(readmePath, []byte("# generated\n"), 0644); err != nil {
		t.Fatalf("create README.md: %v", err)
	}

	if err := journal.markActionApplied(readmeRollbackIndex); err != nil {
		t.Fatalf("mark create rollback action applied: %v", err)
	}

	if err := recoverPendingJournals(targetRoot); err != nil {
		t.Fatalf("recoverPendingJournals() error = %v", err)
	}

	mainContent, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("read recovered main.go: %v", err)
	}
	if string(mainContent) != string(originalMain) {
		t.Fatalf("unexpected recovered main.go content: %q", string(mainContent))
	}

	if _, err := os.Stat(readmePath); !os.IsNotExist(err) {
		t.Fatalf("expected README.md rollback removal, stat err = %v", err)
	}

	if _, err := os.Stat(journal.Path); !os.IsNotExist(err) {
		t.Fatalf("expected journal cleanup, stat err = %v", err)
	}
}

func TestRecoverPendingJournalsBlocksWhenGeneratedFileChanged(t *testing.T) {
	targetDir := t.TempDir()
	targetRoot, err := filepath.Abs(targetDir)
	if err != nil {
		t.Fatalf("resolve target root: %v", err)
	}

	generatedPath := filepath.Join(targetDir, "main.go")
	if err := os.WriteFile(generatedPath, []byte("package useredit\n"), 0644); err != nil {
		t.Fatalf("seed user-edited generated file: %v", err)
	}

	journal, err := beginApplyJournal(targetRoot)
	if err != nil {
		t.Fatalf("beginApplyJournal() error = %v", err)
	}

	rollbackIndex, err := journal.appendAction(rollbackAction{
		Kind:           rollbackRestore,
		Root:           targetRoot,
		Path:           generatedPath,
		Content:        []byte("package original\n"),
		Mode:           0644,
		ExpectedExists: true,
		ExpectedHash:   sha256Hex([]byte("package generated\n")),
		ExpectedSize:   int64(len([]byte("package generated\n"))),
		ExpectedMode:   0644,
	})
	if err != nil {
		t.Fatalf("append rollback action: %v", err)
	}

	if err := journal.markActionApplied(rollbackIndex); err != nil {
		t.Fatalf("mark rollback action applied: %v", err)
	}

	err = recoverPendingJournals(targetRoot)
	if err == nil {
		t.Fatal("expected recovery guard to block when generated file changed")
	}

	if !strings.Contains(err.Error(), "recovery blocked") {
		t.Fatalf("expected blocked recovery error, got: %v", err)
	}

	content, readErr := os.ReadFile(generatedPath)
	if readErr != nil {
		t.Fatalf("read generated file: %v", readErr)
	}
	if string(content) != "package useredit\n" {
		t.Fatalf("expected file to remain unchanged when recovery blocks, got: %q", string(content))
	}
}

func TestApplyRecoversPendingJournalBeforeScaffolding(t *testing.T) {
	fsys := fstest.MapFS{
		"templates/basic/main.go.tmpl": {Data: []byte("package main\n")},
	}

	targetDir := t.TempDir()
	targetRoot, err := filepath.Abs(targetDir)
	if err != nil {
		t.Fatalf("resolve target root: %v", err)
	}

	stalePath := filepath.Join(targetDir, "stale.txt")
	if err := os.WriteFile(stalePath, []byte("stale\n"), 0644); err != nil {
		t.Fatalf("seed stale file: %v", err)
	}

	journal, err := beginApplyJournal(targetRoot)
	if err != nil {
		t.Fatalf("beginApplyJournal() error = %v", err)
	}

	staleRollbackIndex, err := journal.appendAction(rollbackAction{
		Kind:           rollbackRemove,
		Root:           targetRoot,
		Path:           stalePath,
		ExpectedExists: true,
		ExpectedHash:   sha256Hex([]byte("stale\n")),
		ExpectedSize:   int64(len([]byte("stale\n"))),
		ExpectedMode:   0644,
	})
	if err != nil {
		t.Fatalf("append stale-file rollback action: %v", err)
	}

	if err := journal.markActionApplied(staleRollbackIndex); err != nil {
		t.Fatalf("mark stale-file rollback action applied: %v", err)
	}

	result, err := Apply(fsys, "templates/basic", targetDir, TemplateData{}, ApplyOptions{ExistingFilePolicy: ExistingFilePolicyOverwrite})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if len(result.Created) != 1 || result.Created[0] != "main.go" {
		t.Fatalf("unexpected created result after recovery: %#v", result.Created)
	}

	if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
		t.Fatalf("expected stale file to be removed by recovery, stat err = %v", err)
	}
}

func TestApplyWritesManifestWithFileProvenance(t *testing.T) {
	fsys := fstest.MapFS{
		"templates/basic/main.go.tmpl": {Data: []byte("package main\n")},
	}

	targetDir := t.TempDir()
	result, err := Apply(fsys, "templates/basic", targetDir, TemplateData{}, ApplyOptions{ExistingFilePolicy: ExistingFilePolicyOverwrite})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if len(result.Created) != 1 || result.Created[0] != "main.go" {
		t.Fatalf("unexpected created result: %#v", result.Created)
	}

	manifest := mustReadManifest(t, targetDir)
	if manifest.Version != scaffoldManifestV1 {
		t.Fatalf("unexpected manifest version: %d", manifest.Version)
	}
	if manifest.Generator != "gokart" {
		t.Fatalf("unexpected manifest generator: %q", manifest.Generator)
	}
	if manifest.TemplateRoot != "templates/basic" {
		t.Fatalf("unexpected template root: %q", manifest.TemplateRoot)
	}
	if manifest.ExistingFilePolicy != ExistingFilePolicyOverwrite {
		t.Fatalf("unexpected manifest policy: %q", manifest.ExistingFilePolicy)
	}

	if len(manifest.Files) != 1 {
		t.Fatalf("expected 1 manifest file entry, got %d", len(manifest.Files))
	}

	entry := manifest.Files[0]
	if entry.Path != "main.go" {
		t.Fatalf("unexpected manifest path: %q", entry.Path)
	}
	if entry.Action != "create" {
		t.Fatalf("unexpected manifest action: %q", entry.Action)
	}

	expectedHash := sha256Hex([]byte("package main\n"))
	if entry.TemplateSHA256 != expectedHash {
		t.Fatalf("unexpected template hash: %q", entry.TemplateSHA256)
	}
	if entry.ContentSHA256 != expectedHash {
		t.Fatalf("unexpected content hash: %q", entry.ContentSHA256)
	}
	if entry.Mode != uint32(0644) {
		t.Fatalf("unexpected manifest mode: %o", entry.Mode)
	}
}

func TestApplyManifestTracksSkippedContent(t *testing.T) {
	fsys := fstest.MapFS{
		"templates/basic/main.go.tmpl": {Data: []byte("package main\n")},
	}

	targetDir := t.TempDir()
	dest := filepath.Join(targetDir, "main.go")
	seedContent := []byte("package stale\n")
	if err := os.WriteFile(dest, seedContent, 0644); err != nil {
		t.Fatalf("seed destination file: %v", err)
	}

	result, err := Apply(fsys, "templates/basic", targetDir, TemplateData{}, ApplyOptions{ExistingFilePolicy: ExistingFilePolicySkip})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if len(result.Skipped) != 1 || result.Skipped[0] != "main.go" {
		t.Fatalf("unexpected skipped result: %#v", result.Skipped)
	}

	manifest := mustReadManifest(t, targetDir)
	if len(manifest.Files) != 1 {
		t.Fatalf("expected 1 manifest file entry, got %d", len(manifest.Files))
	}

	entry := manifest.Files[0]
	if entry.Action != "skip" {
		t.Fatalf("unexpected manifest action: %q", entry.Action)
	}

	templateHash := sha256Hex([]byte("package main\n"))
	contentHash := sha256Hex(seedContent)
	if entry.TemplateSHA256 != templateHash {
		t.Fatalf("unexpected template hash: %q", entry.TemplateSHA256)
	}
	if entry.ContentSHA256 != contentHash {
		t.Fatalf("unexpected skipped content hash: %q", entry.ContentSHA256)
	}
	if entry.ContentSHA256 == entry.TemplateSHA256 {
		t.Fatal("expected skipped file content hash to differ from template hash")
	}
}

func TestApplyDryRunDoesNotWriteManifest(t *testing.T) {
	fsys := fstest.MapFS{
		"templates/basic/main.go.tmpl": {Data: []byte("package main\n")},
	}

	targetDir := t.TempDir()
	_, err := Apply(fsys, "templates/basic", targetDir, TemplateData{}, ApplyOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	manifestPath := filepath.Join(targetDir, scaffoldManifestPath)
	if _, err := os.Stat(manifestPath); !os.IsNotExist(err) {
		t.Fatalf("expected no manifest in dry-run mode, stat err = %v", err)
	}
}

func TestApplySkipManifestOptionDoesNotWriteManifest(t *testing.T) {
	fsys := fstest.MapFS{
		"templates/basic/main.go.tmpl": {Data: []byte("package main\n")},
	}

	targetDir := t.TempDir()
	result, err := Apply(fsys, "templates/basic", targetDir, TemplateData{}, ApplyOptions{ExistingFilePolicy: ExistingFilePolicyOverwrite, SkipManifest: true})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if len(result.Created) != 1 || result.Created[0] != "main.go" {
		t.Fatalf("unexpected created result: %#v", result.Created)
	}

	manifestPath := filepath.Join(targetDir, scaffoldManifestPath)
	if _, err := os.Stat(manifestPath); !os.IsNotExist(err) {
		t.Fatalf("expected no manifest when SkipManifest=true, stat err = %v", err)
	}
}

func TestManifestIsDeterministicWithoutGeneratedAtTimestamp(t *testing.T) {
	fsys := fstest.MapFS{
		"templates/basic/main.go.tmpl": {Data: []byte("package main\n")},
	}

	targetDir := t.TempDir()
	if _, err := Apply(fsys, "templates/basic", targetDir, TemplateData{}, ApplyOptions{ExistingFilePolicy: ExistingFilePolicyOverwrite}); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	manifestPath := filepath.Join(targetDir, scaffoldManifestPath)
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest file: %v", err)
	}

	if strings.Contains(string(manifestBytes), "generated_at") {
		t.Fatalf("expected deterministic manifest without generated_at, got: %s", string(manifestBytes))
	}
}

func mustReadManifest(t *testing.T, targetDir string) scaffoldManifest {
	t.Helper()

	manifestPath := filepath.Join(targetDir, scaffoldManifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest file: %v", err)
	}

	var manifest scaffoldManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("decode manifest file: %v", err)
	}

	return manifest
}
