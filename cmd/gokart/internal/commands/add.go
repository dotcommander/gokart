package commands

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dotcommander/gokart/cmd/gokart/internal/generator"
)

type addCommand struct {
	Integrations  []string      `arg:"" name:"integration" help:"Integration: sqlite, postgres, ai, or redis."`
	DryRun        bool          `help:"Preview changes without writing files."`
	Force         bool          `help:"Overwrite user-modified files."`
	JSON          bool          `help:"Print machine-readable JSON result."`
	Verify        bool          `help:"Run go test ./... after adding."`
	VerifyTimeout time.Duration `default:"5m" help:"Maximum verification time."`
	exec          *executor
}

func (c *addCommand) Run() error {
	dir, err := c.exec.deps.Getwd()
	if err != nil {
		return classify(fmt.Errorf("get working directory: %w", err))
	}
	result, err := c.exec.deps.Projects.Add(c.exec.ctx, generator.AddRequest{
		Dir: dir, Integrations: c.Integrations, DryRun: c.DryRun, Force: c.Force,
		Verify: c.Verify, VerifyTimeout: c.VerifyTimeout,
	}, c.exec.runtime(c.JSON))
	if c.JSON {
		return c.emitJSON(result, err)
	}
	if err != nil {
		return classify(err)
	}
	if len(result.AlreadyPresent) > 0 {
		for _, name := range result.AlreadyPresent {
			_, _ = fmt.Fprintf(c.exec.deps.Stdout, "Warning: %s already enabled\n", name)
		}
	}
	if len(result.Added) > 0 {
		verb := "Added"
		if result.DryRun {
			verb = "Dry run: would add"
		}
		_, _ = fmt.Fprintf(c.exec.deps.Stdout, "%s %s\n", verb, strings.Join(result.Added, ", "))
	}
	return nil
}

func (c *addCommand) emitJSON(result generator.AddResult, err error) error {
	out := addOutputFrom(result)
	if err == nil {
		out.Outcome = outcomeSuccess
		return c.writeJSON(out)
	}

	classified := classify(err)
	out.Outcome, out.ErrorCode, out.ExitCode, out.Error = outcomeFailure, classified.kind, classified.exitCode, err.Error()
	var op *generator.OperationError
	if errors.As(err, &op) && op.Partial {
		out.Outcome = "partial_success"
	}
	if writeErr := c.writeJSON(out); writeErr != nil {
		return writeErr
	}
	classified.emitted = true
	return classified
}

func (c *addCommand) writeJSON(out addOutput) error {
	if err := emitJSON(c.exec.deps.Stdout, out); err != nil {
		writeDiagnostic(c.exec.deps.Stderr, err)
		return &commandError{Err: err, exitCode: 8, emitted: true}
	}
	return nil
}
