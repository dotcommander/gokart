package main

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestParseNewInvocation(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantPreset string
		wantArg    string
		wantErr    bool
	}{
		{name: "legacy form", args: []string{"myapp"}, wantPreset: defaultPreset, wantArg: "myapp"},
		{name: "preset form", args: []string{"cli", "myapp"}, wantPreset: defaultPreset, wantArg: "myapp"},
		{name: "ambiguous preset only", args: []string{"cli"}, wantErr: true},
		{name: "unknown preset", args: []string{"api", "myapp"}, wantErr: true},
		{name: "missing args", args: []string{}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			preset, arg, err := parseNewInvocation(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("parseNewInvocation() error = %v", err)
			}

			if preset != tc.wantPreset || arg != tc.wantArg {
				t.Fatalf("got preset=%q arg=%q, want preset=%q arg=%q", preset, arg, tc.wantPreset, tc.wantArg)
			}
		})
	}
}

func TestResolveUseGlobal(t *testing.T) {
	tests := []struct {
		name        string
		flat        bool
		local       bool
		global      bool
		scope       string
		want        bool
		wantWarning bool
		wantErr     bool
	}{
		{name: "structured defaults global", scope: configScopeAuto, want: true},
		{name: "structured local legacy", local: true, scope: configScopeAuto, want: false},
		{name: "flat defaults local", flat: true, scope: configScopeAuto, want: false},
		{name: "flat legacy global", flat: true, global: true, scope: configScopeAuto, want: true},
		{name: "explicit local", scope: configScopeLocal, want: false},
		{name: "explicit global", scope: configScopeGlobal, want: true},
		{name: "conflicting legacy flags", local: true, global: true, scope: configScopeAuto, wantErr: true},
		{name: "scope with legacy flags", local: true, scope: configScopeGlobal, wantErr: true},
		{name: "flat local warning", flat: true, local: true, scope: configScopeAuto, wantWarning: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, warnings, err := resolveUseGlobal(tc.flat, tc.local, tc.global, tc.scope)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("resolveUseGlobal() error = %v", err)
			}

			if got != tc.want {
				t.Fatalf("resolveUseGlobal() = %v, want %v", got, tc.want)
			}

			hasWarning := len(warnings) > 0
			if hasWarning != tc.wantWarning {
				t.Fatalf("resolveUseGlobal() warnings=%v, expected warning=%v", warnings, tc.wantWarning)
			}
		})
	}
}

func TestResolveExistingFilePolicy(t *testing.T) {
	policy, err := resolveExistingFilePolicy(false, false)
	if err != nil {
		t.Fatalf("resolveExistingFilePolicy() error = %v", err)
	}
	if policy != ExistingFilePolicyFail {
		t.Fatalf("policy = %q, want %q", policy, ExistingFilePolicyFail)
	}

	policy, err = resolveExistingFilePolicy(true, false)
	if err != nil {
		t.Fatalf("resolveExistingFilePolicy(force) error = %v", err)
	}
	if policy != ExistingFilePolicyOverwrite {
		t.Fatalf("policy = %q, want %q", policy, ExistingFilePolicyOverwrite)
	}

	policy, err = resolveExistingFilePolicy(false, true)
	if err != nil {
		t.Fatalf("resolveExistingFilePolicy(skip) error = %v", err)
	}
	if policy != ExistingFilePolicySkip {
		t.Fatalf("policy = %q, want %q", policy, ExistingFilePolicySkip)
	}

	if _, err := resolveExistingFilePolicy(true, true); err == nil {
		t.Fatal("expected conflict error for --force + --skip-existing")
	}
}

func TestNormalizeProjectArg(t *testing.T) {
	name, dir, err := normalizeProjectArg("myapp")
	if err != nil {
		t.Fatalf("normalizeProjectArg() error = %v", err)
	}
	if name != "myapp" {
		t.Fatalf("name = %q, want myapp", name)
	}
	if dir != filepath.Join(".", "myapp") {
		t.Fatalf("dir = %q, want %q", dir, filepath.Join(".", "myapp"))
	}

	if _, _, err := normalizeProjectArg("."); err == nil {
		t.Fatal("expected error for invalid project name")
	}

	if _, _, err := normalizeProjectArg("bad name"); err == nil {
		t.Fatal("expected error for space in project name")
	}
}

func TestValidateModulePath(t *testing.T) {
	if err := validateModulePath("github.com/acme/myapp"); err != nil {
		t.Fatalf("validateModulePath(valid) error = %v", err)
	}

	if err := validateModulePath("myapp"); err != nil {
		t.Fatalf("validateModulePath(simple) error = %v", err)
	}

	if err := validateModulePath("github.com/acme/my app"); err == nil {
		t.Fatal("expected error for whitespace in module path")
	}

	if err := validateModulePath("github.com//acme"); err == nil {
		t.Fatal("expected error for empty module path segment")
	}
}

func TestNewGokartAppStartupContract(t *testing.T) {
	app := newGokartApp("test-version")
	root := app.Root()

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
		t.Fatalf("root help template drifted")
	}

	newCmd, _, err := root.Find([]string{"new"})
	if err != nil {
		t.Fatalf("find new command: %v", err)
	}
	if newCmd == nil {
		t.Fatal("expected new command")
	}

	if newCmd.Use != "new <project-name> | new cli <project-name>" {
		t.Fatalf("new command Use = %q", newCmd.Use)
	}

	if err := newCmd.Args(newCmd, []string{"myapp"}); err != nil {
		t.Fatalf("new command rejected valid args: %v", err)
	}

	completionVisible := false
	for _, cmd := range root.Commands() {
		if cmd.Name() == "completion" && !cmd.Hidden {
			completionVisible = true
		}
	}

	if completionVisible {
		t.Fatal("completion command should be hidden")
	}
}

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

func TestValidateNewArgs(t *testing.T) {
	cmd := newNewCommandForTest()

	if err := validateNewArgs(cmd, []string{"myapp"}); err != nil {
		t.Fatalf("validateNewArgs(valid) error = %v", err)
	}

	if err := validateNewArgs(cmd, []string{"cli"}); err == nil {
		t.Fatal("expected error for missing project name")
	} else {
		var cmdErr *commandError
		if !errors.As(err, &cmdErr) {
			t.Fatalf("expected commandError, got %T", err)
		}
		if cmdErr.ExitCode != exitCodeInvalidArguments {
			t.Fatalf("unexpected exit code: %d", cmdErr.ExitCode)
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

func TestRunNewCommandDryRunVerifyRunsAgainstTempScaffold(t *testing.T) {
	cmd := newNewCommandForTest()
	mustSetFlagTrue(t, cmd, "flat")
	mustSetFlagTrue(t, cmd, "dry-run")
	mustSetFlagTrue(t, cmd, "verify")
	mustSetFlagTrue(t, cmd, "json")

	targetDir := filepath.Join(t.TempDir(), "dryrun-app")

	originalVerify := verifyProjectFunc
	verifyCalled := false
	verifyDir := ""
	verifyProjectFunc = func(dir string, timeout time.Duration, verbose bool) error {
		verifyCalled = true
		verifyDir = dir
		if timeout != defaultVerifyTimeout {
			t.Fatalf("timeout = %s, want %s", timeout, defaultVerifyTimeout)
		}
		if _, err := os.Stat(filepath.Join(dir, "main.go")); err != nil {
			t.Fatalf("expected temporary scaffold main.go to exist: %v", err)
		}
		return nil
	}
	t.Cleanup(func() {
		verifyProjectFunc = originalVerify
	})

	var runErr error
	stdout := captureStdout(t, func() {
		runErr = runNewCommand(cmd, []string{targetDir})
	})
	if runErr != nil {
		t.Fatalf("runNewCommand() error = %v", runErr)
	}

	if !verifyCalled {
		t.Fatal("expected verify to run for --dry-run --verify")
	}
	if verifyDir == targetDir {
		t.Fatalf("verify dir %q should differ from target dir %q", verifyDir, targetDir)
	}

	if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
		t.Fatalf("expected target dir to remain absent after dry-run, stat err = %v", err)
	}
	if _, err := os.Stat(verifyDir); !os.IsNotExist(err) {
		t.Fatalf("expected temporary verify dir to be cleaned up, stat err = %v", err)
	}

	var output newCommandOutput
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("decode JSON output: %v\noutput=%q", err, stdout)
	}
	if !output.VerifyRan || !output.VerifyPassed {
		t.Fatalf("unexpected verify status: ran=%v passed=%v", output.VerifyRan, output.VerifyPassed)
	}
	if !output.DryRun {
		t.Fatal("expected dry_run=true in output")
	}
	if output.Outcome != commandOutcomeSuccess || output.ExitCode != 0 {
		t.Fatalf("unexpected outcome: outcome=%q exit_code=%d", output.Outcome, output.ExitCode)
	}
}

func TestRunNewCommandDryRunStructuredIncludesGitignore(t *testing.T) {
	cmd := newNewCommandForTest()
	mustSetFlagTrue(t, cmd, "dry-run")
	mustSetFlagTrue(t, cmd, "json")

	targetDir := filepath.Join(t.TempDir(), "structured-app")

	var runErr error
	stdout := captureStdout(t, func() {
		runErr = runNewCommand(cmd, []string{targetDir})
	})
	if runErr != nil {
		t.Fatalf("runNewCommand() error = %v", runErr)
	}

	var output newCommandOutput
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("decode JSON output: %v\noutput=%q", err, stdout)
	}

	if output.Result == nil {
		t.Fatal("expected scaffold result in JSON output")
	}

	if !containsString(output.Result.Created, ".gitignore") {
		t.Fatalf("expected .gitignore in created files, got: %#v", output.Result.Created)
	}
}

func TestRunNewCommandVerifyFailureReturnsExplicitError(t *testing.T) {
	cmd := newNewCommandForTest()
	mustSetFlagTrue(t, cmd, "flat")
	mustSetFlagTrue(t, cmd, "verify")
	mustSetFlagTrue(t, cmd, "json")

	targetDir := filepath.Join(t.TempDir(), "verify-fail-app")

	originalVerify := verifyProjectFunc
	verifyProjectFunc = func(dir string, timeout time.Duration, verbose bool) error {
		return errors.New("kaboom")
	}
	t.Cleanup(func() {
		verifyProjectFunc = originalVerify
	})

	var runErr error
	stdout := captureStdout(t, func() {
		runErr = runNewCommand(cmd, []string{targetDir})
	})
	if runErr == nil {
		t.Fatal("expected verify failure error")
	}

	var output newCommandOutput
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("decode JSON output: %v\noutput=%q", err, stdout)
	}
	if !output.VerifyRan || output.VerifyPassed {
		t.Fatalf("unexpected verify status: ran=%v passed=%v", output.VerifyRan, output.VerifyPassed)
	}
	if output.Error == "" {
		t.Fatal("expected error message in JSON output")
	}
	if output.Outcome != commandOutcomePartialSuccess {
		t.Fatalf("expected partial_success outcome, got %q", output.Outcome)
	}
	if output.ErrorCode != errorCodeVerifyFailed {
		t.Fatalf("expected verify_failed error code, got %q", output.ErrorCode)
	}
	if output.ExitCode != exitCodeVerifyFailed {
		t.Fatalf("unexpected exit code: %d", output.ExitCode)
	}
}

func TestRunNewCommandVerifyOnlyRunsVerifyWithoutScaffolding(t *testing.T) {
	cmd := newNewCommandForTest()
	mustSetFlagTrue(t, cmd, "verify-only")
	mustSetFlagTrue(t, cmd, "json")

	targetDir := filepath.Join(t.TempDir(), "verify-only-app")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("create target dir: %v", err)
	}

	originalVerify := verifyProjectFunc
	originalFlat := scaffoldFlatFunc
	originalStructured := scaffoldStructuredFunc

	verifyCalled := false
	verifyProjectFunc = func(dir string, timeout time.Duration, verbose bool) error {
		verifyCalled = true
		if dir != targetDir {
			t.Fatalf("verify dir = %q, want %q", dir, targetDir)
		}
		return nil
	}
	scaffoldFlatFunc = func(dir, name, module string, useGlobal, includeExample bool, opts ApplyOptions) (*ApplyResult, error) {
		t.Fatalf("scaffoldFlatFunc should not be called in --verify-only mode")
		return nil, nil
	}
	scaffoldStructuredFunc = func(dir, name, module string, useSQLite, usePostgres, useAI, useRedis, useGlobal, includeExample bool, opts ApplyOptions) (*ApplyResult, error) {
		t.Fatalf("scaffoldStructuredFunc should not be called in --verify-only mode")
		return nil, nil
	}

	t.Cleanup(func() {
		verifyProjectFunc = originalVerify
		scaffoldFlatFunc = originalFlat
		scaffoldStructuredFunc = originalStructured
	})

	var runErr error
	stdout := captureStdout(t, func() {
		runErr = runNewCommand(cmd, []string{targetDir})
	})
	if runErr != nil {
		t.Fatalf("runNewCommand() error = %v", runErr)
	}

	if !verifyCalled {
		t.Fatal("expected verify to run in --verify-only mode")
	}

	var output newCommandOutput
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("decode JSON output: %v\noutput=%q", err, stdout)
	}
	if !output.VerifyOnly || !output.VerifyRan || !output.VerifyPassed {
		t.Fatalf("unexpected verify-only output: %+v", output)
	}
	if output.Outcome != commandOutcomeSuccess || output.ExitCode != 0 {
		t.Fatalf("unexpected outcome: outcome=%q exit_code=%d", output.Outcome, output.ExitCode)
	}
}

func TestHandlePersistentPreRunErrorEmitsJSON(t *testing.T) {
	cmd := newNewCommandForTest()
	mustSetFlagTrue(t, cmd, "json")

	var gotErr error
	stdout := captureStdout(t, func() {
		gotErr = handlePersistentPreRunError(cmd, errors.New("config boom"))
	})

	var cmdErr *commandError
	if !errors.As(gotErr, &cmdErr) {
		t.Fatalf("expected commandError, got %T", gotErr)
	}
	if cmdErr.ExitCode != exitCodeConfigInitFailed {
		t.Fatalf("unexpected exit code: %d", cmdErr.ExitCode)
	}

	var output newCommandOutput
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("decode JSON output: %v\noutput=%q", err, stdout)
	}
	if output.ErrorCode != errorCodeConfigInitFailed {
		t.Fatalf("unexpected error code: %q", output.ErrorCode)
	}
	if output.ExitCode != exitCodeConfigInitFailed {
		t.Fatalf("unexpected exit code in output: %d", output.ExitCode)
	}
	if output.Outcome != commandOutcomeFailure {
		t.Fatalf("unexpected outcome: %q", output.Outcome)
	}
}

func TestExitCodeForErrorUsesCommandError(t *testing.T) {
	if got := exitCodeForError(nil); got != 0 {
		t.Fatalf("exitCodeForError(nil) = %d, want 0", got)
	}

	if got := exitCodeForError(errors.New("plain")); got != exitCodeFailure {
		t.Fatalf("exitCodeForError(plain) = %d, want %d", got, exitCodeFailure)
	}

	err := &commandError{Err: errors.New("boom"), ExitCode: exitCodeTargetLocked}
	if got := exitCodeForError(err); got != exitCodeTargetLocked {
		t.Fatalf("exitCodeForError(commandError) = %d, want %d", got, exitCodeTargetLocked)
	}
}

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
