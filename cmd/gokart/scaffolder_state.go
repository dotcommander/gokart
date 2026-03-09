package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
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

		journalPath := filepath.Join(txDir, entry.Name())
		journal, err := loadApplyJournal(journalPath)
		if err != nil {
			return fmt.Errorf("load scaffold journal %s: %w", entry.Name(), err)
		}
		journal.State.TargetRoot = targetRoot

		if journal.State.Completed {
			if err := journal.cleanup(); err != nil {
				return fmt.Errorf("cleanup completed scaffold journal %s: %w", entry.Name(), err)
			}
			continue
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
		return applyJournalEntry{}, fmt.Errorf("rollback action root is empty")
	}

	if action.Path == "" {
		return applyJournalEntry{}, fmt.Errorf("rollback action path is empty")
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
		return nil, fmt.Errorf("journal target root is empty")
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

	info, err := os.Lstat(destPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &journalRecoveryMismatchError{RelPath: entry.RelPath, Reason: "expected generated file is missing"}
		}
		return fmt.Errorf("inspect expected state for %s: %w", entry.RelPath, err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return &journalRecoveryMismatchError{RelPath: entry.RelPath, Reason: "expected generated file became a symlink"}
	}

	if info.IsDir() {
		return &journalRecoveryMismatchError{RelPath: entry.RelPath, Reason: "expected generated file became a directory"}
	}

	if !info.Mode().IsRegular() {
		return &journalRecoveryMismatchError{RelPath: entry.RelPath, Reason: "expected generated file is not regular"}
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

func acquireApplyLock(targetDir string) (func() error, error) {
	targetRoot, err := filepath.Abs(targetDir)
	if err != nil {
		return nil, fmt.Errorf("resolve target directory %q for lock: %w", targetDir, err)
	}

	createdTargetDir := false
	info, err := os.Stat(targetRoot)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("check target directory %q for lock: %w", targetDir, err)
		}

		if err := os.MkdirAll(targetRoot, 0755); err != nil {
			return nil, fmt.Errorf("create target directory %q for lock: %w", targetDir, err)
		}
		createdTargetDir = true
	} else if !info.IsDir() {
		return nil, fmt.Errorf("target path %q exists and is not a directory", targetDir)
	}

	lockPath := filepath.Join(targetRoot, ".gokart.lock")
	if err := ensureNoSymlinkFromRoot(targetRoot, lockPath); err != nil {
		if createdTargetDir {
			_ = os.Remove(targetRoot)
		}
		return nil, fmt.Errorf("cannot lock target directory %q: %w", targetDir, err)
	}

	if err := createApplyLockFile(lockPath); err != nil {
		if !os.IsExist(err) {
			if createdTargetDir {
				_ = os.Remove(targetRoot)
			}
			return nil, fmt.Errorf("create lock file %s: %w", filepath.ToSlash(lockPath), err)
		}

		isStale, reason, staleErr := shouldReclaimStaleLock(lockPath)
		if staleErr != nil {
			if createdTargetDir {
				_ = os.Remove(targetRoot)
			}
			return nil, &ApplyLockError{TargetDir: targetDir, Reason: fmt.Sprintf("existing lock unreadable: %v", staleErr)}
		}

		if !isStale {
			if createdTargetDir {
				_ = os.Remove(targetRoot)
			}
			return nil, &ApplyLockError{TargetDir: targetDir}
		}

		if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
			if createdTargetDir {
				_ = os.Remove(targetRoot)
			}
			return nil, fmt.Errorf("remove stale lock file %s: %w", filepath.ToSlash(lockPath), err)
		}

		if err := createApplyLockFile(lockPath); err != nil {
			if createdTargetDir {
				_ = os.Remove(targetRoot)
			}
			if os.IsExist(err) {
				return nil, &ApplyLockError{TargetDir: targetDir}
			}
			return nil, fmt.Errorf("recreate lock file %s after stale cleanup (%s): %w", filepath.ToSlash(lockPath), reason, err)
		}
	}

	return func() error {
		var errs []error

		if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("remove lock file %s: %w", filepath.ToSlash(lockPath), err))
		}

		if createdTargetDir {
			_ = os.Remove(targetRoot)
		}

		if len(errs) == 0 {
			return nil
		}

		return errors.Join(errs...)
	}, nil
}

func createApplyLockFile(lockPath string) error {
	lockFile, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}

	metadata := applyLockMetadata{
		PID:        os.Getpid(),
		CreatedAt:  time.Now().UTC(),
		StaleAfter: int64(applyLockStaleAfter / time.Second),
	}

	if _, err := lockFile.Write(metadata.encode()); err != nil {
		_ = lockFile.Close()
		_ = os.Remove(lockPath)
		return fmt.Errorf("write lock metadata: %w", err)
	}

	if err := lockFile.Sync(); err != nil {
		_ = lockFile.Close()
		_ = os.Remove(lockPath)
		return fmt.Errorf("sync lock metadata: %w", err)
	}

	if err := lockFile.Close(); err != nil {
		_ = os.Remove(lockPath)
		return fmt.Errorf("close lock file: %w", err)
	}

	return nil
}

type applyLockMetadata struct {
	PID        int
	CreatedAt  time.Time
	StaleAfter int64
}

func (m applyLockMetadata) encode() []byte {
	createdAt := m.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	staleAfter := m.StaleAfter
	if staleAfter <= 0 {
		staleAfter = int64(applyLockStaleAfter / time.Second)
	}

	return []byte(fmt.Sprintf("pid=%d\ncreated_unix=%d\nstale_after_seconds=%d\n", m.PID, createdAt.Unix(), staleAfter))
}

func shouldReclaimStaleLock(lockPath string) (bool, string, error) {
	info, err := os.Lstat(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return true, "lock vanished", nil
		}
		return false, "", err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return false, "", fmt.Errorf("lock path is a symlink")
	}

	if info.IsDir() {
		return false, "", fmt.Errorf("lock path is a directory")
	}

	if !info.Mode().IsRegular() {
		return false, "", fmt.Errorf("lock path is not a regular file")
	}

	content, err := os.ReadFile(lockPath)
	if err != nil {
		return false, "", fmt.Errorf("read lock file: %w", err)
	}

	metadata, parseErr := parseApplyLockMetadata(content)
	if parseErr != nil {
		if lockAge := time.Since(info.ModTime()); lockAge > applyLockStaleAfter {
			return true, fmt.Sprintf("unparseable lock older than %s", applyLockStaleAfter), nil
		}
		return false, "", parseErr
	}

	lockAge := time.Since(info.ModTime())
	maxAge := applyLockStaleAfter
	if metadata.StaleAfter > 0 {
		maxAge = time.Duration(metadata.StaleAfter) * time.Second
	}

	if metadata.PID > 0 {
		running, err := processIsRunning(metadata.PID)
		if err != nil {
			return false, "", fmt.Errorf("check lock pid %d: %w", metadata.PID, err)
		}
		if !running {
			return true, fmt.Sprintf("pid %d is not running", metadata.PID), nil
		}

		if lockAge > maxAge {
			return true, fmt.Sprintf("pid %d is running but lock is older than %s", metadata.PID, maxAge), nil
		}

		return false, "", nil
	}

	if lockAge > maxAge {
		return true, fmt.Sprintf("lock older than %s", maxAge), nil
	}

	return false, "", nil
}

func parseApplyLockMetadata(content []byte) (applyLockMetadata, error) {
	metadata := applyLockMetadata{}
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return applyLockMetadata{}, fmt.Errorf("invalid lock metadata line %q", line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "pid":
			pid, err := parsePositiveInt(value)
			if err != nil {
				return applyLockMetadata{}, fmt.Errorf("invalid pid %q", value)
			}
			metadata.PID = pid
		case "created_unix":
			createdUnix, err := parsePositiveInt64(value)
			if err != nil {
				return applyLockMetadata{}, fmt.Errorf("invalid created_unix %q", value)
			}
			metadata.CreatedAt = time.Unix(createdUnix, 0).UTC()
		case "stale_after_seconds":
			seconds, err := parsePositiveInt64(value)
			if err != nil {
				return applyLockMetadata{}, fmt.Errorf("invalid stale_after_seconds %q", value)
			}
			metadata.StaleAfter = seconds
		}
	}

	if metadata.PID <= 0 {
		return applyLockMetadata{}, fmt.Errorf("missing pid metadata")
	}

	return metadata, nil
}

func processIsRunning(pid int) (bool, error) {
	if pid <= 0 {
		return false, nil
	}

	err := syscall.Kill(pid, 0)
	if err == nil {
		return true, nil
	}

	if errors.Is(err, syscall.ESRCH) {
		return false, nil
	}

	if errors.Is(err, syscall.EPERM) {
		return true, nil
	}

	return false, err
}

func parsePositiveInt(value string) (int, error) {
	parsed, err := parsePositiveInt64(value)
	if err != nil {
		return 0, err
	}
	return int(parsed), nil
}

func parsePositiveInt64(value string) (int64, error) {
	if value == "" {
		return 0, fmt.Errorf("empty numeric value")
	}

	var n int64
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid numeric value %q", value)
		}
		n = n*10 + int64(ch-'0')
	}

	return n, nil
}

func writeFileAtomic(path string, data []byte, perm fs.FileMode) error {
	dir := filepath.Dir(path)

	tmpFile, err := os.CreateTemp(dir, ".gokart-*")
	if err != nil {
		return fmt.Errorf("create temp file for %s: %w", path, err)
	}

	tmpPath := tmpFile.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write temp file for %s: %w", path, err)
	}

	if err := tmpFile.Chmod(perm); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("set mode for %s: %w", path, err)
	}

	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("sync temp file for %s: %w", path, err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file for %s: %w", path, err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		renameErr := err

		info, statErr := os.Stat(path)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				return fmt.Errorf("replace %s: %w", path, renameErr)
			}
			return fmt.Errorf("replace %s: %w (check destination: %v)", path, renameErr, statErr)
		}
		if info.IsDir() {
			return fmt.Errorf("replace %s: destination is a directory", path)
		}

		backupFile, err := os.CreateTemp(dir, ".gokart-backup-*")
		if err != nil {
			return fmt.Errorf("replace %s: %w (create backup temp file: %v)", path, renameErr, err)
		}
		backupPath := backupFile.Name()

		if err := backupFile.Close(); err != nil {
			return fmt.Errorf("replace %s: %w (close backup temp file: %v)", path, renameErr, err)
		}

		if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("replace %s: %w (prepare backup path: %v)", path, renameErr, err)
		}

		if err := os.Rename(path, backupPath); err != nil {
			return fmt.Errorf("replace %s: %w (stage existing file: %v)", path, renameErr, err)
		}

		if err := os.Rename(tmpPath, path); err != nil {
			if restoreErr := os.Rename(backupPath, path); restoreErr != nil {
				return fmt.Errorf("replace %s: %w (restore failed: %v)", path, err, restoreErr)
			}
			return fmt.Errorf("replace %s: %w", path, err)
		}

		_ = os.Remove(backupPath)
	}

	if err := syncDirBestEffort(dir); err != nil {
		return fmt.Errorf("sync directory for %s: %w", path, err)
	}

	cleanup = false
	return nil
}

func syncDirBestEffort(dir string) error {
	fd, err := os.Open(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer fd.Close()

	if err := fd.Sync(); err != nil {
		if errors.Is(err, syscall.EINVAL) || errors.Is(err, syscall.ENOTSUP) || errors.Is(err, syscall.EPERM) {
			return nil
		}
		return err
	}

	return nil
}
