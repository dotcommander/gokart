package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewCommandFlagDefaultsContract(t *testing.T) {
	cmd := newNewCommand()

	tests := []struct {
		name       string
		wantDefVal string
	}{
		{name: newFlagFlat, wantDefVal: "false"},
		{name: newFlagModule, wantDefVal: ""},
		{name: newFlagSQLite, wantDefVal: "false"},
		{name: newFlagPostgres, wantDefVal: "false"},
		{name: newFlagAI, wantDefVal: "false"},
		{name: newFlagRedis, wantDefVal: "false"},
		{name: newFlagExample, wantDefVal: "false"},
		{name: newFlagLocal, wantDefVal: "false"},
		{name: newFlagGlobal, wantDefVal: "false"},
		{name: newFlagConfigScope, wantDefVal: configScopeAuto},
		{name: newFlagDryRun, wantDefVal: "false"},
		{name: newFlagForce, wantDefVal: "false"},
		{name: newFlagSkipExisting, wantDefVal: "false"},
		{name: newFlagNoManifest, wantDefVal: "false"},
		{name: newFlagVerify, wantDefVal: "false"},
		{name: newFlagVerifyOnly, wantDefVal: "false"},
		{name: newFlagVerifyTimeout, wantDefVal: defaultVerifyTimeout.String()},
		{name: newFlagJSON, wantDefVal: "false"},
	}

	for _, tc := range tests {
		flag := cmd.Flags().Lookup(tc.name)
		if flag == nil {
			t.Fatalf("missing flag %q", tc.name)
		}
		if flag.DefValue != tc.wantDefVal {
			t.Fatalf("flag %q default = %q, want %q", tc.name, flag.DefValue, tc.wantDefVal)
		}
	}
}

func TestVerifyOnlyIgnoredFlagsListMatchesRegisteredFlags(t *testing.T) {
	cmd := newNewCommandForTest()

	for _, name := range verifyOnlyIgnoredFlagNames {
		if cmd.Flags().Lookup(name) == nil {
			t.Fatalf("verify-only ignored flag %q is not registered", name)
		}
	}
}

func TestBuildNewRequestRejectsNegativeVerifyTimeout(t *testing.T) {
	cmd := newNewCommandForTest()
	if err := cmd.Flags().Set("verify-timeout", "-1s"); err != nil {
		t.Fatalf("set verify-timeout flag: %v", err)
	}

	if _, err := buildNewRequest(cmd, []string{"myapp"}); err == nil {
		t.Fatal("expected error for negative verify timeout")
	}
}

func TestBuildNewRequestManifestDefaultsEnabled(t *testing.T) {
	cmd := newNewCommandForTest()
	req, err := buildNewRequest(cmd, []string{"myapp"})
	if err != nil {
		t.Fatalf("buildNewRequest() error = %v", err)
	}

	if !req.WriteManifest {
		t.Fatal("expected write manifest to default to true")
	}

	if req.IncludeExample {
		t.Fatal("expected include example to default to false")
	}
}

func TestBuildNewRequestExampleFlagEnablesExampleScaffold(t *testing.T) {
	cmd := newNewCommandForTest()
	mustSetFlagTrue(t, cmd, newFlagExample)

	req, err := buildNewRequest(cmd, []string{"myapp"})
	if err != nil {
		t.Fatalf("buildNewRequest() error = %v", err)
	}

	if !req.IncludeExample {
		t.Fatal("expected include example to be true when --example is set")
	}
}

func TestBuildNewRequestNoManifestDisablesManifest(t *testing.T) {
	cmd := newNewCommandForTest()
	if err := cmd.Flags().Set("no-manifest", "true"); err != nil {
		t.Fatalf("set no-manifest flag: %v", err)
	}

	req, err := buildNewRequest(cmd, []string{"myapp"})
	if err != nil {
		t.Fatalf("buildNewRequest() error = %v", err)
	}

	if req.WriteManifest {
		t.Fatal("expected write manifest to be false when --no-manifest is set")
	}
}

func TestBuildNewRequestVerifyOnlyRejectsDryRun(t *testing.T) {
	cmd := newNewCommandForTest()
	mustSetFlagTrue(t, cmd, "verify-only")
	mustSetFlagTrue(t, cmd, "dry-run")

	if _, err := buildNewRequest(cmd, []string{"myapp"}); err == nil {
		t.Fatal("expected error when --verify-only and --dry-run are combined")
	}
}

func TestBuildNewRequestVerifyOnlyRequiresExistingTarget(t *testing.T) {
	cmd := newNewCommandForTest()
	mustSetFlagTrue(t, cmd, "verify-only")

	missing := filepath.Join(t.TempDir(), "missing-app")
	if _, err := buildNewRequest(cmd, []string{missing}); err == nil {
		t.Fatal("expected error for missing target in --verify-only mode")
	}
}

func TestBuildNewRequestVerifyOnlyIgnoresGenerationConflicts(t *testing.T) {
	cmd := newNewCommandForTest()
	mustSetFlagTrue(t, cmd, "verify-only")
	mustSetFlagTrue(t, cmd, "force")
	mustSetFlagTrue(t, cmd, "skip-existing")

	targetDir := filepath.Join(t.TempDir(), "existing-app")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("create target dir: %v", err)
	}

	req, err := buildNewRequest(cmd, []string{targetDir})
	if err != nil {
		t.Fatalf("buildNewRequest() error = %v", err)
	}

	if !req.VerifyOnly || !req.Verify {
		t.Fatalf("unexpected verify flags: verifyOnly=%v verify=%v", req.VerifyOnly, req.Verify)
	}

	if len(req.Warnings) == 0 {
		t.Fatal("expected warnings for ignored generation flags")
	}
	joined := strings.Join(req.Warnings, "\n")
	if !strings.Contains(joined, "--force") || !strings.Contains(joined, "--skip-existing") {
		t.Fatalf("expected warning to include ignored flags, got: %q", joined)
	}
}

func TestShellQuote(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "simple", input: "myapp", want: "'myapp'"},
		{name: "space", input: "my app", want: "'my app'"},
		{name: "single quote", input: "a'b", want: "'a'\"'\"'b'"},
		{name: "empty", input: "", want: "''"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := shellQuote(tc.input)
			if got != tc.want {
				t.Fatalf("shellQuote(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
