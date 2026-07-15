package generator

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func beginApplyJournal(targetRoot string) (*applyJournal, error) {
	journalDir := filepath.Join(targetRoot, ".gokart", "tx")
	if err := ensureNoSymlinkFromRoot(targetRoot, journalDir); err != nil {
		return nil, fmt.Errorf("prepare journal directory: %w", err)
	}
	if err := os.MkdirAll(journalDir, 0755); err != nil {
		return nil, fmt.Errorf("create journal directory %s: %w", filepath.ToSlash(journalDir), err)
	}
	id := fmt.Sprintf("apply-%d-%d", time.Now().UnixNano(), os.Getpid())
	path := filepath.Join(journalDir, id+".json")
	journal := &applyJournal{
		Path: path,
		State: applyJournalState{
			Version: applyJournalVersion, ID: id, TargetRoot: targetRoot,
			CreatedAt: time.Now().UTC(), Completed: false,
		},
	}
	if err := journal.save(); err != nil {
		return nil, fmt.Errorf("initialize scaffold journal: %w", err)
	}
	return journal, nil
}

func recoverPendingJournals(targetRoot string) error {
	txDir := filepath.Join(targetRoot, ".gokart", "tx")
	entries, err := os.ReadDir(txDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("list scaffold journals in %s: %w", filepath.ToSlash(txDir), err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		if err := recoverSingleJournal(txDir, targetRoot, entry); err != nil {
			return err
		}
	}
	return nil
}

func recoverSingleJournal(journalDir, targetRoot string, entry os.DirEntry) error {
	journalPath := filepath.Join(journalDir, entry.Name())
	journal, err := loadApplyJournal(journalPath)
	if err != nil {
		return fmt.Errorf("load scaffold journal %s: %w", entry.Name(), err)
	}
	journal.State.TargetRoot = targetRoot
	if journal.State.Completed {
		if err := journal.cleanup(); err != nil {
			return fmt.Errorf("cleanup completed scaffold journal %s: %w", entry.Name(), err)
		}
		return nil
	}

	actions, err := journal.State.rollbackActions()
	if err != nil {
		var mismatchErr *journalRecoveryMismatchError
		if errors.As(err, &mismatchErr) {
			return fmt.Errorf("recovery blocked for unfinished scaffold journal %s: %s (target files changed after failure; resolve manually or remove %s if safe)", entry.Name(), mismatchErr.Error(), filepath.ToSlash(journal.Path))
		}
		return fmt.Errorf("decode rollback actions for scaffold journal %s: %w", entry.Name(), err)
	}
	if err := rollbackWrites(actions); err != nil {
		return fmt.Errorf("recover unfinished scaffold journal %s: %w", entry.Name(), err)
	}
	journal.State.Completed = true
	if err := journal.save(); err != nil {
		return fmt.Errorf("mark recovered scaffold journal %s complete: %w", entry.Name(), err)
	}
	if err := journal.cleanup(); err != nil {
		return fmt.Errorf("cleanup recovered scaffold journal %s: %w", entry.Name(), err)
	}
	return nil
}

func loadApplyJournal(path string) (*applyJournal, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read journal file: %w", err)
	}
	var state applyJournalState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("decode journal file: %w", err)
	}
	if state.Version == 1 {
		for i := range state.Actions {
			state.Actions[i].Applied = true
		}
		state.Version = applyJournalVersion
	}
	if state.Version != applyJournalVersion {
		return nil, fmt.Errorf("unsupported journal version %d", state.Version)
	}
	if strings.TrimSpace(state.TargetRoot) == "" {
		txDir := filepath.Dir(path)
		gokartDir := filepath.Dir(txDir)
		state.TargetRoot = filepath.Dir(gokartDir)
	}
	return &applyJournal{Path: path, State: state}, nil
}
