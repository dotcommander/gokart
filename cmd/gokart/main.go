package main

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/dotcommander/gokart/cli"
	"github.com/spf13/cobra"
)

var gokartVersion = "dev"

const (
	gokartAppName = "gokart"

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
	newFlagDB            = "db"
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
	newFlagNoVerify      = "no-verify"
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
		newFlagDB,
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
	app.AddCommand(newConfigCommand())
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
	flags.String(newFlagDB, "none", "Database backend: sqlite|postgres|none")
	flags.Bool(newFlagAI, false, "Include OpenAI client (openai-go/v3)")
	flags.Bool(newFlagRedis, false, "Include Redis cache (go-redis/v9)")
	flags.Bool(newFlagExample, false, "Include example greet command and action")
	flags.Bool(newFlagLocal, false, "Disable global config (structured only, default is global)")
	flags.Bool(newFlagGlobal, false, "Enable global config (flat only, default is local)")
	flags.String(newFlagConfigScope, configScopeAuto, "Config scope: auto|local|global")
	flags.Bool(newFlagDryRun, false, "Preview scaffold operations without writing files")
	flags.Bool(newFlagForce, false, "Overwrite ALL existing files (including user edits)")
	flags.Bool(newFlagSkipExisting, false, "Skip files that already exist; only write new/missing ones")
	flags.Bool(newFlagNoManifest, false, "Skip writing .gokart-manifest.json")
	flags.Bool(newFlagVerify, false, "Force run go mod tidy and go test ./... after generation (default-on for normal scaffolds)")
	flags.Bool(newFlagNoVerify, false, "Skip post-generation verification (overrides the default-on behavior)")
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
	return runNewRequest(cmdContext(cmd), req, jsonOutput, &output)
}
