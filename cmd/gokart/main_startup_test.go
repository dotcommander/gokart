package main

import (
	"testing"

	"github.com/spf13/cobra"
)

// findRequiredCommand resolves a command (or alias) on root, failing the test
// if the lookup errors or returns nil. Returns the resolved command for
// further assertions.
func findRequiredCommand(t *testing.T, root *cobra.Command, path []string) *cobra.Command {
	t.Helper()
	cmd, _, err := root.Find(path)
	if err != nil {
		t.Fatalf("find %v: %v", path, err)
	}
	if cmd == nil {
		t.Fatalf("expected command for %v, got nil", path)
	}
	return cmd
}

func TestNewGokartAppStartupContractRootCommand(t *testing.T) {
	root := newGokartApp("test-version").Root()

	if root == nil {
		t.Fatal("expected root command")
	}
	if root.Use != gokartAppName {
		t.Fatalf("root.Use = %q, want %q", root.Use, gokartAppName)
	}
	if root.Version != "test-version" {
		t.Fatalf("root.Version = %q, want %q", root.Version, "test-version")
	}
	if root.HelpTemplate() != rootHelpTemplate {
		t.Fatal("root help template drifted")
	}
}

func TestNewGokartAppStartupContractNewCommand(t *testing.T) {
	root := newGokartApp("test-version").Root()

	newCmd := findRequiredCommand(t, root, []string{"new"})

	if newCmd.Use != "new <project-name> | new cli <project-name>" {
		t.Fatalf("new command Use = %q", newCmd.Use)
	}

	if err := newCmd.Args(newCmd, []string{"myapp"}); err != nil {
		t.Fatalf("new command rejected valid args: %v", err)
	}
}

func TestNewGokartAppStartupContractVersionCommand(t *testing.T) {
	root := newGokartApp("test-version").Root()

	versionCmd := findRequiredCommand(t, root, []string{"version"})
	if versionCmd.Name() != "version" {
		t.Fatalf("expected version command, got %q", versionCmd.Name())
	}
}

func TestNewGokartAppStartupContractNewAliases(t *testing.T) {
	root := newGokartApp("test-version").Root()

	aliases := []string{"create", "init"}
	for _, alias := range aliases {
		t.Run(alias, func(t *testing.T) {
			cmd := findRequiredCommand(t, root, []string{alias})
			if cmd.Name() != "new" {
				t.Fatalf("expected %q to resolve to 'new' command, got %q", alias, cmd.Name())
			}
		})
	}
}

func TestNewGokartAppStartupContractCompletionHidden(t *testing.T) {
	root := newGokartApp("test-version").Root()

	for _, cmd := range root.Commands() {
		if cmd.Name() == "completion" && !cmd.Hidden {
			t.Fatal("completion command should be hidden")
		}
	}
}
