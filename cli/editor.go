package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CaptureInput opens $EDITOR with initial content and returns the edited text.
//
// This is useful for capturing long-form input like commit messages, SQL queries,
// or configuration in CLI applications.
//
// The extension parameter determines the temp file suffix (e.g., "md", "sql", "json"),
// which helps editors apply syntax highlighting.
//
// Falls back to "vim" if $EDITOR is unset.
//
// Example:
//
//	text, err := cli.CaptureInput("# Enter your notes\n", "md")
//	if err != nil {
//	    return err
//	}
//	fmt.Println("You entered:", text)
func CaptureInput(initial string, extension string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	return CaptureInputWithEditor(editor, initial, extension)
}

// CaptureInputWithEditor opens a specific editor with initial content.
//
// This variant is useful for testing or when you want to override the default editor.
//
// Example:
//
//	// Force nano regardless of $EDITOR
//	text, err := cli.CaptureInputWithEditor("nano", "", "txt")
func CaptureInputWithEditor(editor, initial, extension string) (string, error) {
	// Create temp file with appropriate extension
	pattern := "gokart-input-*"
	if extension != "" {
		pattern = "gokart-input-*." + extension
	}

	tmpfile, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	// Ensure restrictive permissions for potentially sensitive content
	if err := tmpfile.Chmod(0600); err != nil {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
		return "", fmt.Errorf("set temp file permissions: %w", err)
	}
	tmpPath := tmpfile.Name()
	defer os.Remove(tmpPath)

	// Write initial content if provided
	if initial != "" {
		if _, err := tmpfile.WriteString(initial); err != nil {
			tmpfile.Close()
			return "", fmt.Errorf("write initial content: %w", err)
		}
	}

	// Close file before opening editor (some editors need exclusive access)
	if err := tmpfile.Close(); err != nil {
		return "", fmt.Errorf("close temp file: %w", err)
	}

	// Open editor with the temp file
	cmd := exec.Command(editor, tmpPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("run editor: %w", err)
	}

	// Read the edited content
	content, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", fmt.Errorf("read edited content: %w", err)
	}

	// Strip trailing newlines added by editors
	result := strings.TrimRight(string(content), "\n")

	return result, nil
}
