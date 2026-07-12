package main

import (
	"bytes"
	"strings"
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

func TestNewGokartAppHelpMatchesDatabaseFlagContract(t *testing.T) {
	t.Parallel()
	root := newGokartApp("test-version").Root()
	var rootHelp bytes.Buffer
	root.SetOut(&rootHelp)
	if err := root.Help(); err != nil {
		t.Fatalf("root help: %v", err)
	}

	newCmd := findRequiredCommand(t, root, []string{"new"})
	var newHelp bytes.Buffer
	newCmd.SetOut(&newHelp)
	if err := newCmd.Help(); err != nil {
		t.Fatalf("new help: %v", err)
	}

	for name, help := range map[string]string{
		"root": rootHelp.String(),
		"new":  newHelp.String(),
	} {
		if strings.Contains(help, "--sqlite") || strings.Contains(help, "--postgres") {
			t.Fatalf("%s help still advertises removed database flags:\n%s", name, help)
		}
		if !strings.Contains(help, "--db") {
			t.Fatalf("%s help does not advertise --db:\n%s", name, help)
		}
	}
}

func TestNewGokartAppStartupContractRootCommand(t *testing.T) {
	t.Parallel()
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

func TestSelectGokartVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		version       string
		moduleVersion string
		want          string
	}{
		{name: "linker version wins", version: "v0.10.2-local", moduleVersion: "v0.10.2", want: "v0.10.2-local"},
		{name: "installed module version", version: "dev", moduleVersion: "v0.10.2", want: "v0.10.2"},
		{name: "local development build", version: "dev", moduleVersion: "(devel)", want: "dev"},
		{name: "missing build info version", version: "dev", want: "dev"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := selectGokartVersion(tc.version, tc.moduleVersion); got != tc.want {
				t.Fatalf("selectGokartVersion(%q, %q) = %q, want %q", tc.version, tc.moduleVersion, got, tc.want)
			}
		})
	}
}

func TestNewGokartAppStartupContractNewCommand(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	root := newGokartApp("test-version").Root()

	versionCmd := findRequiredCommand(t, root, []string{"version"})
	if versionCmd.Name() != "version" {
		t.Fatalf("expected version command, got %q", versionCmd.Name())
	}
}

func TestNewGokartAppStartupContractNewAliases(t *testing.T) {
	t.Parallel()

	aliases := []string{"create", "init"}
	for _, alias := range aliases {
		t.Run(alias, func(t *testing.T) {
			t.Parallel()
			// Build a fresh root per subtest — cobra's Find mutates internal
			// state, so a shared root races under parallel subtests.
			root := newGokartApp("test-version").Root()
			cmd := findRequiredCommand(t, root, []string{alias})
			if cmd.Name() != "new" {
				t.Fatalf("expected %q to resolve to 'new' command, got %q", alias, cmd.Name())
			}
		})
	}
}

func TestNewGokartAppStartupContractCompletionHidden(t *testing.T) {
	t.Parallel()
	root := newGokartApp("test-version").Root()

	for _, cmd := range root.Commands() {
		if cmd.Name() == "completion" && !cmd.Hidden {
			t.Fatal("completion command should be hidden")
		}
	}
}
