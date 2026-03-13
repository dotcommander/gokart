package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/dotcommander/gokart/cli"
	"github.com/spf13/cobra"
)

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

func failNewCommand(err error, jsonOutput bool, output *newCommandOutput, fail commandFailureInfo) error {
	if jsonOutput && output != nil {
		var conflictErr *ExistingFileConflictError
		if errors.As(err, &conflictErr) && len(output.Conflicts) == 0 {
			output.Conflicts = append([]string(nil), conflictErr.Paths...)
		}
	}
	return emitCommandError(err, jsonOutput, output, fail)
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
