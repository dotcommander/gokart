package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dotcommander/gokart/cli"
)

func runVerifyForRequest(req newRequest, verbose bool) error {
	if req.VerifyOnly {
		return verifyProjectFunc(req.TargetDir, req.VerifyTimeout, verbose)
	}

	if req.DryRun {
		return runDryRunVerify(req, verbose)
	}

	return verifyProjectFunc(req.TargetDir, req.VerifyTimeout, verbose)
}

func runDryRunVerify(req newRequest, verbose bool) error {
	tempDir, err := os.MkdirTemp("", "gokart-verify-*")
	if err != nil {
		return fmt.Errorf("create temporary directory for verification: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }() //nolint:errcheck // cleanup of temp dir, error is meaningless

	if verbose {
		cli.Info("Verifying dry-run scaffold in %s", tempDir)
	}

	opts := ApplyOptions{DryRun: false, ExistingFilePolicy: ExistingFilePolicyFail, SkipManifest: !req.WriteManifest}

	switch req.Mode {
	case modeFlat:
		if _, err := scaffoldFlatFunc(tempDir, req.ProjectName, req.Module, req.UseGlobal, req.IncludeExample, opts); err != nil {
			return fmt.Errorf("prepare dry-run verification scaffold: %w", err)
		}
	case modeStructured:
		if _, err := scaffoldStructuredFunc(tempDir, req.ProjectName, req.Module, req.UseSQLite, req.UsePostgres, req.UseAI, req.UseRedis, req.UseGlobal, req.IncludeExample, opts); err != nil {
			return fmt.Errorf("prepare dry-run verification scaffold: %w", err)
		}
	default:
		return fmt.Errorf("unsupported mode %q", req.Mode)
	}

	return verifyProjectFunc(tempDir, req.VerifyTimeout, verbose)
}

func runVerify(targetDir string, timeout time.Duration, verbose bool) error {
	if timeout < 0 {
		return errors.New("verify timeout must be >= 0")
	}

	if verbose {
		if timeout > 0 {
			cli.Info("Verifying generated project in %s (timeout %s)", targetDir, timeout)
		} else {
			cli.Info("Verifying generated project in %s", targetDir)
		}
	}

	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	if err := runCommand(ctx, targetDir, verbose, "go", "mod", "tidy"); err != nil {
		return err
	}

	if err := runCommand(ctx, targetDir, verbose, "go", "test", "./..."); err != nil {
		return err
	}

	return nil
}

//nolint:unparam // general-purpose command runner
func runCommand(ctx context.Context, dir string, verbose bool, name string, args ...string) error {
	commandLabel := strings.Join(append([]string{name}, args...), " ")

	if verbose {
		cli.Info("Running: %s", commandLabel)
	}

	execCmd := exec.CommandContext(ctx, name, args...)
	execCmd.Dir = dir

	var commandOutput bytes.Buffer
	if verbose {
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
	} else {
		execCmd.Stdout = &commandOutput
		execCmd.Stderr = &commandOutput
	}

	if err := execCmd.Run(); err != nil {
		return formatCommandError(ctx, err, commandLabel, verbose, &commandOutput)
	}

	return nil
}

func formatCommandError(ctx context.Context, err error, label string, verbose bool, captured *bytes.Buffer) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("%s timed out", label)
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return fmt.Errorf("%s canceled", label)
	}

	if !verbose {
		output := strings.TrimSpace(captured.String())
		if output != "" {
			const maxOutput = 4096
			if len(output) > maxOutput {
				output = output[:maxOutput] + "\n... (truncated)"
			}
			return fmt.Errorf("%s failed: %w\n%s", label, err, output)
		}
	}

	return fmt.Errorf("%s failed: %w", label, err)
}
