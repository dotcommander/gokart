package main

import (
	"fmt"
	"os"
	"regexp"
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
	newFlagRedis         = "redis"
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
		newFlagRedis,
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
  --redis          Redis cache (go-redis/v9)
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

  # With Redis cache
  gokart new myapi --redis

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
  gokart new myapp --redis
  gokart add sqlite ai

  Defaults: structured = global config · flat = local config

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
	app.AddCommand(newVersionCommand(version))
	configureRootCommand(app.Root())

	return app
}

func newNewCommand() *cobra.Command {
	newCmd := &cobra.Command{
		Use:     "new <project-name> | new cli <project-name>",
		Aliases: []string{"create", "init"},
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
	flags.Bool(newFlagRedis, false, "Include Redis cache (go-redis/v9)")
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

func newVersionCommand(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s version %s\n", gokartAppName, version)
		},
	}
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
	configureJSONCommand(cmd, jsonOutput)

	req, err := buildNewRequest(cmd, args)
	output := newCommandOutput{}
	if err != nil {
		return failNewCommand(err, jsonOutput, &output, commandFailureInfo{Code: errorCodeInvalidArguments, Outcome: commandOutcomeFailure, ExitCode: exitCodeInvalidArguments})
	}

	output = newCommandOutputFromRequest(req)
	printWarnings(jsonOutput, req.Warnings)
	return runNewRequest(req, jsonOutput, &output)
}
