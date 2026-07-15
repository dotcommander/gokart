package commands

import (
	"errors"
	"fmt"
	"time"

	"github.com/dotcommander/gokart/cmd/gokart/internal/generator"
)

type newCommand struct {
	Args          []string      `arg:"" name:"project" help:"Project name, optionally preceded by cli."`
	Flat          bool          `help:"Assert flat structure."`
	Structured    bool          `help:"Force structured project layout."`
	Module        string        `help:"Go module path."`
	DB            string        `default:"none" enum:"sqlite,postgres,none" help:"Database backend."`
	AI            bool          `help:"Include OpenAI client."`
	Redis         bool          `help:"Include Redis cache."`
	Example       bool          `help:"Include example greet command and action."`
	Local         bool          `help:"Use local-only config."`
	Global        bool          `help:"Enable global config."`
	ConfigScope   string        `default:"auto" enum:"auto,local,global" help:"Config scope."`
	DryRun        bool          `help:"Preview scaffold operations without writing files."`
	Force         bool          `help:"Overwrite all existing files."`
	SkipExisting  bool          `help:"Skip files that already exist."`
	NoManifest    bool          `help:"Skip writing the scaffold manifest."`
	Verify        bool          `help:"Force post-generation verification."`
	NoVerify      bool          `help:"Skip post-generation verification."`
	VerifyOnly    bool          `help:"Verify without scaffolding."`
	VerifyTimeout time.Duration `default:"5m" help:"Maximum verification time; zero disables timeout."`
	JSON          bool          `help:"Print machine-readable JSON result."`
	changed       map[string]bool
	exec          *executor
}

func (c *newCommand) Run() error {
	workingDir, err := c.exec.deps.Getwd()
	if err != nil {
		return classify(fmt.Errorf("get working directory: %w", err))
	}
	req := generator.CreateRequest{
		Args: c.Args, Flat: c.Flat, Structured: c.Structured, Module: c.Module,
		DB: c.DB, AI: c.AI, Redis: c.Redis, Example: c.Example, Local: c.Local,
		Global: c.Global, ConfigScope: c.ConfigScope, DryRun: c.DryRun, Force: c.Force,
		SkipExisting: c.SkipExisting, NoManifest: c.NoManifest, Verify: c.Verify,
		NoVerify: c.NoVerify, VerifyOnly: c.VerifyOnly, VerifyTimeout: c.VerifyTimeout,
		Changed: c.changed, WorkingDir: workingDir,
	}
	runtime := c.exec.runtime(c.JSON)
	result, err := c.exec.deps.Projects.Create(c.exec.ctx, req, runtime)
	if c.JSON {
		return c.emitJSON(result, err)
	}
	if err != nil {
		return classify(err)
	}
	renderCreate(c.exec.deps.Stdout, result)
	return nil
}

func (c *newCommand) emitJSON(result generator.CreateResult, err error) error {
	out := createOutputFrom(result)
	if err == nil {
		out.Outcome = outcomeSuccess
		return c.writeJSON(out)
	}

	classified := classify(err)
	out.Outcome, out.ErrorCode, out.ExitCode, out.Error = outcomeFailure, classified.kind, classified.exitCode, err.Error()
	applyCreateOperationError(&out, err)
	if writeErr := c.writeJSON(out); writeErr != nil {
		return writeErr
	}
	classified.emitted = true
	return classified
}

func applyCreateOperationError(out *createOutput, err error) {
	var op *generator.OperationError
	if !errors.As(err, &op) {
		return
	}
	out.Conflicts = append([]string(nil), op.Conflicts...)
	if op.Partial {
		out.Outcome = "partial_success"
	}
}

func (c *newCommand) writeJSON(out createOutput) error {
	if err := emitJSON(c.exec.deps.Stdout, out); err != nil {
		writeDiagnostic(c.exec.deps.Stderr, err)
		return &commandError{Err: err, exitCode: 8, emitted: true}
	}
	return nil
}
