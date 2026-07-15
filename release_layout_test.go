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
	if !strings.Contains(moduleText, "github.com/alecthomas/kong v1.15.0") {
		t.Fatal("cmd/gokart does not require Kong")
	}
	if strings.Contains(moduleText, "\nreplace") {
		t.Fatal("cmd/gokart contains a release-time replace directive")
	}

	justfile, err := os.ReadFile("justfile")
	if err != nil {
		t.Fatalf("read justfile: %v", err)
	}
	justfileText := string(justfile)
	if !strings.Contains(justfileText, "cli cmd/gokart") {
		t.Fatal("release module list omits cli or cmd/gokart")
	}
	if strings.Contains(justfileText, "git push") {
		t.Fatal("local tag recipe performs an external push")
	}
	if !strings.Contains(justfileText, `git tag -a "$tag" -m "Release $tag"`) || !strings.Contains(justfileText, `git tag -a "$v" -m "Release $v"`) {
		t.Fatal("release tags must be annotated")
	}

	workspace, err := os.ReadFile("go.work")
	if err != nil {
		t.Fatalf("read go.work: %v", err)
	}
	workspaceText := string(workspace)
	if !strings.HasPrefix(workspaceText, "go 1.26.1\n\ntoolchain go1.26.3\n") {
		t.Fatal("workspace Go/toolchain contract drifted")
	}
	if strings.Contains(workspaceText, "\nreplace ") {
		t.Fatal("workspace modules must resolve through local use entries")
	}
	if !strings.Contains(workspaceText, "\n\t./cli\n") {
		t.Fatal("workspace omits cli")
	}
	if !strings.Contains(workspaceText, "\n\t./cmd/gokart\n") {
		t.Fatal("release module list omits cmd/gokart")
	}
}
