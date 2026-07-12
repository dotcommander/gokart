package main

import (
	"bytes"
	"io"
	"os"
	"sync"
	"testing"

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

	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	original := os.Stdout
	os.Stdout = write
	fn()
	_ = write.Close()
	os.Stdout = original
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, read)
	_ = read.Close()
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
