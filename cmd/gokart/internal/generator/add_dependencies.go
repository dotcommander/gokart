package generator

import (
	"fmt"
	"os"
	"path/filepath"
)

type dependencyRollbackAction struct {
	index  int
	action rollbackAction
}

func journalDependencyFiles(dir string, journal *applyJournal) ([]dependencyRollbackAction, error) {
	actions := make([]dependencyRollbackAction, 0, 2)
	for _, relPath := range []string{"go.mod", "go.sum"} {
		path := filepath.Join(dir, relPath)
		action, err := rollbackActionForPath(dir, path)
		if err != nil {
			return nil, fmt.Errorf("prepare rollback for %s: %w", relPath, err)
		}
		index, err := journal.appendAction(action)
		if err != nil {
			return nil, fmt.Errorf("record rollback intent for %s: %w", relPath, err)
		}
		actions = append(actions, dependencyRollbackAction{index: index, action: action})
	}
	return actions, nil
}

func markDependencyFilesApplied(actions []dependencyRollbackAction, journal *applyJournal, applied []rollbackAction) ([]rollbackAction, error) {
	for _, dep := range actions {
		if err := updateJournalExpectedState(journal, dep.index); err != nil {
			return applied, err
		}
		if err := journal.markActionApplied(dep.index); err != nil {
			return append(applied, dep.action), err
		}
		applied = append(applied, dep.action)
	}
	return applied, nil
}

func updateJournalExpectedState(journal *applyJournal, index int) error {
	if journal == nil {
		return nil
	}
	if index < 0 || index >= len(journal.State.Actions) {
		return fmt.Errorf("journal action index %d out of range", index)
	}
	entry := &journal.State.Actions[index]
	path := filepath.Join(journal.State.TargetRoot, filepath.FromSlash(entry.RelPath))
	info, err := assertRegularFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			entry.ExpectedExists = false
			entry.ExpectedSHA256 = ""
			entry.ExpectedSize = 0
			entry.ExpectedMode = 0
			return journal.save()
		}
		return fmt.Errorf("inspect dependency file %s: %w", entry.RelPath, err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read dependency file %s: %w", entry.RelPath, err)
	}
	entry.ExpectedExists = true
	entry.ExpectedSHA256 = sha256Hex(content)
	entry.ExpectedSize = int64(len(content))
	entry.ExpectedMode = uint32(info.Mode().Perm())
	return journal.save()
}
