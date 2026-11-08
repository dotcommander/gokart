package app

import (
	"fmt"
	"os"
	"path/filepath"
)

// ConfigDir returns the demo config directory (created if missing)
// under the platform user config directory.
func ConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("config dir: %w", err)
	}
	dir := filepath.Join(base, "demo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("config dir: %w", err)
	}
	return dir, nil
}

// EnsureConfigDir creates the config directory and writes the default
// config.yaml if it does not already exist.
func EnsureConfigDir() error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "config.yaml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte(defaultConfig), 0o644); err != nil {
			return fmt.Errorf("write default config: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("stat config: %w", err)
	}
	return nil
}

const defaultConfig = `# demo configuration
# Run: demo --help
`
