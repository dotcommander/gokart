package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dotcommander/gokart/cli"
	"github.com/spf13/cobra"
)

const (
	gokartAppName = "gokart"
	gokartVersion = "0.1.0"

	defaultPreset = "cli"

	defaultVerifyTimeout = 5 * time.Minute

	configScopeAuto   = "auto"
	configScopeLocal  = "local"
	configScopeGlobal = "global"

	commandOutcomeSuccess        commandOutcome = "success"
	commandOutcomePartialSuccess commandOutcome = "partial_success"
	commandOutcomeFailure        commandOutcome = "failure"

	errorCodeInvalidArguments     commandErrorCode = "invalid_arguments"
	errorCodeConfigInitFailed     commandErrorCode = "config_init_failed"
	errorCodeExistingFileConflict commandErrorCode = "existing_file_conflict"
	errorCodeTargetLocked         commandErrorCode = "target_locked"
	errorCodeScaffoldFailed       commandErrorCode = "scaffold_failed"
	errorCodeVerifyFailed         commandErrorCode = "verify_failed"
	errorCodeJSONEncodeFailed     commandErrorCode = "json_encode_failed"

	exitCodeFailure              = 1
	exitCodeInvalidArguments     = 2
	exitCodeExistingFileConflict = 3
	exitCodeVerifyFailed         = 4
	exitCodeTargetLocked         = 5
	exitCodeConfigInitFailed     = 6
	exitCodeScaffoldFailed       = 7
	exitCodeJSONEncodeFailed     = 8

	newFlagFlat          = "flat"
	newFlagModule        = "module"
	newFlagSQLite        = "sqlite"
	newFlagPostgres      = "postgres"
	newFlagAI            = "ai"
	newFlagExample       = "example"
	newFlagLocal         = "local"
	newFlagGlobal        = "global"
	newFlagConfigScope   = "config-scope"
	newFlagDryRun        = "dry-run"
	newFlagForce         = "force"
	newFlagSkipExisting  = "skip-existing"
	newFlagNoManifest    = "no-manifest"
	newFlagVerify        = "verify"
	newFlagVerifyOnly    = "verify-only"
	newFlagVerifyTimeout = "verify-timeout"
	newFlagJSON          = "json"
)

var (
	projectNamePattern         = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	moduleSegmentPattern       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	verifyOnlyIgnoredFlagNames = []string{
		newFlagFlat,
		newFlagModule,
		newFlagSQLite,
		newFlagPostgres,
		newFlagAI,
		newFlagExample,
		newFlagLocal,
		newFlagGlobal,
		newFlagConfigScope,
		newFlagForce,
		newFlagSkipExisting,
		newFlagNoManifest,
	}
)

type commandOutcome string

type commandErrorCode string

type commandError struct {
	Err      error
	Code     commandErrorCode
	Outcome  commandOutcome
	ExitCode int
}

func (e *commandError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *commandError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

const logo = `
   ____       _  __          _
  / ___| ___ | |/ /__ _ _ __| |_
 | |  _ / _ \| ' // _' | '__| __|
 | |_| | (_) | . \ (_| | |  | |_
  \____|\___/|_|\_\__,_|_|   \__|
`

const rootLongDescription = logo + `
	gokart new <name> [flags]
	gokart new cli <name> [flags]
	gokart add <integration>... [flags]

  --sqlite         SQLite database (modernc.org/sqlite)
  --postgres       PostgreSQL pool (pgx/v5)
  --ai             OpenAI client (openai-go/v3)
  --example        Include example greet command/action scaffold
  --flat           Single main.go (no internal/)
  --local          No global config (structured: default is global)
  --global         Global config (flat: default is local)
  --config-scope   Config scope: auto|local|global
  --module         Custom module path
  --dry-run        Preview scaffold operations without writing files
  --force          Overwrite existing generated files
  --skip-existing  Keep existing files and write only missing ones
  --no-manifest    Skip writing .gokart-manifest.json
  --verify         Run go mod tidy and go test ./... after generation
  --verify-only    Run verification only against an existing project directory
  --verify-timeout Max duration for --verify commands (default 5m, 0 disables)
  --json           Print machine-readable JSON result`

const newCommandLong = `Create a new Go project with sensible defaults and optional integrations.

Structured mode (default) creates:
  myapp/
  ├── cmd/main.go                # Entry point
  ├── internal/commands/         # Cobra command definitions
  └── go.mod

Use --example to include greet command/action examples.

Flat mode creates a single main.go for quick scripts.`

const newCommandExample = `  # Basic structured project
  gokart new myapi

  # Explicit preset (same output as command above)
  gokart new cli myapi

  # With PostgreSQL and OpenAI
  gokart new myapi --postgres --ai

  # Include example command/action scaffold
  gokart new myapi --example

  # With SQLite for local-first CLI
  gokart new mycli --sqlite

  # Quick script (single main.go)
  gokart new script --flat

  # Preview without writing files
  gokart new myapi --dry-run

  # Overwrite existing generated files
  gokart new myapi --force

  # Generate and verify immediately
  gokart new myapi --verify

  # Verify an existing generated project without changing files
  gokart new myapi --verify-only

  # JSON output for CI tooling
  gokart new myapi --dry-run --json

  # Custom module path
  gokart new myapi --module github.com/myorg/myapi`

const rootHelpTemplate = `{{.Long}}

  gokart new myapp
  gokart new cli myapp
  gokart new myapp --postgres --ai
  gokart add sqlite ai

  gokart new --help    Full options
  gokart add --help    Add integrations
`

func main() {
	if err := run(); err != nil {
		os.Exit(exitCodeForError(err))
	}
}

func run() error {
	app := newGokartApp(gokartVersion)
	return app.Run()
}

func newGokartApp(version string) *cli.App {
	app := cli.NewApp(gokartAppName, version).
		WithDescription("Scaffold Go service projects").
		WithLongDescription(rootLongDescription)

	app.AddCommand(newNewCommand())
	app.AddCommand(newAddCommand())
	configureRootCommand(app.Root())

	return app
}

func newNewCommand() *cobra.Command {
	newCmd := &cobra.Command{
		Use:     "new <project-name> | new cli <project-name>",
		Short:   "Create a new GoKart project",
		Long:    newCommandLong,
		Example: newCommandExample,
		Args:    validateNewArgs,
		RunE:    runNewCommand,
	}

	configureNewCommandFlags(newCmd)

	return newCmd
}

func configureNewCommandFlags(cmd *cobra.Command) {
	if cmd == nil {
		return
	}

	flags := cmd.Flags()
	flags.Bool(newFlagFlat, false, "Use flat structure (single main.go)")
	flags.String(newFlagModule, "", "Go module path (defaults to project name)")
	flags.Bool(newFlagSQLite, false, "Include SQLite database wiring (modernc.org/sqlite)")
	flags.Bool(newFlagPostgres, false, "Include PostgreSQL connection pool (pgx/v5)")
	flags.Bool(newFlagAI, false, "Include OpenAI client (openai-go/v3)")
	flags.Bool(newFlagExample, false, "Include example greet command and action")
	flags.Bool(newFlagLocal, false, "Disable global config (structured only, default is global)")
	flags.Bool(newFlagGlobal, false, "Enable global config (flat only, default is local)")
	flags.String(newFlagConfigScope, configScopeAuto, "Config scope: auto|local|global")
	flags.Bool(newFlagDryRun, false, "Preview scaffold operations without writing files")
	flags.Bool(newFlagForce, false, "Overwrite existing generated files")
	flags.Bool(newFlagSkipExisting, false, "Keep existing files and only generate missing files")
	flags.Bool(newFlagNoManifest, false, "Skip writing .gokart-manifest.json")
	flags.Bool(newFlagVerify, false, "Run go mod tidy and go test ./... after generation")
	flags.Bool(newFlagVerifyOnly, false, "Run go mod tidy and go test ./... without scaffolding")
	flags.Duration(newFlagVerifyTimeout, defaultVerifyTimeout, "Maximum time for --verify commands (e.g. 2m, 30s; 0 disables timeout)")
	flags.Bool(newFlagJSON, false, "Print machine-readable JSON result")
}

func configureRootCommand(root *cobra.Command) {
	if root == nil {
		return
	}

	for _, cmd := range root.Commands() {
		if cmd.Name() == "completion" {
			cmd.Hidden = true
		}
		cli.SetStyledHelp(cmd)
	}

	root.SetHelpTemplate(rootHelpTemplate)
	wrapPersistentPreRunJSONErrors(root)
}

type newCommandOutput struct {
	Outcome            commandOutcome     `json:"outcome,omitempty"`
	ErrorCode          commandErrorCode   `json:"error_code,omitempty"`
	ExitCode           int                `json:"exit_code"`
	Preset             string             `json:"preset,omitempty"`
	Mode               string             `json:"mode,omitempty"`
	ProjectName        string             `json:"project_name,omitempty"`
	TargetDir          string             `json:"target_dir,omitempty"`
	Module             string             `json:"module,omitempty"`
	ConfigScope        string             `json:"config_scope,omitempty"`
	UseGlobal          bool               `json:"use_global"`
	DryRun             bool               `json:"dry_run"`
	WriteManifest      bool               `json:"write_manifest"`
	VerifyRequested    bool               `json:"verify_requested"`
	VerifyOnly         bool               `json:"verify_only"`
	VerifyRan          bool               `json:"verify_ran"`
	VerifyPassed       bool               `json:"verify_passed"`
	ExistingFilePolicy ExistingFilePolicy `json:"existing_file_policy,omitempty"`
	Warnings           []string           `json:"warnings,omitempty"`
	Conflicts          []string           `json:"conflicts,omitempty"`
	Result             *ApplyResult       `json:"result,omitempty"`
	Next               *nextStep          `json:"next,omitempty"`
	NextCommand        string             `json:"next_command,omitempty"`
	Error              string             `json:"error,omitempty"`
}

type nextStep struct {
	Dir     string   `json:"dir,omitempty"`
	Command string   `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
}

type newRequest struct {
	Preset             string
	Mode               string
	ProjectName        string
	TargetDir          string
	Module             string
	ConfigScope        string
	UseSQLite          bool
	UsePostgres        bool
	UseAI              bool
	IncludeExample     bool
	UseGlobal          bool
	DryRun             bool
	WriteManifest      bool
	Verify             bool
	VerifyOnly         bool
	VerifyTimeout      time.Duration
	ExistingFilePolicy ExistingFilePolicy
	Warnings           []string
}

var (
	scaffoldFlatFunc       = ScaffoldFlat
	scaffoldStructuredFunc = ScaffoldStructured
	verifyProjectFunc      = runVerify
)

func validateNewArgs(cmd *cobra.Command, args []string) error {
	_, _, err := parseNewInvocation(args)
	if err == nil {
		return nil
	}

	cmdErr := &commandError{
		Err:      err,
		Code:     errorCodeInvalidArguments,
		Outcome:  commandOutcomeFailure,
		ExitCode: exitCodeInvalidArguments,
	}

	jsonOutput, flagErr := cmd.Flags().GetBool(newFlagJSON)
	if flagErr != nil || !jsonOutput {
		return cmdErr
	}

	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	if emitErr := emitJSON(newCommandOutput{
		Outcome:   commandOutcomeFailure,
		ErrorCode: errorCodeInvalidArguments,
		ExitCode:  exitCodeInvalidArguments,
		Error:     err.Error(),
	}); emitErr != nil {
		fmt.Fprintf(os.Stderr, "failed to write JSON output: %v\n", emitErr)
	}

	return cmdErr
}

func runNewCommand(cmd *cobra.Command, args []string) error {
	jsonOutput, _ := cmd.Flags().GetBool(newFlagJSON)
	if jsonOutput {
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
	}

	req, err := buildNewRequest(cmd, args)
	output := newCommandOutput{}
	if err != nil {
		return failNewCommand(err, jsonOutput, &output, errorCodeInvalidArguments, commandOutcomeFailure, exitCodeInvalidArguments)
	}

	output = newCommandOutputFromRequest(req)
	for _, warning := range req.Warnings {
		if !jsonOutput {
			cli.Warning("%s", warning)
		}
	}

	if req.VerifyOnly {
		output.VerifyRan = true
		if !jsonOutput {
			if req.VerifyTimeout > 0 {
				cli.Info("Verification-only mode for %s (timeout %s)", req.TargetDir, req.VerifyTimeout)
			} else {
				cli.Info("Verification-only mode for %s", req.TargetDir)
			}
		}

		if err := runVerifyForRequest(req, !jsonOutput); err != nil {
			output.VerifyPassed = false
			return failNewCommand(fmt.Errorf("verification failed for %s: %w", req.TargetDir, err), jsonOutput, &output, errorCodeVerifyFailed, commandOutcomeFailure, exitCodeVerifyFailed)
		}

		output.VerifyPassed = true
		if !jsonOutput {
			cli.Success("Verification passed")
		}

		if jsonOutput {
			if err := emitJSON(output); err != nil {
				return &commandError{Err: fmt.Errorf("encode JSON output: %w", err), Code: errorCodeJSONEncodeFailed, Outcome: commandOutcomeFailure, ExitCode: exitCodeJSONEncodeFailed}
			}
		}

		return nil
	}

	opts := ApplyOptions{
		DryRun:             req.DryRun,
		ExistingFilePolicy: req.ExistingFilePolicy,
		SkipManifest:       !req.WriteManifest,
	}

	var result *ApplyResult
	if req.Mode == "flat" {
		if req.UseSQLite || req.UsePostgres || req.UseAI {
			flatWarning := "--sqlite, --postgres, and --ai flags are ignored in flat mode"
			output.Warnings = append(output.Warnings, flatWarning)
			if !jsonOutput {
				cli.Warning("%s", flatWarning)
			}
		}
		if req.DryRun && !jsonOutput {
			cli.Info("Dry run: planning flat project (%s preset): %s", req.Preset, req.ProjectName)
		} else if !jsonOutput {
			cli.Info("Scaffolding flat project (%s preset): %s", req.Preset, req.ProjectName)
		}
		result, err = scaffoldFlatFunc(req.TargetDir, req.ProjectName, req.Module, req.UseGlobal, req.IncludeExample, opts)
	} else {
		if req.DryRun && !jsonOutput {
			cli.Info("Dry run: planning structured project (%s preset): %s", req.Preset, req.ProjectName)
		} else if !jsonOutput {
			cli.Info("Scaffolding structured project (%s preset): %s", req.Preset, req.ProjectName)
		}
		result, err = scaffoldStructuredFunc(req.TargetDir, req.ProjectName, req.Module, req.UseSQLite, req.UsePostgres, req.UseAI, req.UseGlobal, req.IncludeExample, opts)
	}
	output.Mode = req.Mode
	if err != nil {
		var conflictErr *ExistingFileConflictError
		if errors.As(err, &conflictErr) {
			output.Conflicts = append([]string(nil), conflictErr.Paths...)
			if !jsonOutput {
				cli.Warning("Found %d conflicting existing file(s):", len(conflictErr.Paths))
				for _, path := range conflictErr.Paths {
					cli.Dim("  conflict   %s", path)
				}
			}
			return failNewCommand(err, jsonOutput, &output, errorCodeExistingFileConflict, commandOutcomeFailure, exitCodeExistingFileConflict)
		}

		var lockErr *ApplyLockError
		if errors.As(err, &lockErr) {
			return failNewCommand(err, jsonOutput, &output, errorCodeTargetLocked, commandOutcomeFailure, exitCodeTargetLocked)
		}

		return failNewCommand(err, jsonOutput, &output, errorCodeScaffoldFailed, commandOutcomeFailure, exitCodeScaffoldFailed)
	}
	output.Result = result

	if req.DryRun {
		if !jsonOutput {
			cli.Success("Dry run complete for %s", req.TargetDir)
		}
	} else {
		if !jsonOutput {
			cli.Success("Project created at %s", req.TargetDir)
		}
	}

	if !jsonOutput {
		printApplyResult(result, req.DryRun)
	}

	if req.Verify {
		output.VerifyRan = true
		if err := runVerifyForRequest(req, !jsonOutput); err != nil {
			output.VerifyPassed = false
			if req.DryRun {
				return failNewCommand(fmt.Errorf("dry-run verification failed: %w", err), jsonOutput, &output, errorCodeVerifyFailed, commandOutcomeFailure, exitCodeVerifyFailed)
			}
			return failNewCommand(fmt.Errorf("project generated at %s, but verification failed: %w", req.TargetDir, err), jsonOutput, &output, errorCodeVerifyFailed, commandOutcomePartialSuccess, exitCodeVerifyFailed)
		}
		output.VerifyPassed = true
		if !jsonOutput {
			cli.Success("Verification passed")
		}
	}

	if !req.DryRun {
		nextHint := nextStep{Dir: req.TargetDir, Command: "go", Args: []string{"mod", "tidy"}}
		output.Next = &nextHint
		nextCommand := fmt.Sprintf("cd %s && go mod tidy", shellQuote(req.TargetDir))
		output.NextCommand = nextCommand
		if !jsonOutput {
			cli.Dim("  %s", nextCommand)
		}
	}

	if jsonOutput {
		if err := emitJSON(output); err != nil {
			return &commandError{Err: fmt.Errorf("encode JSON output: %w", err), Code: errorCodeJSONEncodeFailed, Outcome: commandOutcomeFailure, ExitCode: exitCodeJSONEncodeFailed}
		}
	}

	return nil
}

func buildNewRequest(cmd *cobra.Command, args []string) (newRequest, error) {
	var req newRequest

	preset, projectArg, err := parseNewInvocation(args)
	if err != nil {
		return req, err
	}
	if preset != defaultPreset {
		return req, fmt.Errorf("unsupported preset %q", preset)
	}
	req.Preset = preset

	flat, _ := cmd.Flags().GetBool(newFlagFlat)
	module, _ := cmd.Flags().GetString(newFlagModule)
	req.UseSQLite, _ = cmd.Flags().GetBool(newFlagSQLite)
	req.UsePostgres, _ = cmd.Flags().GetBool(newFlagPostgres)
	req.UseAI, _ = cmd.Flags().GetBool(newFlagAI)
	req.IncludeExample, _ = cmd.Flags().GetBool(newFlagExample)
	local, _ := cmd.Flags().GetBool(newFlagLocal)
	global, _ := cmd.Flags().GetBool(newFlagGlobal)
	req.ConfigScope, _ = cmd.Flags().GetString(newFlagConfigScope)
	req.DryRun, _ = cmd.Flags().GetBool(newFlagDryRun)
	force, _ := cmd.Flags().GetBool(newFlagForce)
	skipExisting, _ := cmd.Flags().GetBool(newFlagSkipExisting)
	noManifest, _ := cmd.Flags().GetBool(newFlagNoManifest)
	req.WriteManifest = !noManifest
	req.Verify, _ = cmd.Flags().GetBool(newFlagVerify)
	req.VerifyOnly, _ = cmd.Flags().GetBool(newFlagVerifyOnly)
	req.VerifyTimeout, _ = cmd.Flags().GetDuration(newFlagVerifyTimeout)

	if req.VerifyTimeout < 0 {
		return req, fmt.Errorf("invalid --verify-timeout %s (must be >= 0)", req.VerifyTimeout)
	}

	if flat {
		req.Mode = "flat"
	} else {
		req.Mode = "structured"
	}

	projectName, targetDir, err := normalizeProjectArg(projectArg)
	if err != nil {
		return req, err
	}
	req.ProjectName = projectName
	req.TargetDir = targetDir

	if module == "" {
		module = projectName
	}
	req.Module = module

	if req.VerifyOnly {
		if req.DryRun {
			return req, fmt.Errorf("cannot combine --verify-only with --dry-run")
		}

		req.Verify = true
		req.ExistingFilePolicy = ExistingFilePolicyFail
		useGlobal, _, resolveErr := resolveUseGlobal(flat, false, false, configScopeAuto)
		if resolveErr != nil {
			return req, resolveErr
		}
		req.UseGlobal = useGlobal

		if ignored := verifyOnlyIgnoredFlags(cmd); len(ignored) > 0 {
			req.Warnings = append(req.Warnings, fmt.Sprintf("--verify-only ignores generation flags: %s", strings.Join(ignored, ", ")))
		}

		if err := requireExistingTargetDir(targetDir); err != nil {
			return req, err
		}

		return req, nil
	}

	useGlobal, warnings, err := resolveUseGlobal(flat, local, global, req.ConfigScope)
	if err != nil {
		return req, err
	}
	req.UseGlobal = useGlobal
	req.Warnings = append(req.Warnings, warnings...)

	existingPolicy, err := resolveExistingFilePolicy(force, skipExisting)
	if err != nil {
		return req, err
	}
	req.ExistingFilePolicy = existingPolicy

	if err := validateModulePath(module); err != nil {
		return req, fmt.Errorf("invalid module path %q: %w", module, err)
	}

	if err := validateTargetDir(targetDir); err != nil {
		return req, err
	}

	return req, nil
}

func verifyOnlyIgnoredFlags(cmd *cobra.Command) []string {
	if cmd == nil {
		return nil
	}

	ignored := make([]string, 0, len(verifyOnlyIgnoredFlagNames))
	for _, name := range verifyOnlyIgnoredFlagNames {
		if cmd.Flags().Changed(name) {
			ignored = append(ignored, "--"+name)
		}
	}

	return ignored
}

func newCommandOutputFromRequest(req newRequest) newCommandOutput {
	return newCommandOutput{
		Outcome:            commandOutcomeSuccess,
		ExitCode:           0,
		Preset:             req.Preset,
		Mode:               req.Mode,
		ProjectName:        req.ProjectName,
		TargetDir:          req.TargetDir,
		Module:             req.Module,
		ConfigScope:        req.ConfigScope,
		UseGlobal:          req.UseGlobal,
		DryRun:             req.DryRun,
		WriteManifest:      req.WriteManifest,
		VerifyRequested:    req.Verify,
		VerifyOnly:         req.VerifyOnly,
		ExistingFilePolicy: req.ExistingFilePolicy,
		Warnings:           append([]string(nil), req.Warnings...),
	}
}

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
	defer os.RemoveAll(tempDir)

	if verbose {
		cli.Info("Verifying dry-run scaffold in %s", tempDir)
	}

	opts := ApplyOptions{DryRun: false, ExistingFilePolicy: ExistingFilePolicyFail, SkipManifest: !req.WriteManifest}

	switch req.Mode {
	case "flat":
		if _, err := scaffoldFlatFunc(tempDir, req.ProjectName, req.Module, req.UseGlobal, req.IncludeExample, opts); err != nil {
			return fmt.Errorf("prepare dry-run verification scaffold: %w", err)
		}
	case "structured":
		if _, err := scaffoldStructuredFunc(tempDir, req.ProjectName, req.Module, req.UseSQLite, req.UsePostgres, req.UseAI, req.UseGlobal, req.IncludeExample, opts); err != nil {
			return fmt.Errorf("prepare dry-run verification scaffold: %w", err)
		}
	default:
		return fmt.Errorf("unsupported mode %q", req.Mode)
	}

	return verifyProjectFunc(tempDir, req.VerifyTimeout, verbose)
}

func failNewCommand(err error, jsonOutput bool, output *newCommandOutput, code commandErrorCode, outcome commandOutcome, exitCode int) error {
	cmdErr := &commandError{
		Err:      err,
		Code:     code,
		Outcome:  outcome,
		ExitCode: exitCode,
	}

	if jsonOutput && output != nil {
		output.Outcome = outcome
		output.ErrorCode = code
		output.ExitCode = exitCode
		output.Error = err.Error()

		var conflictErr *ExistingFileConflictError
		if errors.As(err, &conflictErr) && len(output.Conflicts) == 0 {
			output.Conflicts = append([]string(nil), conflictErr.Paths...)
		}

		if emitErr := emitJSON(output); emitErr != nil {
			fmt.Fprintf(os.Stderr, "failed to write JSON output: %v\n", emitErr)
			return &commandError{
				Err:      fmt.Errorf("%w; failed to write JSON output: %v", err, emitErr),
				Code:     errorCodeJSONEncodeFailed,
				Outcome:  commandOutcomeFailure,
				ExitCode: exitCodeJSONEncodeFailed,
			}
		}
	}

	return cmdErr
}

func emitJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(os.Stdout, string(data))
	return err
}

func wrapPersistentPreRunJSONErrors(root *cobra.Command) {
	if root == nil || root.PersistentPreRunE == nil {
		return
	}

	preRun := root.PersistentPreRunE
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if err := preRun(cmd, args); err != nil {
			return handlePersistentPreRunError(cmd, err)
		}
		return nil
	}
}

func handlePersistentPreRunError(cmd *cobra.Command, err error) error {
	var existing *commandError
	if errors.As(err, &existing) {
		return err
	}

	wrapped := &commandError{
		Err:      err,
		Code:     errorCodeConfigInitFailed,
		Outcome:  commandOutcomeFailure,
		ExitCode: exitCodeConfigInitFailed,
	}

	if !isJSONOutputEnabled(cmd) {
		return wrapped
	}

	if cmd != nil {
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
	}

	if emitErr := emitJSON(newCommandOutput{
		Outcome:   commandOutcomeFailure,
		ErrorCode: errorCodeConfigInitFailed,
		ExitCode:  exitCodeConfigInitFailed,
		Error:     err.Error(),
	}); emitErr != nil {
		fmt.Fprintf(os.Stderr, "failed to write JSON output: %v\n", emitErr)
	}

	return wrapped
}

func isJSONOutputEnabled(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}

	if cmd.Flags().Lookup(newFlagJSON) != nil {
		if enabled, err := cmd.Flags().GetBool(newFlagJSON); err == nil {
			return enabled
		}
	}

	if cmd.InheritedFlags().Lookup(newFlagJSON) != nil {
		if enabled, err := cmd.InheritedFlags().GetBool(newFlagJSON); err == nil {
			return enabled
		}
	}

	return false
}

func exitCodeForError(err error) int {
	if err == nil {
		return 0
	}

	var cmdErr *commandError
	if errors.As(err, &cmdErr) && cmdErr.ExitCode > 0 {
		return cmdErr.ExitCode
	}

	return exitCodeFailure
}

func parseNewInvocation(args []string) (preset string, projectArg string, err error) {
	switch len(args) {
	case 1:
		if strings.EqualFold(strings.TrimSpace(args[0]), defaultPreset) {
			return "", "", fmt.Errorf("missing project name: use `gokart new %s <project-name>` (or `gokart new ./%s` to create a project named %s)", defaultPreset, defaultPreset, defaultPreset)
		}
		return defaultPreset, args[0], nil
	case 2:
		preset := strings.ToLower(strings.TrimSpace(args[0]))
		if preset != defaultPreset {
			return "", "", fmt.Errorf("unknown preset %q (supported presets: %s)", args[0], defaultPreset)
		}
		return preset, args[1], nil
	default:
		return "", "", fmt.Errorf("usage: gokart new <project-name> or gokart new %s <project-name>", defaultPreset)
	}
}

func resolveUseGlobal(flat, local, global bool, configScope string) (bool, []string, error) {
	if local && global {
		return false, nil, fmt.Errorf("cannot use --local and --global together")
	}

	scope := strings.ToLower(strings.TrimSpace(configScope))
	if scope == "" {
		scope = configScopeAuto
	}

	if scope != configScopeAuto && (local || global) {
		return false, nil, fmt.Errorf("cannot combine --config-scope with --local or --global")
	}

	switch scope {
	case configScopeAuto:
		warnings := make([]string, 0, 1)
		if flat {
			if local {
				warnings = append(warnings, "--local has no effect in flat mode")
			}
			return global, warnings, nil
		}

		if global {
			warnings = append(warnings, "--global is already the default in structured mode")
		}
		return !local, warnings, nil
	case configScopeLocal:
		return false, nil, nil
	case configScopeGlobal:
		return true, nil, nil
	default:
		return false, nil, fmt.Errorf("invalid --config-scope %q (valid values: auto, local, global)", configScope)
	}
}

func resolveExistingFilePolicy(force, skipExisting bool) (ExistingFilePolicy, error) {
	if force && skipExisting {
		return "", fmt.Errorf("cannot use --force and --skip-existing together")
	}

	if force {
		return ExistingFilePolicyOverwrite, nil
	}

	if skipExisting {
		return ExistingFilePolicySkip, nil
	}

	return ExistingFilePolicyFail, nil
}

func normalizeProjectArg(projectArg string) (projectName, targetDir string, err error) {
	raw := strings.TrimSpace(projectArg)
	if raw == "" {
		return "", "", fmt.Errorf("project name is required")
	}

	cleanArg := filepath.Clean(raw)
	projectName = filepath.Base(cleanArg)

	if projectName == "." || projectName == ".." || projectName == string(filepath.Separator) || projectName == "" {
		return "", "", fmt.Errorf("invalid project name %q", projectArg)
	}

	if !projectNamePattern.MatchString(projectName) {
		return "", "", fmt.Errorf("invalid project name %q (allowed: letters, numbers, ., _, -)", projectName)
	}

	if filepath.IsAbs(raw) {
		targetDir = cleanArg
	} else {
		targetDir = filepath.Join(".", cleanArg)
	}

	return projectName, targetDir, nil
}

func validateModulePath(module string) error {
	mod := strings.TrimSpace(module)
	if mod == "" {
		return fmt.Errorf("cannot be empty")
	}

	if strings.ContainsAny(mod, " \t\r\n") {
		return fmt.Errorf("cannot contain whitespace")
	}

	if strings.HasPrefix(mod, "/") || strings.HasSuffix(mod, "/") {
		return fmt.Errorf("cannot start or end with '/'")
	}

	for _, segment := range strings.Split(mod, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf("contains invalid path segment %q", segment)
		}
		if !moduleSegmentPattern.MatchString(segment) {
			return fmt.Errorf("contains invalid path segment %q", segment)
		}
	}

	return nil
}

func validateTargetDir(targetDir string) error {
	info, err := os.Stat(targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("check target directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("target path %q exists and is not a directory", targetDir)
	}

	return nil
}

func requireExistingTargetDir(targetDir string) error {
	info, err := os.Stat(targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("target directory %q does not exist (required for --verify-only)", targetDir)
		}
		return fmt.Errorf("check target directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("target path %q exists and is not a directory", targetDir)
	}

	return nil
}

func printApplyResult(result *ApplyResult, dryRun bool) {
	if result == nil {
		return
	}

	label := "Applied"
	if dryRun {
		label = "Planned"
	}

	cli.Info("%s: %d create, %d overwrite, %d skip, %d unchanged", label, len(result.Created), len(result.Overwritten), len(result.Skipped), len(result.Unchanged))
	printResultPaths("create", result.Created)
	printResultPaths("overwrite", result.Overwritten)
	printResultPaths("skip", result.Skipped)
	printResultPaths("unchanged", result.Unchanged)
}

func printResultPaths(label string, paths []string) {
	for _, path := range paths {
		cli.Dim("  %-10s %s", label, path)
	}
}

func shellQuote(path string) string {
	if path == "" {
		return "''"
	}

	return "'" + strings.ReplaceAll(path, "'", "'\"'\"'") + "'"
}

func runVerify(targetDir string, timeout time.Duration, verbose bool) error {
	if timeout < 0 {
		return fmt.Errorf("verify timeout must be >= 0")
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
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("%s timed out", commandLabel)
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return fmt.Errorf("%s canceled", commandLabel)
		}

		if !verbose {
			output := strings.TrimSpace(commandOutput.String())
			if output != "" {
				const maxOutput = 4096
				if len(output) > maxOutput {
					output = output[:maxOutput] + "\n... (truncated)"
				}
				return fmt.Errorf("%s failed: %w\n%s", commandLabel, err, output)
			}
		}

		return fmt.Errorf("%s failed: %w", commandLabel, err)
	}

	return nil
}
