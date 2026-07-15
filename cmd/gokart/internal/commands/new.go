package commands

import (
	"errors"
	"fmt"
	"time"

	"github.com/dotcommander/gokart/cmd/gokart/internal/generator"
)

type newCommand struct {
	Args          []string      `arg:"" name:"project" help:"Project name."`
	Flat          bool          `help:"Use a single-package root layout (default)." group:"Project"`
	Structured    bool          `help:"Use a managed cmd/ and internal/ layout for later integrations." group:"Project"`
	Module        string        `help:"Go module path." group:"Project"`
	Example       bool          `help:"Include a runnable greet command and tests." group:"Project"`
	DB            string        `default:"none" enum:"sqlite,postgres,none" help:"Database backend." group:"Integrations / Configuration"`
	AI            bool          `help:"Include OpenAI client." group:"Integrations / Configuration"`
	Redis         bool          `help:"Include Redis cache." group:"Integrations / Configuration"`
	Local         bool          `help:"Use local-only config." group:"Integrations / Configuration"`
	Global        bool          `help:"Enable global config." group:"Integrations / Configuration"`
	ConfigScope   string        `default:"auto" enum:"auto,local,global" help:"Config scope." group:"Integrations / Configuration"`
	DryRun        bool          `help:"Preview scaffold operations without writing files." group:"Safety / Verification"`
	Force         bool          `help:"Overwrite all existing files." group:"Safety / Verification"`
	SkipExisting  bool          `help:"Skip files that already exist." group:"Safety / Verification"`
	NoManifest    bool          `help:"Opt a structured project out of GoKart management." group:"Safety / Verification"`
	Verify        bool          `help:"Force post-generation tests and build." group:"Safety / Verification"`
	NoVerify      bool          `help:"Skip tests and build, but still prepare dependencies." group:"Safety / Verification"`
	VerifyOnly    bool          `help:"Tidy, test, and build an existing project without scaffolding." group:"Safety / Verification"`
	VerifyTimeout time.Duration `default:"5m" help:"Maximum verification time; zero disables timeout." group:"Safety / Verification"`
	JSON          bool          `help:"Print machine-readable JSON result." group:"Automation"`
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
	runtime := c.exec.runtime()
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
		out.Outcome = outcomePartialSuccess
	}
}

func (c *newCommand) writeJSON(out createOutput) error {
	if err := emitJSON(c.exec.deps.Stdout, out); err != nil {
		writeDiagnostic(c.exec.deps.Stderr, err)
		return &commandError{Err: err, exitCode: 8, emitted: true}
	}
	return nil
}
