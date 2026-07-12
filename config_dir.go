package gokart

import (
	"fmt"
	"os"
	"path/filepath"
)

// ConfigDir returns the app's platform-specific configuration directory,
// creating it when necessary.
func ConfigDir(appName string) (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("get user config directory: %w", err)
	}
	dir := filepath.Join(base, appName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create config directory: %w", err)
	}
	return dir, nil
}

// EnsureConfigDir creates the app's configuration directory and initializes
// config.yaml with defaultContent when it does not already exist.
func EnsureConfigDir(appName string, defaultContent []byte) error {
	dir, err := ConfigDir(appName)
	if err != nil {
		return err
	}
	if _, err := readOrCreateFile(filepath.Join(dir, "config.yaml"), defaultContent, 0o644); err != nil {
		return fmt.Errorf("initialize config file: %w", err)
	}
	return nil
}
