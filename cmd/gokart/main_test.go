package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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
	t.Parallel()
	cmd := newNewCommandForTest()
	mustSetFlagTrue(t, cmd, "dry-run")
	mustSetFlagTrue(t, cmd, "no-verify")
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
	t.Parallel()
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
	t.Parallel()
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

func TestValidateTargetDirRejectsNonEmptyWithFailPolicy(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "existing.txt"), []byte("data"), 0644); err != nil {
		t.Fatalf("create test file: %v", err)
	}

	err := validateTargetDir(dir, ExistingFilePolicyFail)
	if err == nil {
		t.Fatal("expected error for non-empty directory with fail policy")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("unexpected error message: %v", err)
	}

	// --force should pass
	if err := validateTargetDir(dir, ExistingFilePolicyOverwrite); err != nil {
		t.Fatalf("expected overwrite policy to pass: %v", err)
	}

	// Empty dir should pass even with fail policy
	emptyDir := t.TempDir()
	if err := validateTargetDir(emptyDir, ExistingFilePolicyFail); err != nil {
		t.Fatalf("expected empty dir to pass: %v", err)
	}

	// Non-existent dir should pass
	if err := validateTargetDir(filepath.Join(t.TempDir(), "missing"), ExistingFilePolicyFail); err != nil {
		t.Fatalf("expected missing dir to pass: %v", err)
	}
}
