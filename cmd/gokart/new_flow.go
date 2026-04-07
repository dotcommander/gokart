package main

import (
	"context"
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

	if !req.DryRun {
		if err := resolveNewDependencies(req, !jsonOutput); err != nil {
			if !jsonOutput {
				cli.Warning("Dependency resolution failed: %v", err)
				cli.Dim("  Run manually: cd %s && go get ./... && go mod tidy", shellQuote(req.TargetDir))
			}
			output.Warnings = append(output.Warnings, fmt.Sprintf("dependency resolution failed: %v", err))
		}
	}

	if req.Verify { //nolint:nestif // verify-on-scaffold flow, nesting is inherent
		output.VerifyRan = true
		if err := runVerifyForRequest(req, !jsonOutput); err != nil {
			output.VerifyPassed = false
			if req.DryRun {
				return failNewCommand(fmt.Errorf("dry-run verification failed: %w", err), jsonOutput, output, commandFailureInfo{Code: errorCodeVerifyFailed, Outcome: commandOutcomeFailure, ExitCode: exitCodeVerifyFailed})
			}
			return failNewCommand(fmt.Errorf("project generated at %s, but verification failed: %w", req.TargetDir, err), jsonOutput, output, commandFailureInfo{Code: errorCodeVerifyFailed, Outcome: commandOutcomePartialSuccess, ExitCode: exitCodeVerifyFailed})
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
		return failNewCommand(fmt.Errorf("verification failed for %s: %w", req.TargetDir, err), jsonOutput, output, commandFailureInfo{Code: errorCodeVerifyFailed, Outcome: commandOutcomeFailure, ExitCode: exitCodeVerifyFailed})
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
	if req.Mode == modeFlat && (req.UseSQLite || req.UsePostgres || req.UseAI || req.UseRedis) {
		return nil, failNewCommand(
			errors.New("integrations (--sqlite, --postgres, --ai, --redis) require structured mode — remove --flat or omit the integration flags"),
			jsonOutput, output, commandFailureInfo{
				Code:     errorCodeInvalidArguments,
				Outcome:  commandOutcomeFailure,
				ExitCode: exitCodeInvalidArguments,
			},
		)
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
		return failNewCommand(err, jsonOutput, output, commandFailureInfo{Code: errorCodeExistingFileConflict, Outcome: commandOutcomeFailure, ExitCode: exitCodeExistingFileConflict})
	}

	var lockErr *ApplyLockError
	if errors.As(err, &lockErr) {
		return failNewCommand(err, jsonOutput, output, commandFailureInfo{Code: errorCodeTargetLocked, Outcome: commandOutcomeFailure, ExitCode: exitCodeTargetLocked})
	}

	return failNewCommand(err, jsonOutput, output, commandFailureInfo{Code: errorCodeScaffoldFailed, Outcome: commandOutcomeFailure, ExitCode: exitCodeScaffoldFailed})
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
	case modeFlat:
		return scaffoldFlatFunc(req.TargetDir, req.ProjectName, req.Module, req.UseGlobal, req.IncludeExample, opts)
	case modeStructured:
		return scaffoldStructuredFunc(req.TargetDir, req.ProjectName, req.Module, req.UseSQLite, req.UsePostgres, req.UseAI, req.UseRedis, req.UseGlobal, req.IncludeExample, opts)
	default:
		return nil, fmt.Errorf("unsupported mode %q", req.Mode)
	}
}

func setNewNextStep(req newRequest, jsonOutput bool, output *newCommandOutput) {
	nextHint := nextStep{Dir: req.TargetDir, Command: "go", Args: []string{"build", "./..."}}
	output.Next = &nextHint
	nextCommand := fmt.Sprintf("cd %s && go build ./...", shellQuote(req.TargetDir))
	output.NextCommand = nextCommand
	if !jsonOutput {
		cli.Dim("  %s", nextCommand)
	}
}

func resolveNewDependencies(req newRequest, verbose bool) error {
	packages := []string{"github.com/dotcommander/gokart/cli@" + defaultGokartCLIVersion}
	if req.UseSQLite {
		packages = append(packages, "github.com/dotcommander/gokart/sqlite@"+defaultGokartSQLiteVersion)
	}
	if req.UsePostgres {
		packages = append(packages, "github.com/dotcommander/gokart/postgres@"+defaultGokartPostgresVersion)
	}
	if req.UseAI {
		packages = append(packages, "github.com/dotcommander/gokart/ai@"+defaultGokartAIVersion)
	}
	if req.UseRedis {
		packages = append(packages, "github.com/dotcommander/gokart/cache@"+defaultGokartCacheVersion)
	}

	ctx := context.Background()

	goGetArgs := append([]string{"get"}, packages...)
	if err := runCommand(ctx, req.TargetDir, verbose, "go", goGetArgs...); err != nil {
		return fmt.Errorf("go get: %w", err)
	}

	if err := runCommand(ctx, req.TargetDir, verbose, "go", "mod", "tidy"); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}

	return nil
}
