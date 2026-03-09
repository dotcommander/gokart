package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dotcommander/gokart/cli"
	"github.com/spf13/cobra"
)

type addRequest struct {
	Dir           string
	Integrations  []string
	DryRun        bool
	Force         bool
	Verify        bool
	VerifyTimeout time.Duration
	JSONOutput    bool
}

type addPlan struct {
	Manifest      *scaffoldManifest
	ToAdd         []string
	Data          TemplateData
	RenderedFiles map[string][]byte
	RenderedPaths []string
}

func wrapAddFlowError(err error, code commandErrorCode, exitCode int) error {
	if err == nil {
		return nil
	}
	return &addFlowError{Err: err, Code: code, ExitCode: exitCode}
}

func buildAddRequest(cmd *cobra.Command, integrations []string) (addRequest, error) {
	dryRun, _ := cmd.Flags().GetBool(addFlagDryRun)
	force, _ := cmd.Flags().GetBool(addFlagForce)
	jsonOutput, _ := cmd.Flags().GetBool(addFlagJSON)
	verify, _ := cmd.Flags().GetBool(addFlagVerify)
	verifyTimeout, _ := cmd.Flags().GetDuration(addFlagVerifyTimeout)

	dir, err := os.Getwd()
	if err != nil {
		return addRequest{}, fmt.Errorf("get working directory: %w", err)
	}

	return addRequest{
		Dir:           dir,
		Integrations:  integrations,
		DryRun:        dryRun,
		Force:         force,
		Verify:        verify,
		VerifyTimeout: verifyTimeout,
		JSONOutput:    jsonOutput,
	}, nil
}

func collectAddIntegrations(args []string) ([]string, error) {
	seen := make(map[string]bool, len(args))
	requested := make([]string, 0, len(args))
	for _, arg := range args {
		name := strings.ToLower(strings.TrimSpace(arg))
		if !validIntegrations[name] {
			return nil, fmt.Errorf("unknown integration: %s (valid: sqlite, postgres, ai)", name)
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		requested = append(requested, name)
	}

	return requested, nil
}

func planAddChanges(req addRequest, output *addCommandOutput) (*addPlan, error) {
	manifest, err := readAddManifest(req.Dir)
	if err != nil {
		return nil, wrapAddFlowError(err, errorCodeManifestNotFound, exitCodeManifestNotFound)
	}

	if isFlatProject(manifest) {
		return nil, wrapAddFlowError(fmt.Errorf("gokart add requires a structured project (flat projects don't support integrations)"), errorCodeFlatModeUnsupported, exitCodeFlatUnsupported)
	}

	goModModule, current := detectCurrentIntegrations(manifest, req.Dir)

	var toAdd []string
	for _, name := range req.Integrations {
		if integrationAlreadyEnabled(current, name) {
			output.AlreadyPresent = append(output.AlreadyPresent, name)
			continue
		}
		toAdd = append(toAdd, name)
	}
	output.Added = append(output.Added[:0], toAdd...)

	if len(toAdd) == 0 {
		return &addPlan{Manifest: manifest}, nil
	}

	data, err := inferTemplateData(manifest, req.Dir, current, toAdd, goModModule)
	if err != nil {
		return nil, wrapAddFlowError(fmt.Errorf("infer project state: %w", err), errorCodeScaffoldFailed, exitCodeScaffoldFailed)
	}

	renderedFiles, err := renderIntegrationFiles(data)
	if err != nil {
		return nil, wrapAddFlowError(fmt.Errorf("render templates: %w", err), errorCodeScaffoldFailed, exitCodeScaffoldFailed)
	}

	renderedPaths := make([]string, 0, len(renderedFiles))
	for relPath := range renderedFiles {
		renderedPaths = append(renderedPaths, relPath)
	}
	sort.Strings(renderedPaths)

	for _, relPath := range renderedPaths {
		safety := checkFileSafety(req.Dir, relPath, manifest)
		switch safety {
		case fileSafetyCreate:
			output.FilesCreated = append(output.FilesCreated, relPath)
		case fileSafetySafe:
			output.FilesOverwritten = append(output.FilesOverwritten, relPath)
		case fileSafetyConflict:
			if !req.Force {
				return nil, wrapAddFlowError(fmt.Errorf("file %s has been modified (use --force to overwrite)", relPath), errorCodeExistingFileConflict, exitCodeExistingFileConflict)
			}
			output.FilesOverwritten = append(output.FilesOverwritten, relPath)
			output.Warnings = append(output.Warnings, fmt.Sprintf("force-overwriting modified file: %s", relPath))
		}
	}

	return &addPlan{
		Manifest:      manifest,
		ToAdd:         toAdd,
		Data:          data,
		RenderedFiles: renderedFiles,
		RenderedPaths: renderedPaths,
	}, nil
}

func printAddAlreadyPresent(output addCommandOutput, jsonOutput bool) {
	if jsonOutput {
		return
	}

	for _, name := range output.AlreadyPresent {
		cli.Warning("%s already enabled", name)
	}
}

func printAddResult(req addRequest, output addCommandOutput) {
	if req.JSONOutput {
		return
	}

	if req.DryRun {
		cli.Info("Dry run: would add %s", strings.Join(output.Added, ", "))
	} else {
		cli.Success("Added %s", strings.Join(output.Added, ", "))
	}

	for _, path := range output.FilesCreated {
		cli.Dim("  create     %s", path)
	}
	for _, path := range output.FilesOverwritten {
		cli.Dim("  overwrite  %s", path)
	}
	for _, warning := range output.Warnings {
		cli.Warning("%s", warning)
	}
}

func applyAddChanges(req addRequest, plan *addPlan, output *addCommandOutput) error {
	for _, relPath := range plan.RenderedPaths {
		destPath := filepath.Join(req.Dir, relPath)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return wrapAddFlowError(fmt.Errorf("create directory for %s: %w", relPath, err), errorCodeScaffoldFailed, exitCodeScaffoldFailed)
		}
		if err := writeFileAtomic(destPath, plan.RenderedFiles[relPath], 0644); err != nil {
			return wrapAddFlowError(fmt.Errorf("write %s: %w", relPath, err), errorCodeScaffoldFailed, exitCodeScaffoldFailed)
		}
	}

	if err := addGoDependencies(req.Dir, plan.ToAdd, !req.JSONOutput); err != nil {
		return wrapAddFlowError(fmt.Errorf("add dependencies: %w", err), errorCodeScaffoldFailed, exitCodeScaffoldFailed)
	}

	if err := updateAddManifest(req.Dir, plan.Manifest, plan.Data, plan.RenderedFiles); err != nil {
		return wrapAddFlowError(fmt.Errorf("update manifest: %w", err), errorCodeScaffoldFailed, exitCodeScaffoldFailed)
	}

	if !req.JSONOutput {
		printAddResult(req, *output)
	}

	if !req.Verify {
		return nil
	}

	output.VerifyRequested = true
	if !req.JSONOutput {
		cli.Info("Verifying project...")
	}
	if err := runVerify(req.Dir, req.VerifyTimeout, !req.JSONOutput); err != nil {
		output.VerifyPassed = false
		return wrapAddFlowError(fmt.Errorf("verification failed: %w", err), errorCodeVerifyFailed, exitCodeVerifyFailed)
	}
	output.VerifyPassed = true
	if !req.JSONOutput {
		cli.Success("Verification passed")
	}

	return nil
}
