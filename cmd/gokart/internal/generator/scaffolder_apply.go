package generator

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Apply walks the template filesystem and renders templates to targetDir.
// Files that render to empty content (whitespace only) are skipped.
//
//nolint:gocognit // orchestration function, complexity is inherent
func Apply(fsys fs.FS, root string, targetDir string, data TemplateData, opts ApplyOptions) (result *ApplyResult, err error) {
	normalizedOpts, err := normalizeApplyOptions(opts)
	if err != nil {
		return nil, err
	}

	targetRoot, err := filepath.Abs(targetDir)
	if err != nil {
		return nil, fmt.Errorf("resolve target directory %q: %w", targetDir, err)
	}

	apply := func() error {
		plan, planErr := buildPlan(fsys, root, targetDir, data, normalizedOpts.ExistingFilePolicy)
		if planErr != nil {
			return planErr
		}

		result = collectResult(plan, normalizedOpts.DryRun)
		if normalizedOpts.DryRun {
			return nil
		}

		journal, journalErr := beginApplyJournal(targetRoot)
		if journalErr != nil {
			return journalErr
		}
		if writeErr := applyPlanWrites(plan, result, journal, targetRoot, root, normalizedOpts); writeErr != nil {
			return writeErr
		}
		if completeErr := journal.markCompleted(); completeErr != nil {
			return completeErr
		}
		return journal.cleanup()
	}

	if normalizedOpts.DryRun {
		err = apply()
		return result, err
	}
	err = withTargetMutationLock(targetRoot, apply)
	return result, err
}

//nolint:revive // params are distinct concerns
func applyPlanWrites(plan []plannedFile, result *ApplyResult, journal *applyJournal, targetRoot string, templateRoot string, opts ApplyOptions) error {
	applied := make([]rollbackAction, 0, len(plan))

	for _, file := range plan {
		switch file.Action { //nolint:exhaustive // planActionSkip and planActionUnchanged are no-ops here
		case planActionCreate:
			perm := fs.FileMode(0644)
			action := rollbackAction{
				Kind:           rollbackRemove,
				Root:           file.TargetRoot,
				Path:           file.DestPath,
				ExpectedExists: true,
				ExpectedHash:   sha256Hex(file.Rendered),
				ExpectedSize:   int64(len(file.Rendered)),
				ExpectedMode:   perm,
			}
			var err error
			applied, err = writeAndJournal(file, action, ensureCreateStateUnchanged, perm, journal, applied)
			if err != nil {
				return err
			}
			result.Created = append(result.Created, file.RelPath)
		case planActionOverwrite:
			perm := file.ExistingMode
			if perm == 0 {
				perm = 0644
			}
			action := rollbackAction{
				Kind:           rollbackRestore,
				Root:           file.TargetRoot,
				Path:           file.DestPath,
				Content:        file.Existing,
				Mode:           file.ExistingMode,
				ExpectedExists: true,
				ExpectedHash:   sha256Hex(file.Rendered),
				ExpectedSize:   int64(len(file.Rendered)),
				ExpectedMode:   perm,
			}
			var err error
			applied, err = writeAndJournal(file, action, ensureOverwriteStateUnchanged, perm, journal, applied)
			if err != nil {
				return err
			}
			result.Overwritten = append(result.Overwritten, file.RelPath)
		}
	}

	if opts.SkipManifest {
		return nil
	}

	manifestPath := filepath.Join(targetRoot, scaffoldManifestPath)
	manifestData, err := renderScaffoldManifest(templateRoot, plan, opts)
	if err != nil {
		return rollbackWithError(fmt.Errorf("render scaffold manifest: %w", err), applied, journal)
	}
	manifestRollbackAction, err := rollbackActionForPath(targetRoot, manifestPath)
	if err != nil {
		return rollbackWithError(fmt.Errorf("prepare manifest rollback action: %w", err), applied, journal)
	}
	manifestRollbackAction.ExpectedExists = true
	manifestRollbackAction.ExpectedHash = sha256Hex(manifestData)
	manifestRollbackAction.ExpectedSize = int64(len(manifestData))
	manifestRollbackAction.ExpectedMode = 0644

	journalIndex, err := journal.appendAction(manifestRollbackAction)
	if err != nil {
		return rollbackWithError(fmt.Errorf("record manifest rollback intent: %w", err), applied, journal)
	}

	if err := writeScaffoldManifest(targetRoot, manifestData); err != nil {
		return rollbackWithError(fmt.Errorf("write scaffold manifest: %w", err), applied, journal)
	}

	if err := journal.markActionApplied(journalIndex); err != nil {
		return rollbackWithError(fmt.Errorf("mark manifest rollback intent applied: %w", err), append(applied, manifestRollbackAction), journal)
	}

	return nil
}

//nolint:revive // params are distinct concerns
func writeAndJournal(
	file plannedFile,
	action rollbackAction,
	ensureUnchanged func(plannedFile) error,
	perm fs.FileMode,
	journal *applyJournal,
	applied []rollbackAction,
) ([]rollbackAction, error) {
	journalIndex, err := journal.appendAction(action)
	if err != nil {
		return applied, rollbackWithError(fmt.Errorf("record rollback intent for %s: %w", file.RelPath, err), applied, journal)
	}
	if err := ensureUnchanged(file); err != nil {
		return applied, rollbackWithError(err, applied, journal)
	}
	if err := writePlannedFile(file, perm); err != nil {
		return applied, rollbackWithError(err, applied, journal)
	}
	if err := journal.markActionApplied(journalIndex); err != nil {
		return applied, rollbackWithError(fmt.Errorf("mark rollback intent applied for %s: %w", file.RelPath, err), append(applied, action), journal)
	}
	action.Mode = perm
	applied = append(applied, action)
	return applied, nil
}

func rollbackWithError(writeErr error, applied []rollbackAction, journal *applyJournal) error {
	rollbackErr := rollbackWrites(applied)
	if rollbackErr != nil {
		return fmt.Errorf("%w; rollback failed: %w", writeErr, rollbackErr)
	}

	if journal != nil {
		var finalizeErrs []error
		if err := journal.markCompleted(); err != nil {
			finalizeErrs = append(finalizeErrs, fmt.Errorf("mark journal complete: %w", err))
		}
		if err := journal.cleanup(); err != nil {
			finalizeErrs = append(finalizeErrs, fmt.Errorf("cleanup journal: %w", err))
		}
		if len(finalizeErrs) > 0 {
			return fmt.Errorf("%w; rollback completed but journal finalization failed: %w", writeErr, errors.Join(finalizeErrs...))
		}
	}

	return writeErr
}

func rollbackWrites(applied []rollbackAction) error {
	var errs []error

	for i := len(applied) - 1; i >= 0; i-- {
		action := applied[i]
		if action.Root != "" {
			if err := ensureNoSymlinkFromRoot(action.Root, action.Path); err != nil {
				errs = append(errs, fmt.Errorf("rollback path %s is unsafe: %w", action.Path, err))
				continue
			}
		}
		switch action.Kind {
		case rollbackRemove:
			if err := os.Remove(action.Path); err != nil && !os.IsNotExist(err) {
				errs = append(errs, fmt.Errorf("remove %s: %w", action.Path, err))
			}
		case rollbackRestore:
			if err := writeFileAtomic(action.Path, action.Content, action.Mode); err != nil {
				errs = append(errs, fmt.Errorf("restore %s: %w", action.Path, err))
			}
		}
	}

	if len(errs) == 0 {
		return nil
	}

	return errors.Join(errs...)
}

func rollbackActionForPath(targetRoot, path string) (rollbackAction, error) {
	if err := ensureNoSymlinkFromRoot(targetRoot, path); err != nil {
		return rollbackAction{}, fmt.Errorf("destination path is unsafe: %w", err)
	}

	info, err := assertRegularFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return rollbackAction{Kind: rollbackRemove, Root: targetRoot, Path: path}, nil
		}
		var nrf *notRegularFileError
		if errors.As(err, &nrf) {
			return rollbackAction{}, fmt.Errorf("destination path %s %s", filepath.ToSlash(path), nrf.Reason)
		}
		return rollbackAction{}, fmt.Errorf("inspect destination path: %w", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return rollbackAction{}, fmt.Errorf("read destination path: %w", err)
	}

	mode := info.Mode().Perm()
	if mode == 0 {
		mode = 0644
	}

	return rollbackAction{
		Kind:    rollbackRestore,
		Root:    targetRoot,
		Path:    path,
		Content: content,
		Mode:    mode,
	}, nil
}
