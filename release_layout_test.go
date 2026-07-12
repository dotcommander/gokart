package gokart

import (
	"os"
	"strings"
	"testing"
)

func TestReleaseIncludesInstallableCLI(t *testing.T) {
	t.Parallel()

	commandModule, err := os.ReadFile("cmd/gokart/go.mod")
	if err != nil {
		t.Fatalf("read cmd/gokart/go.mod: %v", err)
	}
	moduleText := string(commandModule)
	if !strings.Contains(moduleText, "github.com/dotcommander/gokart/cli v0.11.0") {
		t.Fatal("cmd/gokart does not require the release-matched cli module")
	}
	if strings.Contains(moduleText, "replace github.com/dotcommander/gokart/cli") {
		t.Fatal("cmd/gokart contains a release-time local cli replacement")
	}

	justfile, err := os.ReadFile("justfile")
	if err != nil {
		t.Fatalf("read justfile: %v", err)
	}
	if !strings.Contains(string(justfile), "cmd/gokart") {
		t.Fatal("release module list omits cmd/gokart")
	}
	if strings.Contains(string(justfile), "git push") {
		t.Fatal("local tag recipe performs an external push")
	}
}
