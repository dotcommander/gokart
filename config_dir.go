package gokart

import (
	"fmt"
	"os"
	"path/filepath"
)

// ConfigDir returns the app's platform-specific configuration directory,
// creating it when necessary.
// validateLocalName rejects names that could escape their parent directory:
// absolute paths, `..` segments, and empty/`"` values. Relative subpaths
// without parent segments (e.g. `sub/dir`) stay allowed.
func validateLocalName(kind, name string) error {
	if !filepath.IsLocal(name) || filepath.Clean(name) == "." {
		return fmt.Errorf("invalid %s %q: must name a local application directory", kind, name)
	}
	return nil
}

func ConfigDir(appName string) (string, error) {
	if err := validateLocalName("app name", appName); err != nil {
		return "", err
	}

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
