package fs

import (
	"fmt"
	"os"
	"path/filepath"
)

// ConfigDir returns the app's config directory and creates it if missing.
// Uses os.UserConfigDir() for platform-aware paths:
//
//   - macOS: ~/Library/Application Support/<app>/
//   - Linux: ~/.config/<app>/
func ConfigDir(appName string) (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("config dir: %w", err)
	}
	dir := filepath.Join(base, appName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	return dir, nil
}

// WriteFile writes data atomically by writing to a temp file then renaming.
// Prevents partial writes on crash.
func WriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("chmod temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

// ReadOrCreate reads a file, or creates it with defaultContent if it doesn't exist.
func ReadOrCreate(path string, defaultContent []byte) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		return data, nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read file: %w", err)
	}

	// File doesn't exist — create it
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create dir: %w", err)
	}
	if err := os.WriteFile(path, defaultContent, 0o644); err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}
	return defaultContent, nil
}
