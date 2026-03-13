package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

//nolint:gocognit,gocyclo // lock acquisition with stale reclaim, complexity is inherent
func acquireApplyLock(targetDir string) (release func() error, retErr error) {
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

	succeeded := false
	defer func() {
		if !succeeded && createdTargetDir {
			_ = os.Remove(targetRoot)
		}
	}()

	lockPath := filepath.Join(targetRoot, ".gokart.lock")
	if err := ensureNoSymlinkFromRoot(targetRoot, lockPath); err != nil {
		return nil, fmt.Errorf("cannot lock target directory %q: %w", targetDir, err)
	}

	if err := createApplyLockFile(lockPath); err != nil {
		if !os.IsExist(err) {
			return nil, fmt.Errorf("create lock file %s: %w", filepath.ToSlash(lockPath), err)
		}
		if err := reclaimOrFailLock(lockPath, targetDir); err != nil {
			return nil, err
		}
	}

	succeeded = true
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

// reclaimOrFailLock is called when createApplyLockFile returns os.IsExist.
// It checks whether the existing lock is stale and, if so, removes it and
// recreates it. Returns nil on success, *ApplyLockError when another process
// holds the lock, or a wrapped error for unexpected failures.
func reclaimOrFailLock(lockPath, targetDir string) error {
	isStale, reason, staleErr := shouldReclaimStaleLock(lockPath)
	if staleErr != nil {
		return &ApplyLockError{TargetDir: targetDir, Reason: fmt.Sprintf("existing lock unreadable: %v", staleErr)}
	}

	if !isStale {
		return &ApplyLockError{TargetDir: targetDir}
	}

	if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove stale lock file %s: %w", filepath.ToSlash(lockPath), err)
	}

	if err := createApplyLockFile(lockPath); err != nil {
		if os.IsExist(err) {
			return &ApplyLockError{TargetDir: targetDir}
		}
		return fmt.Errorf("recreate lock file %s after stale cleanup (%s): %w", filepath.ToSlash(lockPath), reason, err)
	}

	return nil
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
	info, err := assertRegularFile(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return true, "lock vanished", nil
		}
		var nrf *notRegularFileError
		if errors.As(err, &nrf) {
			return false, "", fmt.Errorf("lock path %s", nrf.Reason)
		}
		return false, "", err
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
		return applyLockMetadata{}, errors.New("missing pid metadata")
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
		return 0, errors.New("empty numeric value")
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
