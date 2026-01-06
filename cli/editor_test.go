package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dotcommander/gokart/cli"
)

func TestCaptureInputWithEditor_Echo(t *testing.T) {
	t.Parallel()

	// Use 'cat' as a no-op editor (just returns the initial content)
	// This works because cat with a file argument outputs to stdout, not modifying the file
	// Instead, use 'true' which does nothing, leaving the file unchanged
	result, err := cli.CaptureInputWithEditor("true", "initial content", "txt")
	if err != nil {
		t.Fatalf("CaptureInputWithEditor failed: %v", err)
	}

	if result != "initial content" {
		t.Errorf("expected 'initial content', got %q", result)
	}
}

func TestCaptureInputWithEditor_EmptyInitial(t *testing.T) {
	t.Parallel()

	// 'true' command does nothing, file remains empty
	result, err := cli.CaptureInputWithEditor("true", "", "md")
	if err != nil {
		t.Fatalf("CaptureInputWithEditor failed: %v", err)
	}

	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestCaptureInputWithEditor_StripsTrailingNewlines(t *testing.T) {
	t.Parallel()

	// Create a script that adds trailing newlines
	script := filepath.Join(t.TempDir(), "add-newlines.sh")
	err := os.WriteFile(script, []byte("#!/bin/sh\necho '' >> \"$1\"\necho '' >> \"$1\"\n"), 0755)
	if err != nil {
		t.Fatalf("create script: %v", err)
	}

	result, err := cli.CaptureInputWithEditor(script, "content", "txt")
	if err != nil {
		t.Fatalf("CaptureInputWithEditor failed: %v", err)
	}

	// Should strip trailing newlines but preserve content
	if result != "content" {
		t.Errorf("expected 'content', got %q", result)
	}
}

func TestCaptureInputWithEditor_PreservesExtension(t *testing.T) {
	t.Parallel()

	// Create a script that outputs the filename extension
	script := filepath.Join(t.TempDir(), "check-ext.sh")
	scriptContent := `#!/bin/sh
# Write the file extension to the temp file
ext="${1##*.}"
echo "extension: $ext" > "$1"
`
	err := os.WriteFile(script, []byte(scriptContent), 0755)
	if err != nil {
		t.Fatalf("create script: %v", err)
	}

	result, err := cli.CaptureInputWithEditor(script, "", "sql")
	if err != nil {
		t.Fatalf("CaptureInputWithEditor failed: %v", err)
	}

	if result != "extension: sql" {
		t.Errorf("expected 'extension: sql', got %q", result)
	}
}

func TestCaptureInputWithEditor_InvalidEditor(t *testing.T) {
	t.Parallel()

	_, err := cli.CaptureInputWithEditor("nonexistent-editor-xyz", "", "txt")
	if err == nil {
		t.Error("expected error for invalid editor")
	}
}

func TestCaptureInput_FallsBackToVim(t *testing.T) {
	// This test just verifies the function doesn't panic when $EDITOR is unset
	// We can't actually test vim interaction in unit tests
	t.Skip("requires interactive editor")
}
