package gokart

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// atomicWriteFile publishes a fully synced sibling file over path.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".gokart-write-*")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if err := tmp.Chmod(perm); err != nil {
		return closeTemporaryFile(tmp, fmt.Errorf("set temporary file permissions: %w", err))
	}
	if _, err := tmp.Write(data); err != nil {
		return closeTemporaryFile(tmp, fmt.Errorf("write temporary file: %w", err))
	}
	if err := tmp.Sync(); err != nil {
		return closeTemporaryFile(tmp, fmt.Errorf("sync temporary file: %w", err))
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("publish file: %w", err)
	}
	if err := syncDirectory(dir); err != nil {
		return fmt.Errorf("sync directory: %w", err)
	}
	return nil
}

// readOrCreateFile publishes defaultContent only when path is still absent.
// Concurrent initializers all return the contents selected by the winner.
func readOrCreateFile(path string, defaultContent []byte, perm os.FileMode) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		return data, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read file: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".gokart-create-*")
	if err != nil {
		return nil, fmt.Errorf("create temporary file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if err := tmp.Chmod(perm); err != nil {
		return nil, closeTemporaryFile(tmp, fmt.Errorf("set temporary file permissions: %w", err))
	}
	if _, err := tmp.Write(defaultContent); err != nil {
		return nil, closeTemporaryFile(tmp, fmt.Errorf("write temporary file: %w", err))
	}
	if err := tmp.Sync(); err != nil {
		return nil, closeTemporaryFile(tmp, fmt.Errorf("sync temporary file: %w", err))
	}
	if err := tmp.Close(); err != nil {
		return nil, fmt.Errorf("close temporary file: %w", err)
	}

	if err := os.Link(tmpPath, path); err != nil {
		if errors.Is(err, os.ErrExist) {
			winner, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil, fmt.Errorf("read concurrently created file: %w", readErr)
			}
			return winner, nil
		}
		return nil, fmt.Errorf("publish file: %w", err)
	}
	if err := syncDirectory(dir); err != nil {
		return nil, fmt.Errorf("sync directory: %w", err)
	}
	return defaultContent, nil
}

func syncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}

func closeTemporaryFile(file *os.File, operationErr error) error {
	if err := file.Close(); err != nil {
		return errors.Join(operationErr, fmt.Errorf("close temporary file: %w", err))
	}
	return operationErr
}
