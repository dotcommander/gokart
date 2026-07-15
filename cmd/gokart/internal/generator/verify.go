package generator

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

func (s *Service) verifyCreate(ctx context.Context, req newRequest, runtime Runtime) error {
	if req.DryRun {
		return s.verifyDryRun(ctx, req, runtime)
	}
	if err := s.verify(ctx, req.TargetDir, req.VerifyTimeout, runtime); err != nil {
		return fmt.Errorf("project generated at %s, but verification failed: %w", req.TargetDir, err)
	}
	return nil
}

func (s *Service) verifyDryRun(ctx context.Context, req newRequest, runtime Runtime) error {
	tempDir, err := os.MkdirTemp("", "gokart-verify-*")
	if err != nil {
		return fmt.Errorf("create temporary directory for verification: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()
	clone := req
	clone.TargetDir = tempDir
	clone.DryRun = false
	if _, err := scaffoldProject(clone, newApplyOptions(clone), s.deps.GeneratorVersion); err != nil {
		return fmt.Errorf("prepare dry-run verification scaffold: %w", err)
	}
	if err := s.verify(ctx, tempDir, req.VerifyTimeout, runtime); err != nil {
		return fmt.Errorf("dry-run verification failed: %w", err)
	}
	return nil
}

func (s *Service) verify(ctx context.Context, targetDir string, timeout time.Duration, runtime Runtime) error {
	if timeout < 0 {
		return errors.New("verify timeout must be >= 0")
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	runtime.report(EventVerificationStart, "Verifying generated project in "+targetDir)
	if err := s.runGoCommand(ctx, targetDir, runtime, "mod", "tidy"); err != nil {
		return err
	}
	return s.runGoCommand(ctx, targetDir, runtime, "test", "./...")
}

func (s *Service) runGoCommand(ctx context.Context, dir string, runtime Runtime, args ...string) error {
	const name = "go"
	label := strings.Join(append([]string{name}, args...), " ")
	runtime.report(EventSubprocessStart, "Running: "+label)
	var captured bytes.Buffer
	stdout, stderr := io.Writer(&captured), io.Writer(&captured)
	if runtime.Verbose {
		stdout, stderr = runtime.Stdout, runtime.Stderr
	}
	err := s.deps.Runner.Run(ctx, Command{Dir: dir, Stdout: stdout, Stderr: stderr, Name: name, Args: args})
	if err == nil {
		return nil
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("%s timed out", label)
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return fmt.Errorf("%s canceled", label)
	}
	output := strings.TrimSpace(captured.String())
	if output != "" {
		const maxOutput = 4096
		if len(output) > maxOutput {
			output = output[:maxOutput] + "\n... (truncated)"
		}
		return fmt.Errorf("%s failed: %w\n%s", label, err, output)
	}
	return fmt.Errorf("%s failed: %w", label, err)
}
