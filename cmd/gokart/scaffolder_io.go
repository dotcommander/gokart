package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

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
		if fallbackErr := renameWithFallback(tmpPath, path, dir, err); fallbackErr != nil {
			return fallbackErr
		}
	}

	if err := syncDirBestEffort(dir); err != nil {
		return fmt.Errorf("sync directory for %s: %w", path, err)
	}

	cleanup = false
	return nil
}

// renameWithFallback is called when os.Rename(tmpPath, path) fails. It attempts
// a backup-and-swap strategy: stage the existing file aside, rename tmpPath into
// place, then remove the backup. If the second rename also fails the backup is
// restored so the destination is not left empty.
func renameWithFallback(tmpPath, path, dir string, renameErr error) error {
	info, statErr := os.Stat(path)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return fmt.Errorf("replace %s: %w", path, renameErr)
		}
		return fmt.Errorf("replace %s: %w (check destination: %v)", path, renameErr, statErr) //nolint:errorlint // secondary error, primary already wrapped
	}
	if info.IsDir() {
		return fmt.Errorf("replace %s: destination is a directory", path)
	}

	backupFile, err := os.CreateTemp(dir, ".gokart-backup-*")
	if err != nil {
		return fmt.Errorf("replace %s: %w (create backup temp file: %v)", path, renameErr, err) //nolint:errorlint // secondary error, primary already wrapped
	}
	backupPath := backupFile.Name()

	if err := backupFile.Close(); err != nil {
		return fmt.Errorf("replace %s: %w (close backup temp file: %v)", path, renameErr, err) //nolint:errorlint // secondary error, primary already wrapped
	}

	if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("replace %s: %w (prepare backup path: %v)", path, renameErr, err) //nolint:errorlint // secondary error, primary already wrapped
	}

	if err := os.Rename(path, backupPath); err != nil {
		return fmt.Errorf("replace %s: %w (stage existing file: %v)", path, renameErr, err) //nolint:errorlint // secondary error, primary already wrapped
	}

	if err := os.Rename(tmpPath, path); err != nil {
		if restoreErr := os.Rename(backupPath, path); restoreErr != nil {
			return fmt.Errorf("replace %s: %w (restore failed: %v)", path, err, restoreErr) //nolint:errorlint // secondary error, primary already wrapped
		}
		return fmt.Errorf("replace %s: %w", path, err)
	}

	_ = os.Remove(backupPath)
	return nil
}

// assertRegularFile calls Lstat on path and verifies the result is a regular file.
// Returns the FileInfo on success. If the path does not exist, returns an error
// satisfying os.IsNotExist. If the path exists but is not a regular file, returns
// *notRegularFileError.
func assertRegularFile(path string) (fs.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return nil, &notRegularFileError{Path: path, Reason: "is a symlink"}
	}

	if info.IsDir() {
		return nil, &notRegularFileError{Path: path, Reason: "is a directory"}
	}

	if !info.Mode().IsRegular() {
		return nil, &notRegularFileError{Path: path, Reason: "is not a regular file"}
	}

	return info, nil
}

func syncDirBestEffort(dir string) error {
	fd, err := os.Open(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer func() { _ = fd.Close() }() //nolint:errcheck // best-effort dir sync

	if err := fd.Sync(); err != nil {
		if errors.Is(err, syscall.EINVAL) || errors.Is(err, syscall.ENOTSUP) || errors.Is(err, syscall.EPERM) {
			return nil
		}
		return err
	}

	return nil
}
