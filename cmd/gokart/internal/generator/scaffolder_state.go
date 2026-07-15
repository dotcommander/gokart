package generator

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	journalActionRemove  = "remove"
	journalActionRestore = "restore"
)

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
		entry.Kind = journalActionRemove
	case rollbackRestore:
		entry.Kind = journalActionRestore
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
		relPath := filepath.FromSlash(entry.RelPath)
		destPath, err := safeDestinationPath(state.TargetRoot, relPath)
		if err != nil {
			return nil, fmt.Errorf("entry %d path %q: %w", i, entry.RelPath, err)
		}

		needsRollback, err := journalEntryNeedsRollback(destPath, entry)
		if err != nil {
			return nil, err
		}
		if !needsRollback {
			continue
		}

		action := rollbackAction{Root: state.TargetRoot, Path: destPath}
		switch entry.Kind {
		case journalActionRemove:
			action.Kind = rollbackRemove
		case journalActionRestore:
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

func journalEntryNeedsRollback(destPath string, entry applyJournalEntry) (bool, error) {
	if entry.Applied {
		if err := verifyJournalActionExpectedState(destPath, entry); err != nil {
			return false, err
		}
		return true, nil
	}

	matchesOriginal, err := journalEntryMatchesOriginalState(destPath, entry)
	if err != nil {
		return false, err
	}
	if matchesOriginal {
		return false, nil
	}
	if !entry.ExpectedExists {
		return false, &journalRecoveryMismatchError{
			RelPath: entry.RelPath,
			Reason:  "pending action changed the destination but does not record an expected generated file",
		}
	}
	if err := verifyJournalActionExpectedState(destPath, entry); err != nil {
		return false, err
	}
	return true, nil
}

func journalEntryMatchesOriginalState(destPath string, entry applyJournalEntry) (bool, error) {
	switch entry.Kind {
	case journalActionRemove:
		_, err := os.Lstat(destPath)
		if os.IsNotExist(err) {
			return true, nil
		}
		if err != nil {
			return false, fmt.Errorf("inspect original state for %s: %w", entry.RelPath, err)
		}
		return false, nil
	case journalActionRestore:
		info, err := assertRegularFile(destPath)
		if err != nil {
			if os.IsNotExist(err) {
				return false, nil
			}
			var nrf *notRegularFileError
			if errors.As(err, &nrf) {
				return false, nil
			}
			return false, fmt.Errorf("inspect original state for %s: %w", entry.RelPath, err)
		}
		content, err := os.ReadFile(destPath)
		if err != nil {
			return false, fmt.Errorf("read original state for %s: %w", entry.RelPath, err)
		}
		originalMode := fs.FileMode(entry.Mode)
		if originalMode == 0 {
			originalMode = 0644
		}
		return bytes.Equal(content, entry.Content) &&
			info.Mode().Perm() == originalMode, nil
	default:
		return false, fmt.Errorf("unknown journal action kind %q", entry.Kind)
	}
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
