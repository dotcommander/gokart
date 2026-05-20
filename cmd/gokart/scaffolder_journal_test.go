package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

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
