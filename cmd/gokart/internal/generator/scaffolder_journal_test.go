package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestApplyFinalizesJournalAfterInProcessRollback(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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

func TestRecoverPendingCreateAcrossAppliedMarkerCrashWindow(t *testing.T) {
	t.Parallel()
	targetRoot := t.TempDir()
	generatedPath := filepath.Join(targetRoot, "generated.txt")
	generated := []byte("generated\n")

	journal, err := beginApplyJournal(targetRoot)
	if err != nil {
		t.Fatalf("beginApplyJournal() error = %v", err)
	}
	if _, err := journal.appendAction(rollbackAction{
		Kind:           rollbackRemove,
		Root:           targetRoot,
		Path:           generatedPath,
		ExpectedExists: true,
		ExpectedHash:   sha256Hex(generated),
		ExpectedSize:   int64(len(generated)),
		ExpectedMode:   0644,
	}); err != nil {
		t.Fatalf("append create rollback action: %v", err)
	}
	if err := writeFileAtomic(generatedPath, generated, 0644); err != nil {
		t.Fatalf("write generated file: %v", err)
	}

	if err := recoverPendingJournals(targetRoot); err != nil {
		t.Fatalf("recoverPendingJournals() error = %v", err)
	}
	if _, err := os.Stat(generatedPath); !os.IsNotExist(err) {
		t.Fatalf("expected pending create to be removed, stat err = %v", err)
	}
}

func TestRecoverPendingOverwriteAcrossAppliedMarkerCrashWindow(t *testing.T) {
	t.Parallel()
	targetRoot := t.TempDir()
	generatedPath := filepath.Join(targetRoot, "main.go")
	original := []byte("package original\n")
	generated := []byte("package generated\n")
	if err := os.WriteFile(generatedPath, original, 0600); err != nil {
		t.Fatalf("seed original file: %v", err)
	}

	journal, err := beginApplyJournal(targetRoot)
	if err != nil {
		t.Fatalf("beginApplyJournal() error = %v", err)
	}
	if _, err := journal.appendAction(rollbackAction{
		Kind:           rollbackRestore,
		Root:           targetRoot,
		Path:           generatedPath,
		Content:        original,
		Mode:           0600,
		ExpectedExists: true,
		ExpectedHash:   sha256Hex(generated),
		ExpectedSize:   int64(len(generated)),
		ExpectedMode:   0600,
	}); err != nil {
		t.Fatalf("append overwrite rollback action: %v", err)
	}
	if err := writeFileAtomic(generatedPath, generated, 0600); err != nil {
		t.Fatalf("write generated file: %v", err)
	}

	if err := recoverPendingJournals(targetRoot); err != nil {
		t.Fatalf("recoverPendingJournals() error = %v", err)
	}
	content, err := os.ReadFile(generatedPath)
	if err != nil {
		t.Fatalf("read recovered file: %v", err)
	}
	if string(content) != string(original) {
		t.Fatalf("recovered content = %q, want %q", content, original)
	}
	info, err := os.Stat(generatedPath)
	if err != nil {
		t.Fatalf("stat recovered file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("recovered mode = %o, want 600", info.Mode().Perm())
	}
}

func TestRecoverPendingCreateBeforeWriteLeavesPathAbsent(t *testing.T) {
	t.Parallel()
	targetRoot := t.TempDir()
	generatedPath := filepath.Join(targetRoot, "generated.txt")
	generated := []byte("generated\n")

	journal, err := beginApplyJournal(targetRoot)
	if err != nil {
		t.Fatalf("beginApplyJournal() error = %v", err)
	}
	if _, err := journal.appendAction(rollbackAction{
		Kind:           rollbackRemove,
		Root:           targetRoot,
		Path:           generatedPath,
		ExpectedExists: true,
		ExpectedHash:   sha256Hex(generated),
		ExpectedSize:   int64(len(generated)),
		ExpectedMode:   0644,
	}); err != nil {
		t.Fatalf("append create rollback action: %v", err)
	}

	if err := recoverPendingJournals(targetRoot); err != nil {
		t.Fatalf("recoverPendingJournals() error = %v", err)
	}
	if _, err := os.Stat(generatedPath); !os.IsNotExist(err) {
		t.Fatalf("expected unwritten path to remain absent, stat err = %v", err)
	}
	if _, err := os.Stat(journal.Path); !os.IsNotExist(err) {
		t.Fatalf("expected recovered journal cleanup, stat err = %v", err)
	}
}

func TestRecoverPendingOverwriteBeforeWriteLeavesOriginalUnchanged(t *testing.T) {
	t.Parallel()
	targetRoot := t.TempDir()
	generatedPath := filepath.Join(targetRoot, "main.go")
	original := []byte("package original\n")
	generated := []byte("package generated\n")
	if err := os.WriteFile(generatedPath, original, 0600); err != nil {
		t.Fatalf("seed original file: %v", err)
	}

	journal, err := beginApplyJournal(targetRoot)
	if err != nil {
		t.Fatalf("beginApplyJournal() error = %v", err)
	}
	if _, err := journal.appendAction(rollbackAction{
		Kind:           rollbackRestore,
		Root:           targetRoot,
		Path:           generatedPath,
		Content:        original,
		Mode:           0600,
		ExpectedExists: true,
		ExpectedHash:   sha256Hex(generated),
		ExpectedSize:   int64(len(generated)),
		ExpectedMode:   0600,
	}); err != nil {
		t.Fatalf("append overwrite rollback action: %v", err)
	}

	if err := recoverPendingJournals(targetRoot); err != nil {
		t.Fatalf("recoverPendingJournals() error = %v", err)
	}
	content, err := os.ReadFile(generatedPath)
	if err != nil {
		t.Fatalf("read original file: %v", err)
	}
	if string(content) != string(original) {
		t.Fatalf("content after recovery = %q, want %q", content, original)
	}
	info, err := os.Stat(generatedPath)
	if err != nil {
		t.Fatalf("stat original file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("mode after recovery = %o, want 600", info.Mode().Perm())
	}
	if _, err := os.Stat(journal.Path); !os.IsNotExist(err) {
		t.Fatalf("expected recovered journal cleanup, stat err = %v", err)
	}
}

func TestRecoverPendingDependencyMutationWithoutExpectedStateBlocks(t *testing.T) {
	t.Parallel()
	targetRoot := t.TempDir()
	dependencyPath := filepath.Join(targetRoot, "go.mod")
	original := []byte("module example.com/original\n")
	mutated := []byte("module example.com/original\n\nrequire example.com/dependency v1.0.0\n")
	if err := os.WriteFile(dependencyPath, original, 0644); err != nil {
		t.Fatalf("seed dependency file: %v", err)
	}

	journal, err := beginApplyJournal(targetRoot)
	if err != nil {
		t.Fatalf("beginApplyJournal() error = %v", err)
	}
	action, err := rollbackActionForPath(targetRoot, dependencyPath)
	if err != nil {
		t.Fatalf("prepare dependency rollback action: %v", err)
	}
	if _, err := journal.appendAction(action); err != nil {
		t.Fatalf("append dependency rollback action: %v", err)
	}
	if err := writeFileAtomic(dependencyPath, mutated, 0644); err != nil {
		t.Fatalf("mutate dependency file: %v", err)
	}

	err = recoverPendingJournals(targetRoot)
	if err == nil {
		t.Fatal("expected recovery to block without an expected dependency state")
	}
	if !strings.Contains(err.Error(), "recovery blocked") {
		t.Fatalf("expected recovery blocked error, got: %v", err)
	}
	content, readErr := os.ReadFile(dependencyPath)
	if readErr != nil {
		t.Fatalf("read dependency file: %v", readErr)
	}
	if string(content) != string(mutated) {
		t.Fatalf("dependency content after blocked recovery = %q, want %q", content, mutated)
	}
	if _, err := os.Stat(journal.Path); err != nil {
		t.Fatalf("expected blocked journal to remain, stat err = %v", err)
	}
}

func TestRecoverPendingJournalsBlocksWhenGeneratedFileChanged(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
