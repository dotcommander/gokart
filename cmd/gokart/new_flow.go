package main

import (
	"errors"
	"fmt"

	"github.com/dotcommander/gokart/cli"
)

func runNewRequest(req newRequest, jsonOutput bool, output *newCommandOutput) error {
	if req.VerifyOnly {
		return runNewVerifyOnlyFlow(req, jsonOutput, output)
	}

	result, err := runNewScaffoldFlow(req, jsonOutput, output)
	if err != nil {
		return err
	}
	output.Result = result

	if req.Verify {
		output.VerifyRan = true
		if err := runVerifyForRequest(req, !jsonOutput); err != nil {
			output.VerifyPassed = false
			if req.DryRun {
				return failNewCommand(fmt.Errorf("dry-run verification failed: %w", err), jsonOutput, output, errorCodeVerifyFailed, commandOutcomeFailure, exitCodeVerifyFailed)
			}
			return failNewCommand(fmt.Errorf("project generated at %s, but verification failed: %w", req.TargetDir, err), jsonOutput, output, errorCodeVerifyFailed, commandOutcomePartialSuccess, exitCodeVerifyFailed)
		}
		output.VerifyPassed = true
		if !jsonOutput {
			cli.Success("Verification passed")
		}
	}

	if !req.DryRun {
		setNewNextStep(req, jsonOutput, output)
	}

	if jsonOutput {
		return emitCommandJSON(output)
	}

	return nil
}

func runNewVerifyOnlyFlow(req newRequest, jsonOutput bool, output *newCommandOutput) error {
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
		return failNewCommand(fmt.Errorf("verification failed for %s: %w", req.TargetDir, err), jsonOutput, output, errorCodeVerifyFailed, commandOutcomeFailure, exitCodeVerifyFailed)
	}

	output.VerifyPassed = true
	if !jsonOutput {
		cli.Success("Verification passed")
	}

	if jsonOutput {
		return emitCommandJSON(output)
	}

	return nil
}

func runNewScaffoldFlow(req newRequest, jsonOutput bool, output *newCommandOutput) (*ApplyResult, error) {
	if req.Mode == "flat" && (req.UseSQLite || req.UsePostgres || req.UseAI) {
		flatWarning := "--sqlite, --postgres, and --ai flags are ignored in flat mode"
		output.Warnings = append(output.Warnings, flatWarning)
		if !jsonOutput {
			cli.Warning("%s", flatWarning)
		}
	}

	printScaffoldStart(req, jsonOutput)

	result, err := scaffoldProject(req, newApplyOptions(req))
	if err != nil {
		return nil, handleNewScaffoldError(err, jsonOutput, output)
	}

	if req.DryRun {
		if !jsonOutput {
			cli.Success("Dry run complete for %s", req.TargetDir)
		}
	} else if !jsonOutput {
		cli.Success("Project created at %s", req.TargetDir)
	}

	if !jsonOutput {
		printApplyResult(result, req.DryRun)
	}

	return result, nil
}

func handleNewScaffoldError(err error, jsonOutput bool, output *newCommandOutput) error {
	var conflictErr *ExistingFileConflictError
	if errors.As(err, &conflictErr) {
		output.Conflicts = append([]string(nil), conflictErr.Paths...)
		if !jsonOutput {
			cli.Warning("Found %d conflicting existing file(s):", len(conflictErr.Paths))
			for _, path := range conflictErr.Paths {
				cli.Dim("  conflict   %s", path)
			}
		}
		return failNewCommand(err, jsonOutput, output, errorCodeExistingFileConflict, commandOutcomeFailure, exitCodeExistingFileConflict)
	}

	var lockErr *ApplyLockError
	if errors.As(err, &lockErr) {
		return failNewCommand(err, jsonOutput, output, errorCodeTargetLocked, commandOutcomeFailure, exitCodeTargetLocked)
	}

	return failNewCommand(err, jsonOutput, output, errorCodeScaffoldFailed, commandOutcomeFailure, exitCodeScaffoldFailed)
}

func printScaffoldStart(req newRequest, jsonOutput bool) {
	if jsonOutput {
		return
	}

	verb := "Scaffolding"
	if req.DryRun {
		verb = "Dry run: planning"
	}

	cli.Info("%s %s project (%s preset): %s", verb, req.Mode, req.Preset, req.ProjectName)
}

func newApplyOptions(req newRequest) ApplyOptions {
	return ApplyOptions{
		DryRun:             req.DryRun,
		ExistingFilePolicy: req.ExistingFilePolicy,
		SkipManifest:       !req.WriteManifest,
	}
}

func scaffoldProject(req newRequest, opts ApplyOptions) (*ApplyResult, error) {
	switch req.Mode {
	case "flat":
		return scaffoldFlatFunc(req.TargetDir, req.ProjectName, req.Module, req.UseGlobal, req.IncludeExample, opts)
	case "structured":
		return scaffoldStructuredFunc(req.TargetDir, req.ProjectName, req.Module, req.UseSQLite, req.UsePostgres, req.UseAI, req.UseGlobal, req.IncludeExample, opts)
	default:
		return nil, fmt.Errorf("unsupported mode %q", req.Mode)
	}
}

func setNewNextStep(req newRequest, jsonOutput bool, output *newCommandOutput) {
	nextHint := nextStep{Dir: req.TargetDir, Command: "go", Args: []string{"mod", "tidy"}}
	output.Next = &nextHint
	nextCommand := fmt.Sprintf("cd %s && go mod tidy", shellQuote(req.TargetDir))
	output.NextCommand = nextCommand
	if !jsonOutput {
		cli.Dim("  %s", nextCommand)
	}
}
