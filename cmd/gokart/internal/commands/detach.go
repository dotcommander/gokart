package commands

import (
	"fmt"
	"time"

	"github.com/dotcommander/gokart/cmd/gokart/internal/generator"
)

type detachCommand struct {
	DryRun        bool          `help:"Preview changes without writing files."`
	Force         bool          `help:"Overwrite user-modified files."`
	JSON          bool          `help:"Print machine-readable JSON result."`
	Verify        bool          `help:"Run go test ./... after detaching."`
	VerifyTimeout time.Duration `default:"5m" help:"Maximum verification time."`
	exec          *executor
}

type detachOutput struct {
	Outcome          string                  `json:"outcome,omitempty"`
	ErrorCode        generator.ErrorKind     `json:"error_code,omitempty"`
	ExitCode         int                     `json:"exit_code"`
	AlreadyDetached  bool                    `json:"already_detached"`
	FilesOverwritten []string                `json:"files_overwritten,omitempty"`
	ManifestRemoved  bool                    `json:"manifest_removed"`
	DryRun           bool                    `json:"dry_run"`
	VerifyRequested  bool                    `json:"verify_requested"`
	VerifyPassed     bool                    `json:"verify_passed"`
	Checks           []generator.CheckResult `json:"checks,omitempty"`
	Warnings         []string                `json:"warnings,omitempty"`
	Error            string                  `json:"error,omitempty"`
}

func detachOutputFrom(r generator.DetachResult) detachOutput {
	return detachOutput{AlreadyDetached: r.AlreadyDetached, FilesOverwritten: r.FilesOverwritten,
		ManifestRemoved: r.ManifestRemoved, DryRun: r.DryRun, VerifyRequested: r.VerifyRequested,
		VerifyPassed: r.VerifyPassed, Checks: r.Checks, Warnings: r.Warnings}
}

func (c *detachCommand) Run() error {
	dir, err := c.exec.deps.Getwd()
	if err != nil {
		return classify(fmt.Errorf("get working directory: %w", err))
	}
	result, err := c.exec.deps.Projects.Detach(c.exec.ctx, generator.DetachRequest{
		Dir: dir, DryRun: c.DryRun, Force: c.Force, Verify: c.Verify, VerifyTimeout: c.VerifyTimeout,
	}, c.exec.runtime())
	if c.JSON {
		return c.emitJSON(result, err)
	}
	if err != nil {
		return classify(err)
	}
	out := c.exec.deps.Stdout
	for _, warning := range result.Warnings {
		writeOutputf(out, "Warning: %s\n", warning)
	}
	switch {
	case result.AlreadyDetached:
		writeOutputf(out, "Already detached: no GoKart runtime imports found\n")
	case result.DryRun:
		writeOutputf(out, "Dry run: would rewrite %d file(s), tidy go.mod, and remove the manifest\n", len(result.FilesOverwritten))
		for _, path := range result.FilesOverwritten {
			writeOutputf(out, "  overwrite %s\n", path)
		}
	default:
		writeOutputf(out, "Detached: rewrote %d file(s) and removed the manifest\n", len(result.FilesOverwritten))
	}
	return nil
}

func (c *detachCommand) emitJSON(result generator.DetachResult, err error) error {
	out := detachOutputFrom(result)
	if err == nil {
		out.Outcome = outcomeSuccess
		return c.writeJSON(out)
	}
	classified, outcome := classifyEmission(err)
	out.Outcome, out.ErrorCode, out.ExitCode, out.Error = outcome, classified.kind, classified.exitCode, err.Error()
	if writeErr := c.writeJSON(out); writeErr != nil {
		return writeErr
	}
	classified.emitted = true
	return classified
}

func (c *detachCommand) writeJSON(out detachOutput) error {
	if err := emitJSON(c.exec.deps.Stdout, out); err != nil {
		writeDiagnostic(c.exec.deps.Stderr, err)
		return &commandError{Err: err, exitCode: 8, emitted: true}
	}
	return nil
}
