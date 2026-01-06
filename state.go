package gokart

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// SaveState saves typed state to ~/.config/{appName}/{filename}.
//
// The file is written as indented JSON for human readability.
// Directory is created with 0755, files with 0644 permissions.
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
	dir, err := stateDir(appName)
	if err != nil {
		return fmt.Errorf("get config dir: %w", err)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("write state file: %w", err)
	}

	return nil
}

// LoadState loads typed state from ~/.config/{appName}/{filename}.
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

	path := StatePath(appName, filename)
	content, err := os.ReadFile(path)
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
// Returns empty string if the user config directory cannot be determined.
//
// Example:
//
//	path := gokart.StatePath("myapp", "state.json")
//	// Returns: /Users/username/.config/myapp/state.json (on macOS)
func StatePath(appName, filename string) string {
	dir, err := stateDir(appName)
	if err != nil {
		return ""
	}
	return filepath.Join(dir, filename)
}

// stateDir returns the config directory for the app.
func stateDir(appName string) (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, appName), nil
}
