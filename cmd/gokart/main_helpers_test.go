package main

import (
	"io"
	"os"
	"testing"

	"github.com/spf13/cobra"
)

func newNewCommandForTest() *cobra.Command {
	cmd := &cobra.Command{Use: "new"}
	configureNewCommandFlags(cmd)
	return cmd
}

func mustSetFlagTrue(t *testing.T, cmd *cobra.Command, name string) {
	t.Helper()
	if err := cmd.Flags().Set(name, "true"); err != nil {
		t.Fatalf("set flag %s=true: %v", name, err)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}

	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = originalStdout

	data, readErr := io.ReadAll(r)
	if readErr != nil {
		t.Fatalf("read stdout capture: %v", readErr)
	}
	_ = r.Close()

	return string(data)
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}

	return false
}
