package gokart

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// SaveState saves typed state under the platform user config directory.
//
// The file is written as indented JSON for human readability.
// Directory is created with 0755, files with 0600 permissions.
//
// Example:
//
//	type AppState struct {
//	    LastOpened string `json:"last_opened"`
//	    WindowSize int    `json:"window_size"`
//	}
//	err := gokart.SaveState("myapp", "state.json", AppState{
//	    LastOpened: "/path/to/file",
//	    WindowSize: 1024,
//	})
func SaveState[T any](appName, filename string, data T) error {
	if err := validateLocalName("state filename", filename); err != nil {
		return err
	}
	dir, err := stateDir(appName)
	if err != nil {
		return fmt.Errorf("get config dir: %w", err)
	}

	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	path := filepath.Join(dir, filename)
	if err := AtomicWriteFile(path, content, 0o600); err != nil {
		return fmt.Errorf("publish state file: %w", err)
	}

	return nil
}

// maxStateBytes bounds LoadState reads. State is human-readable indented
// JSON, so 10 MiB rails against pathological files, not a capacity policy.
const maxStateBytes = 10 << 20

// LoadState loads typed state from the platform user config directory.
//
// The file is read through ReadFileLimit with maxStateBytes: a larger
// file returns zero value and an error satisfying errors.Is(err,
// ErrTooLarge). Callers needing a different bound can read the file
// themselves with ReadFileLimit.
//
// Returns zero value and os.ErrNotExist if the file doesn't exist.
// This allows callers to distinguish between missing file and parse errors.
//
// Example:
//
//	state, err := gokart.LoadState[AppState]("myapp", "state.json")
//	if errors.Is(err, os.ErrNotExist) {
//	    // First run, use defaults
//	    state = AppState{WindowSize: 800}
//	} else if err != nil {
//	    return err
//	}
func LoadState[T any](appName, filename string) (T, error) {
	var zero T
	if err := validateLocalName("state filename", filename); err != nil {
		return zero, err
	}

	path := StatePath(appName, filename)
	if path == "" {
		return zero, fmt.Errorf("invalid app name %q: cannot determine state path", appName)
	}
	content, err := ReadFileLimit(path, maxStateBytes)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return zero, os.ErrNotExist
		}
		return zero, fmt.Errorf("read state file: %w", err)
	}

	var result T
	if err := json.Unmarshal(content, &result); err != nil {
		return zero, fmt.Errorf("unmarshal state: %w", err)
	}

	return result, nil
}

// StatePath returns the full path to a state file.
//
// Returns empty string if the user config directory cannot be determined
// or either name is invalid (absolute, empty, or contains `..`).
//
// Example:
//
//	path := gokart.StatePath("myapp", "state.json")
//	// macOS: ~/Library/Application Support/myapp/state.json
//	// Linux: ~/.config/myapp/state.json
func StatePath(appName, filename string) string {
	if err := validateLocalName("state filename", filename); err != nil {
		return ""
	}
	dir, err := stateDir(appName)
	if err != nil {
		return ""
	}
	return filepath.Join(dir, filename)
}

// stateDir returns the config directory for the app.
func stateDir(appName string) (string, error) {
	if err := validateLocalName("app name", appName); err != nil {
		return "", err
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, appName), nil
}
