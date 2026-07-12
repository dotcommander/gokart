package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestTargetMutationLockRecoversPendingJournalBeforeMutation(t *testing.T) {
	t.Parallel()
	targetDir := t.TempDir()
	createdPath := filepath.Join(targetDir, "created.txt")
	if err := os.WriteFile(createdPath, []byte("unfinished"), 0644); err != nil {
		t.Fatalf("seed unfinished write: %v", err)
	}

	journal, err := beginApplyJournal(targetDir)
	if err != nil {
		t.Fatalf("begin journal: %v", err)
	}
	index, err := journal.appendAction(rollbackAction{
		Kind:           rollbackRemove,
		Root:           targetDir,
		Path:           createdPath,
		ExpectedExists: true,
		ExpectedHash:   sha256Hex([]byte("unfinished")),
		ExpectedSize:   int64(len("unfinished")),
		ExpectedMode:   0644,
	})
	if err != nil {
		t.Fatalf("append journal action: %v", err)
	}
	if err := journal.markActionApplied(index); err != nil {
		t.Fatalf("mark journal action applied: %v", err)
	}

	err = withTargetMutationLock(targetDir, func() error {
		if _, statErr := os.Stat(createdPath); !os.IsNotExist(statErr) {
			t.Fatalf("pending journal was not recovered before mutation: %v", statErr)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("withTargetMutationLock: %v", err)
	}
}

func TestConcurrentTargetMutationsCannotEnterTogether(t *testing.T) {
	t.Parallel()
	targetDir := t.TempDir()
	entered := make(chan struct{})
	release := make(chan struct{})
	firstDone := make(chan error, 1)

	go func() {
		firstDone <- withTargetMutationLock(targetDir, func() error {
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered

	secondEntered := false
	err := withTargetMutationLock(targetDir, func() error {
		secondEntered = true
		return nil
	})
	var lockErr *ApplyLockError
	if !errors.As(err, &lockErr) {
		t.Fatalf("second mutation error = %v, want ApplyLockError", err)
	}
	if secondEntered {
		t.Fatal("second mutation entered while the first held the target lock")
	}

	close(release)
	if err := <-firstDone; err != nil {
		t.Fatalf("first mutation: %v", err)
	}
}
