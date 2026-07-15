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

func (s *Service) prepareDryRun(ctx context.Context, req newRequest, runtime Runtime, checks *[]CheckResult) (dependencyErr, verifyErr error) {
	tempDir, err := os.MkdirTemp("", "gokart-verify-*")
	if err != nil {
		return fmt.Errorf("create temporary directory for dry-run preparation: %w", err), nil
	}
	defer func() { _ = os.RemoveAll(tempDir) }()
	clone := req
	clone.TargetDir = tempDir
	clone.DryRun = false
	if _, err := scaffoldProject(clone, newApplyOptions(clone), s.deps.GeneratorVersion); err != nil {
		return fmt.Errorf("prepare temporary dry-run scaffold: %w", err), nil
	}
	if err := s.resolveDependencies(ctx, clone, runtime, checks); err != nil {
		return fmt.Errorf("dry-run dependency preparation failed: %w", err), nil
	}
	if !req.Verify {
		return nil, nil
	}
	if err := s.verify(ctx, tempDir, runtime, verificationPlan{Timeout: req.VerifyTimeout, Build: true}, checks); err != nil {
		return nil, fmt.Errorf("dry-run verification failed: %w", err)
	}
	return nil, nil
}

type verificationPlan struct {
	Timeout time.Duration
	Tidy    bool
	Build   bool
}

func (s *Service) verify(ctx context.Context, targetDir string, runtime Runtime, plan verificationPlan, checks *[]CheckResult) error {
	if plan.Timeout < 0 {
		return errors.New("verify timeout must be >= 0")
	}
	if plan.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, plan.Timeout)
		defer cancel()
	}
	runtime.report(EventVerificationStart, "Verifying generated project in "+targetDir)
	if plan.Tidy {
		if err := s.runCheckedGoCommand(ctx, targetDir, runtime, checks, "mod", "tidy"); err != nil {
			return err
		}
	}
	if err := s.runCheckedGoCommand(ctx, targetDir, runtime, checks, "test", "./..."); err != nil {
		return err
	}
	if plan.Build {
		return s.runCheckedGoCommand(ctx, targetDir, runtime, checks, "build", "./...")
	}
	return nil
}

func (s *Service) runCheckedGoCommand(ctx context.Context, dir string, runtime Runtime, checks *[]CheckResult, args ...string) error {
	command := "go " + strings.Join(args, " ")
	err := s.runGoCommand(ctx, dir, runtime, args...)
	status := "passed"
	if err != nil {
		status = "failed"
	}
	*checks = append(*checks, CheckResult{Command: command, Status: status})
	return err
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
