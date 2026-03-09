package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// Apply walks the template filesystem and renders templates to targetDir.
// Files that render to empty content (whitespace only) are skipped.
func Apply(fsys fs.FS, root string, targetDir string, data TemplateData, opts ApplyOptions) (result *ApplyResult, err error) {
	normalizedOpts, err := normalizeApplyOptions(opts)
	if err != nil {
		return nil, err
	}

	targetRoot, err := filepath.Abs(targetDir)
	if err != nil {
		return nil, fmt.Errorf("resolve target directory %q: %w", targetDir, err)
	}

	var releaseLock func() error
	if !normalizedOpts.DryRun {
		releaseLock, err = acquireApplyLock(targetRoot)
		if err != nil {
			return nil, err
		}

		defer func() {
			if releaseLock == nil {
				return
			}
			if releaseErr := releaseLock(); releaseErr != nil {
				if err == nil {
					err = fmt.Errorf("release scaffolding lock: %w", releaseErr)
				} else {
					err = fmt.Errorf("%w; release scaffolding lock: %v", err, releaseErr)
				}
			}
		}()

		if err := recoverPendingJournals(targetRoot); err != nil {
			return nil, err
		}
	}

	plan, err := buildPlan(fsys, root, targetDir, data, normalizedOpts.ExistingFilePolicy)
	if err != nil {
		return nil, err
	}

	result = collectResult(plan, normalizedOpts.DryRun)
	if normalizedOpts.DryRun {
		return result, nil
	}

	journal, err := beginApplyJournal(targetRoot)
	if err != nil {
		return nil, err
	}

	if err := applyPlanWrites(plan, result, journal, targetRoot, root, normalizedOpts); err != nil {
		return nil, err
	}

	if err := journal.markCompleted(); err != nil {
		return nil, err
	}

	if err := journal.cleanup(); err != nil {
		return nil, err
	}

	return result, nil
}

func applyPlanWrites(plan []plannedFile, result *ApplyResult, journal *applyJournal, targetRoot string, templateRoot string, opts ApplyOptions) error {
	applied := make([]rollbackAction, 0, len(plan))

	for _, file := range plan {
		switch file.Action {
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
			journalIndex, err := journal.appendAction(action)
			if err != nil {
				return rollbackWithError(fmt.Errorf("record rollback intent for %s: %w", file.RelPath, err), applied, journal)
			}
			if err := ensureCreateStateUnchanged(file); err != nil {
				return rollbackWithError(err, applied, journal)
			}
			if err := writePlannedFile(file, perm); err != nil {
				return rollbackWithError(err, applied, journal)
			}
			if err := journal.markActionApplied(journalIndex); err != nil {
				return rollbackWithError(fmt.Errorf("mark rollback intent applied for %s: %w", file.RelPath, err), append(applied, action), journal)
			}
			applied = append(applied, action)
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
			journalIndex, err := journal.appendAction(action)
			if err != nil {
				return rollbackWithError(fmt.Errorf("record rollback intent for %s: %w", file.RelPath, err), applied, journal)
			}
			if err := ensureOverwriteStateUnchanged(file); err != nil {
				return rollbackWithError(err, applied, journal)
			}
			if err := writePlannedFile(file, perm); err != nil {
				return rollbackWithError(err, applied, journal)
			}
			if err := journal.markActionApplied(journalIndex); err != nil {
				return rollbackWithError(fmt.Errorf("mark rollback intent applied for %s: %w", file.RelPath, err), append(applied, action), journal)
			}
			action.Mode = perm
			applied = append(applied, action)
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

	applied = append(applied, manifestRollbackAction)

	return nil
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

	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return rollbackAction{Kind: rollbackRemove, Root: targetRoot, Path: path}, nil
		}
		return rollbackAction{}, fmt.Errorf("inspect destination path: %w", err)
	}

	if info.IsDir() {
		return rollbackAction{}, fmt.Errorf("destination path %s is a directory", filepath.ToSlash(path))
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return rollbackAction{}, fmt.Errorf("destination path %s is a symlink", filepath.ToSlash(path))
	}

	if !info.Mode().IsRegular() {
		return rollbackAction{}, fmt.Errorf("destination path %s is not a regular file", filepath.ToSlash(path))
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

func renderScaffoldManifest(templateRoot string, plan []plannedFile, opts ApplyOptions) ([]byte, error) {
	manifest, err := buildScaffoldManifest(templateRoot, plan, opts)
	if err != nil {
		return nil, err
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode scaffold manifest: %w", err)
	}

	return append(data, '\n'), nil
}

func writeScaffoldManifest(targetRoot string, manifestData []byte) error {
	manifestPath, err := safeDestinationPath(targetRoot, scaffoldManifestPath)
	if err != nil {
		return err
	}

	if err := writeFileAtomic(manifestPath, manifestData, 0644); err != nil {
		return fmt.Errorf("write manifest file %s: %w", filepath.ToSlash(manifestPath), err)
	}

	return nil
}

func buildScaffoldManifest(templateRoot string, plan []plannedFile, opts ApplyOptions) (scaffoldManifest, error) {
	files := make([]scaffoldManifestFile, 0, len(plan))

	for _, file := range plan {
		actionLabel, err := planActionLabel(file.Action)
		if err != nil {
			return scaffoldManifest{}, fmt.Errorf("build manifest for %s: %w", file.RelPath, err)
		}

		content := manifestContentForPlannedFile(file)
		mode := manifestModeForPlannedFile(file)

		files = append(files, scaffoldManifestFile{
			Path:           file.RelPath,
			Action:         actionLabel,
			TemplateSHA256: sha256Hex(file.Rendered),
			ContentSHA256:  sha256Hex(content),
			Mode:           uint32(mode),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	meta := opts.ManifestMetadata
	version := scaffoldManifestV1
	if meta != nil {
		version = scaffoldManifestV2
	}

	m := scaffoldManifest{
		Version:            version,
		Generator:          "gokart",
		TemplateRoot:       templateRoot,
		ExistingFilePolicy: opts.ExistingFilePolicy,
		Files:              files,
	}

	if meta != nil {
		m.Integrations = meta.Integrations
		m.Mode = meta.Mode
		m.Module = meta.Module
		m.UseGlobal = meta.UseGlobal
	}

	return m, nil
}

func planActionLabel(action planAction) (string, error) {
	switch action {
	case planActionCreate:
		return "create", nil
	case planActionOverwrite:
		return "overwrite", nil
	case planActionSkip:
		return "skip", nil
	case planActionUnchanged:
		return "unchanged", nil
	default:
		return "", fmt.Errorf("unknown plan action %d", action)
	}
}

func manifestContentForPlannedFile(file plannedFile) []byte {
	switch file.Action {
	case planActionCreate, planActionOverwrite:
		return file.Rendered
	case planActionSkip, planActionUnchanged:
		return file.Existing
	default:
		return file.Rendered
	}
}

func manifestModeForPlannedFile(file plannedFile) fs.FileMode {
	switch file.Action {
	case planActionCreate:
		return 0644
	case planActionOverwrite:
		if file.ExistingMode != 0 {
			return file.ExistingMode
		}
		return 0644
	case planActionSkip, planActionUnchanged:
		if file.ExistingMode != 0 {
			return file.ExistingMode
		}
		return 0644
	default:
		return 0644
	}
}
