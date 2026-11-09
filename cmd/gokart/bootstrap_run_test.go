package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunVersionFlag(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"--version"}, "testbin", &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "version") {
		t.Errorf("stdout = %q, want version line", stdout.String())
	}
}

func TestRunInvalidArgsExit2(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"bogus-command"}, "testbin", &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
}
