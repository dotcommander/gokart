package main

import (
	"bytes"
	"sync"
	"testing"

	"github.com/dotcommander/gokart/cli"
	"github.com/spf13/cobra"
)

var captureOutputMu sync.Mutex

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

	captureOutputMu.Lock()
	defer captureOutputMu.Unlock()

	var buf bytes.Buffer
	cli.SetOutput(&buf)
	defer cli.SetOutput(nil)
	fn()

	return buf.String()
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}

	return false
}
