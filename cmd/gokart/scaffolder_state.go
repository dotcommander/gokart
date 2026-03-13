package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
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
			Version:    applyJournalVersion,
			ID:         id,
			TargetRoot: targetRoot,
			CreatedAt:  time.Now().UTC(),
			Completed:  false,
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

func (j *applyJournal) appendAction(action rollbackAction) (int, error) {
	if j == nil {
		return -1, nil
	}

	entry, err := rollbackActionToJournalEntry(j.State.TargetRoot, action)
	if err != nil {
		return -1, err
	}

	j.State.Actions = append(j.State.Actions, entry)
	idx := len(j.State.Actions) - 1
	if err := j.save(); err != nil {
		return -1, err
	}

	return idx, nil
}

func (j *applyJournal) markActionApplied(index int) error {
	if j == nil {
		return nil
	}

	if index < 0 || index >= len(j.State.Actions) {
		return fmt.Errorf("journal action index %d out of range", index)
	}

	j.State.Actions[index].Applied = true
	return j.save()
}

func (j *applyJournal) markCompleted() error {
	if j == nil {
		return nil
	}

	j.State.Completed = true
	return j.save()
}

func (j *applyJournal) cleanup() error {
	if j == nil {
		return nil
	}

	if err := os.Remove(j.Path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove journal file %s: %w", filepath.ToSlash(j.Path), err)
	}

	cleanupJournalDirsBestEffort(j.State.TargetRoot)
	return nil
}

func (j *applyJournal) save() error {
	data, err := json.MarshalIndent(j.State, "", "  ")
	if err != nil {
		return fmt.Errorf("encode journal data: %w", err)
	}

	data = append(data, '\n')
	if err := writeFileAtomic(j.Path, data, 0600); err != nil {
		return fmt.Errorf("write journal file %s: %w", filepath.ToSlash(j.Path), err)
	}

	return nil
}

func rollbackActionToJournalEntry(targetRoot string, action rollbackAction) (applyJournalEntry, error) {
	root := action.Root
	if root == "" {
		root = targetRoot
	}

	if root == "" {
		return applyJournalEntry{}, errors.New("rollback action root is empty")
	}

	if action.Path == "" {
		return applyJournalEntry{}, errors.New("rollback action path is empty")
	}

	relPath, err := filepath.Rel(root, action.Path)
	if err != nil {
		return applyJournalEntry{}, fmt.Errorf("compute rollback path for journal: %w", err)
	}

	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return applyJournalEntry{}, fmt.Errorf("rollback path %s escapes root %s", action.Path, root)
	}

	entry := applyJournalEntry{RelPath: filepath.ToSlash(relPath)}

	switch action.Kind {
	case rollbackRemove:
		entry.Kind = "remove"
	case rollbackRestore:
		entry.Kind = "restore"
		entry.Content = action.Content
		entry.Mode = uint32(action.Mode)
	default:
		return applyJournalEntry{}, fmt.Errorf("unsupported rollback action kind %d", action.Kind)
	}

	entry.ExpectedExists = action.ExpectedExists
	entry.ExpectedSHA256 = action.ExpectedHash
	entry.ExpectedSize = action.ExpectedSize
	entry.ExpectedMode = uint32(action.ExpectedMode)

	return entry, nil
}

func (state applyJournalState) rollbackActions() ([]rollbackAction, error) {
	if strings.TrimSpace(state.TargetRoot) == "" {
		return nil, errors.New("journal target root is empty")
	}

	actions := make([]rollbackAction, 0, len(state.Actions))
	for i, entry := range state.Actions {
		if !entry.Applied {
			continue
		}

		relPath := filepath.FromSlash(entry.RelPath)
		destPath, err := safeDestinationPath(state.TargetRoot, relPath)
		if err != nil {
			return nil, fmt.Errorf("entry %d path %q: %w", i, entry.RelPath, err)
		}

		if err := verifyJournalActionExpectedState(destPath, entry); err != nil {
			return nil, err
		}

		action := rollbackAction{Root: state.TargetRoot, Path: destPath}
		switch entry.Kind {
		case "remove":
			action.Kind = rollbackRemove
		case "restore":
			action.Kind = rollbackRestore
			action.Content = entry.Content
			action.Mode = fs.FileMode(entry.Mode)
			if action.Mode == 0 {
				action.Mode = 0644
			}
		default:
			return nil, fmt.Errorf("entry %d has unknown action kind %q", i, entry.Kind)
		}

		actions = append(actions, action)
	}

	return actions, nil
}

func verifyJournalActionExpectedState(destPath string, entry applyJournalEntry) error {
	if !entry.ExpectedExists {
		return nil
	}

	info, err := assertRegularFile(destPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &journalRecoveryMismatchError{RelPath: entry.RelPath, Reason: "expected generated file is missing"}
		}
		var nrf *notRegularFileError
		if errors.As(err, &nrf) {
			return &journalRecoveryMismatchError{RelPath: entry.RelPath, Reason: "expected generated file " + nrf.Reason}
		}
		return fmt.Errorf("inspect expected state for %s: %w", entry.RelPath, err)
	}

	content, err := os.ReadFile(destPath)
	if err != nil {
		return fmt.Errorf("read expected state for %s: %w", entry.RelPath, err)
	}

	if entry.ExpectedSize >= 0 && int64(len(content)) != entry.ExpectedSize {
		return &journalRecoveryMismatchError{RelPath: entry.RelPath, Reason: "generated file size changed after failure"}
	}

	if entry.ExpectedSHA256 != "" && sha256Hex(content) != entry.ExpectedSHA256 {
		return &journalRecoveryMismatchError{RelPath: entry.RelPath, Reason: "generated file content changed after failure"}
	}

	if entry.ExpectedMode != 0 && info.Mode().Perm() != fs.FileMode(entry.ExpectedMode) {
		return &journalRecoveryMismatchError{RelPath: entry.RelPath, Reason: "generated file mode changed after failure"}
	}

	return nil
}

func cleanupJournalDirsBestEffort(targetRoot string) {
	if strings.TrimSpace(targetRoot) == "" {
		return
	}

	txDir := filepath.Join(targetRoot, ".gokart", "tx")
	_ = os.Remove(txDir)
	_ = os.Remove(filepath.Dir(txDir))
}
