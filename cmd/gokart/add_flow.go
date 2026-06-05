package main

import (
	"context"
	"errors"
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
			return nil, fmt.Errorf("unknown integration: %s (valid: sqlite, postgres, ai, redis)", name)
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

	// Warn (non-fatal) on template-version skew: a project scaffolded by an
	// older gokart may not match templates rendered by the running version.
	if manifest.GeneratorVersion != "" && manifest.GeneratorVersion != gokartVersion {
		cli.Warning("project was scaffolded by gokart %s but you are running %s; templates may not match the original project layout", manifest.GeneratorVersion, gokartVersion)
	}

	if isFlatProject(manifest) {
		return nil, wrapAddFlowError(errors.New("gokart add requires a structured project (flat projects don't support integrations)"), errorCodeFlatModeUnsupported, exitCodeFlatUnsupported)
	}

	goModModule, current := detectCurrentIntegrations(manifest, req.Dir)

	var toAdd []string
	for _, name := range req.Integrations {
		if integrationEnabled(current, name) {
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
			output.Warnings = append(output.Warnings, "force-overwriting modified file: "+relPath)
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

func printAddResult(req addRequest, output addCommandOutput) {
	if req.JSONOutput {
		return
	}

	for _, name := range output.AlreadyPresent {
		cli.Warning("%s already enabled", name)
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

func applyAddChanges(ctx context.Context, req addRequest, plan *addPlan, output *addCommandOutput) error {
	journal, applied, err := applyAddFileWrites(req, plan)
	if err != nil {
		return err
	}

	depActions, err := journalDependencyFiles(req.Dir, journal)
	if err != nil {
		err := rollbackWithError(fmt.Errorf("prepare dependency rollback: %w", err), applied, journal)
		return wrapAddFlowError(err, errorCodeScaffoldFailed, exitCodeScaffoldFailed)
	}

	// From here on, any failure must revert every file written above.
	depErr := addGoDependenciesFunc(ctx, req.Dir, plan.ToAdd, !req.JSONOutput)
	applied, err = markDependencyFilesApplied(depActions, journal, applied)
	if depErr != nil {
		if err != nil {
			depErr = fmt.Errorf("%w; record dependency rollback state: %v", depErr, err)
		}
		err := rollbackWithError(fmt.Errorf("add dependencies: %w", depErr), applied, journal)
		return wrapAddFlowError(err, errorCodeScaffoldFailed, exitCodeScaffoldFailed)
	}
	if err != nil {
		err := rollbackWithError(fmt.Errorf("record dependency rollback state: %w", err), applied, journal)
		return wrapAddFlowError(err, errorCodeScaffoldFailed, exitCodeScaffoldFailed)
	}

	if mfErr := updateAddManifestFunc(req.Dir, plan.Manifest, plan.Data, plan.RenderedFiles); mfErr != nil {
		err := rollbackWithError(fmt.Errorf("update manifest: %w", mfErr), applied, journal)
		return wrapAddFlowError(err, errorCodeScaffoldFailed, exitCodeScaffoldFailed)
	}

	if err := journal.markCompleted(); err != nil {
		return wrapAddFlowError(fmt.Errorf("finalize add journal: %w", err), errorCodeScaffoldFailed, exitCodeScaffoldFailed)
	}
	if err := journal.cleanup(); err != nil {
		return wrapAddFlowError(fmt.Errorf("cleanup add journal: %w", err), errorCodeScaffoldFailed, exitCodeScaffoldFailed)
	}

	if !req.JSONOutput {
		printAddResult(req, *output)
	}

	if !req.Verify {
		return nil
	}

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

// applyAddFileWrites writes every rendered integration file under a recovery
// journal so a later failure (dependency install, manifest update) can revert
// the whole set. It mirrors applyPlanWrites in scaffolder_apply.go: each file is
// journaled BEFORE it is written, and a write failure rolls back what landed so
// far. rollbackActionForPath distinguishes create (file absent -> remove on
// rollback) from overwrite (file present -> restore original content+mode).
func applyAddFileWrites(req addRequest, plan *addPlan) (*applyJournal, []rollbackAction, error) {
	journal, err := beginApplyJournal(req.Dir)
	if err != nil {
		return nil, nil, wrapAddFlowError(fmt.Errorf("begin add journal: %w", err), errorCodeScaffoldFailed, exitCodeScaffoldFailed)
	}

	applied := make([]rollbackAction, 0, len(plan.RenderedPaths))
	for _, relPath := range plan.RenderedPaths {
		destPath := filepath.Join(req.Dir, relPath)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			rbErr := rollbackWithError(fmt.Errorf("create directory for %s: %w", relPath, err), applied, journal)
			return nil, nil, wrapAddFlowError(rbErr, errorCodeScaffoldFailed, exitCodeScaffoldFailed)
		}

		rendered := plan.RenderedFiles[relPath]
		action, err := rollbackActionForPath(req.Dir, destPath)
		if err != nil {
			rbErr := rollbackWithError(fmt.Errorf("prepare rollback for %s: %w", relPath, err), applied, journal)
			return nil, nil, wrapAddFlowError(rbErr, errorCodeScaffoldFailed, exitCodeScaffoldFailed)
		}
		action.ExpectedExists = true
		action.ExpectedHash = sha256Hex(rendered)
		action.ExpectedSize = int64(len(rendered))
		action.ExpectedMode = 0644

		idx, err := journal.appendAction(action)
		if err != nil {
			rbErr := rollbackWithError(fmt.Errorf("record rollback intent for %s: %w", relPath, err), applied, journal)
			return nil, nil, wrapAddFlowError(rbErr, errorCodeScaffoldFailed, exitCodeScaffoldFailed)
		}

		if err := writeFileAtomic(destPath, rendered, 0644); err != nil {
			rbErr := rollbackWithError(fmt.Errorf("write %s: %w", relPath, err), applied, journal)
			return nil, nil, wrapAddFlowError(rbErr, errorCodeScaffoldFailed, exitCodeScaffoldFailed)
		}

		if err := journal.markActionApplied(idx); err != nil {
			rbErr := rollbackWithError(fmt.Errorf("mark rollback intent applied for %s: %w", relPath, err), append(applied, action), journal)
			return nil, nil, wrapAddFlowError(rbErr, errorCodeScaffoldFailed, exitCodeScaffoldFailed)
		}
		action.Mode = 0644
		applied = append(applied, action)
	}

	return journal, applied, nil
}

type dependencyRollbackAction struct {
	index  int
	action rollbackAction
}

var (
	addGoDependenciesFunc = addGoDependencies
	updateAddManifestFunc = updateAddManifest
)

func journalDependencyFiles(dir string, journal *applyJournal) ([]dependencyRollbackAction, error) {
	actions := make([]dependencyRollbackAction, 0, 2)
	for _, relPath := range []string{"go.mod", "go.sum"} {
		path := filepath.Join(dir, relPath)
		action, err := rollbackActionForPath(dir, path)
		if err != nil {
			return nil, fmt.Errorf("prepare rollback for %s: %w", relPath, err)
		}
		index, err := journal.appendAction(action)
		if err != nil {
			return nil, fmt.Errorf("record rollback intent for %s: %w", relPath, err)
		}
		actions = append(actions, dependencyRollbackAction{index: index, action: action})
	}
	return actions, nil
}

func markDependencyFilesApplied(actions []dependencyRollbackAction, journal *applyJournal, applied []rollbackAction) ([]rollbackAction, error) {
	for _, dep := range actions {
		if err := updateJournalExpectedState(journal, dep.index); err != nil {
			return applied, err
		}
		if err := journal.markActionApplied(dep.index); err != nil {
			return append(applied, dep.action), err
		}
		applied = append(applied, dep.action)
	}
	return applied, nil
}

func updateJournalExpectedState(journal *applyJournal, index int) error {
	if journal == nil {
		return nil
	}
	if index < 0 || index >= len(journal.State.Actions) {
		return fmt.Errorf("journal action index %d out of range", index)
	}

	entry := &journal.State.Actions[index]
	path := filepath.Join(journal.State.TargetRoot, filepath.FromSlash(entry.RelPath))
	info, err := assertRegularFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			entry.ExpectedExists = false
			entry.ExpectedSHA256 = ""
			entry.ExpectedSize = 0
			entry.ExpectedMode = 0
			return journal.save()
		}
		return fmt.Errorf("inspect dependency file %s: %w", entry.RelPath, err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read dependency file %s: %w", entry.RelPath, err)
	}
	entry.ExpectedExists = true
	entry.ExpectedSHA256 = sha256Hex(content)
	entry.ExpectedSize = int64(len(content))
	entry.ExpectedMode = uint32(info.Mode().Perm())
	return journal.save()
}
