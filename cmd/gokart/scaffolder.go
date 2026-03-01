package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

// ExistingFilePolicy controls behavior when destination files already exist.
type ExistingFilePolicy string

const (
	ExistingFilePolicyFail      ExistingFilePolicy = "fail"
	ExistingFilePolicySkip      ExistingFilePolicy = "skip"
	ExistingFilePolicyOverwrite ExistingFilePolicy = "overwrite"
)

// ApplyOptions controls scaffolding behavior.
type ApplyOptions struct {
	DryRun             bool
	ExistingFilePolicy ExistingFilePolicy
}

// ApplyResult summarizes what the scaffolder did (or would do, in dry-run mode).
type ApplyResult struct {
	Created     []string `json:"created,omitempty"`
	Overwritten []string `json:"overwritten,omitempty"`
	Skipped     []string `json:"skipped,omitempty"`
	Unchanged   []string `json:"unchanged,omitempty"`
}

// ExistingFileConflictError is returned when policy is fail and one or more destination files already exist.
type ExistingFileConflictError struct {
	Paths []string
}

func (e *ExistingFileConflictError) Error() string {
	if len(e.Paths) == 0 {
		return "destination files already exist"
	}

	if len(e.Paths) == 1 {
		return fmt.Sprintf("destination file %s already exists (use --force to overwrite or --skip-existing to keep existing files)", e.Paths[0])
	}

	return fmt.Sprintf("%d destination files already exist (use --force to overwrite or --skip-existing to keep existing files)", len(e.Paths))
}

type planAction uint8

const (
	planActionCreate planAction = iota
	planActionOverwrite
	planActionSkip
	planActionUnchanged
)

type plannedFile struct {
	RelPath      string
	DestPath     string
	Rendered     []byte
	Action       planAction
	Existing     []byte
	ExistingMode fs.FileMode
}

type rollbackKind uint8

const (
	rollbackRemove rollbackKind = iota
	rollbackRestore
)

type rollbackAction struct {
	Kind    rollbackKind
	Path    string
	Content []byte
	Mode    fs.FileMode
}

// Apply walks the template filesystem and renders templates to targetDir.
// Files that render to empty content (whitespace only) are skipped.
func Apply(fsys fs.FS, root string, targetDir string, data TemplateData, opts ApplyOptions) (*ApplyResult, error) {
	normalizedOpts, err := normalizeApplyOptions(opts)
	if err != nil {
		return nil, err
	}

	plan, err := buildPlan(fsys, root, targetDir, data, normalizedOpts.ExistingFilePolicy)
	if err != nil {
		return nil, err
	}

	result := collectResult(plan, normalizedOpts.DryRun)
	if normalizedOpts.DryRun {
		return result, nil
	}

	if err := applyPlanWrites(plan, result); err != nil {
		return nil, err
	}

	return result, nil
}

func normalizeApplyOptions(opts ApplyOptions) (ApplyOptions, error) {
	if opts.ExistingFilePolicy == "" {
		opts.ExistingFilePolicy = ExistingFilePolicyFail
	}

	switch opts.ExistingFilePolicy {
	case ExistingFilePolicyFail, ExistingFilePolicySkip, ExistingFilePolicyOverwrite:
		return opts, nil
	default:
		return ApplyOptions{}, fmt.Errorf("invalid existing file policy %q", opts.ExistingFilePolicy)
	}
}

func buildPlan(fsys fs.FS, root, targetDir string, data TemplateData, policy ExistingFilePolicy) ([]plannedFile, error) {
	plan := make([]plannedFile, 0, 16)
	conflicts := make([]string, 0)

	err := fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		outPath, err := templateOutputPath(root, path)
		if err != nil {
			return err
		}

		rendered, err := renderTemplate(fsys, path, data)
		if err != nil {
			return err
		}

		if len(bytes.TrimSpace(rendered)) == 0 {
			return nil
		}

		entry := plannedFile{
			RelPath:  filepath.ToSlash(outPath),
			DestPath: filepath.Join(targetDir, outPath),
			Rendered: rendered,
		}

		info, err := os.Stat(entry.DestPath)
		if err != nil {
			if os.IsNotExist(err) {
				entry.Action = planActionCreate
				entry.ExistingMode = 0644
				plan = append(plan, entry)
				return nil
			}
			return fmt.Errorf("check destination %s: %w", entry.RelPath, err)
		}

		if info.IsDir() {
			return fmt.Errorf("destination path %s exists and is a directory", entry.RelPath)
		}

		existing, err := os.ReadFile(entry.DestPath)
		if err != nil {
			return fmt.Errorf("read destination %s: %w", entry.RelPath, err)
		}

		entry.Existing = existing
		entry.ExistingMode = info.Mode().Perm()

		if bytes.Equal(existing, rendered) {
			entry.Action = planActionUnchanged
			plan = append(plan, entry)
			return nil
		}

		switch policy {
		case ExistingFilePolicyFail:
			conflicts = append(conflicts, entry.RelPath)
			return nil
		case ExistingFilePolicySkip:
			entry.Action = planActionSkip
		case ExistingFilePolicyOverwrite:
			entry.Action = planActionOverwrite
		default:
			return fmt.Errorf("invalid existing file policy %q", policy)
		}

		plan = append(plan, entry)
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(conflicts) > 0 {
		sort.Strings(conflicts)
		return nil, &ExistingFileConflictError{Paths: conflicts}
	}

	return plan, nil
}

func templateOutputPath(root, path string) (string, error) {
	relPath, err := filepath.Rel(root, path)
	if err != nil {
		return "", err
	}

	outPath := strings.TrimSuffix(relPath, ".tmpl")
	cleanOut := filepath.Clean(outPath)

	if cleanOut == "." || cleanOut == ".." || strings.HasPrefix(cleanOut, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid output path %q from template %q", outPath, path)
	}

	return cleanOut, nil
}

func renderTemplate(fsys fs.FS, path string, data TemplateData) ([]byte, error) {
	content, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, fmt.Errorf("read template %s: %w", path, err)
	}

	funcs := template.FuncMap{
		"upper": strings.ToUpper,
	}

	tmpl, err := template.New(filepath.Base(path)).Funcs(funcs).Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("parse template %s: %w", path, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute template %s: %w", path, err)
	}

	return buf.Bytes(), nil
}

func collectResult(plan []plannedFile, dryRun bool) *ApplyResult {
	result := &ApplyResult{}

	for _, file := range plan {
		switch file.Action {
		case planActionCreate:
			if dryRun {
				result.Created = append(result.Created, file.RelPath)
			}
		case planActionOverwrite:
			if dryRun {
				result.Overwritten = append(result.Overwritten, file.RelPath)
			}
		case planActionSkip:
			result.Skipped = append(result.Skipped, file.RelPath)
		case planActionUnchanged:
			result.Unchanged = append(result.Unchanged, file.RelPath)
		}
	}

	return result
}

func applyPlanWrites(plan []plannedFile, result *ApplyResult) error {
	applied := make([]rollbackAction, 0, len(plan))

	for _, file := range plan {
		switch file.Action {
		case planActionCreate:
			if err := ensureCreateStateUnchanged(file); err != nil {
				return rollbackWithError(err, applied)
			}
			if err := writePlannedFile(file, 0644); err != nil {
				return rollbackWithError(err, applied)
			}
			applied = append(applied, rollbackAction{Kind: rollbackRemove, Path: file.DestPath})
			result.Created = append(result.Created, file.RelPath)
		case planActionOverwrite:
			if err := ensureOverwriteStateUnchanged(file); err != nil {
				return rollbackWithError(err, applied)
			}
			perm := file.ExistingMode
			if perm == 0 {
				perm = 0644
			}
			if err := writePlannedFile(file, perm); err != nil {
				return rollbackWithError(err, applied)
			}
			applied = append(applied, rollbackAction{
				Kind:    rollbackRestore,
				Path:    file.DestPath,
				Content: file.Existing,
				Mode:    perm,
			})
			result.Overwritten = append(result.Overwritten, file.RelPath)
		}
	}

	return nil
}

func ensureCreateStateUnchanged(file plannedFile) error {
	_, err := os.Stat(file.DestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("check destination %s before write: %w", file.RelPath, err)
	}

	return fmt.Errorf("destination file %s appeared after planning; rerun command (or use --force/--skip-existing)", file.RelPath)
}

func ensureOverwriteStateUnchanged(file plannedFile) error {
	info, err := os.Stat(file.DestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("destination file %s was removed after planning; rerun command", file.RelPath)
		}
		return fmt.Errorf("check destination %s before write: %w", file.RelPath, err)
	}

	if info.IsDir() {
		return fmt.Errorf("destination path %s became a directory after planning", file.RelPath)
	}

	current, err := os.ReadFile(file.DestPath)
	if err != nil {
		return fmt.Errorf("read destination %s before write: %w", file.RelPath, err)
	}

	if !bytes.Equal(current, file.Existing) {
		return fmt.Errorf("destination file %s changed after planning; rerun command", file.RelPath)
	}

	return nil
}

func writePlannedFile(file plannedFile, perm fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(file.DestPath), 0755); err != nil {
		return fmt.Errorf("create directory for %s: %w", file.RelPath, err)
	}

	if err := writeFileAtomic(file.DestPath, file.Rendered, perm); err != nil {
		return fmt.Errorf("write file %s: %w", file.RelPath, err)
	}

	return nil
}

func rollbackWithError(writeErr error, applied []rollbackAction) error {
	rollbackErr := rollbackWrites(applied)
	if rollbackErr != nil {
		return fmt.Errorf("%w; rollback failed: %w", writeErr, rollbackErr)
	}
	return writeErr
}

func rollbackWrites(applied []rollbackAction) error {
	var errs []error

	for i := len(applied) - 1; i >= 0; i-- {
		action := applied[i]
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

func writeFileAtomic(path string, data []byte, perm fs.FileMode) error {
	dir := filepath.Dir(path)

	tmpFile, err := os.CreateTemp(dir, ".gokart-*")
	if err != nil {
		return fmt.Errorf("create temp file for %s: %w", path, err)
	}

	tmpPath := tmpFile.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write temp file for %s: %w", path, err)
	}

	if err := tmpFile.Chmod(perm); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("set mode for %s: %w", path, err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file for %s: %w", path, err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		renameErr := err

		info, statErr := os.Stat(path)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				return fmt.Errorf("replace %s: %w", path, renameErr)
			}
			return fmt.Errorf("replace %s: %w (check destination: %v)", path, renameErr, statErr)
		}
		if info.IsDir() {
			return fmt.Errorf("replace %s: destination is a directory", path)
		}

		backupFile, err := os.CreateTemp(dir, ".gokart-backup-*")
		if err != nil {
			return fmt.Errorf("replace %s: %w (create backup temp file: %v)", path, renameErr, err)
		}
		backupPath := backupFile.Name()

		if err := backupFile.Close(); err != nil {
			return fmt.Errorf("replace %s: %w (close backup temp file: %v)", path, renameErr, err)
		}

		if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("replace %s: %w (prepare backup path: %v)", path, renameErr, err)
		}

		if err := os.Rename(path, backupPath); err != nil {
			return fmt.Errorf("replace %s: %w (stage existing file: %v)", path, renameErr, err)
		}

		if err := os.Rename(tmpPath, path); err != nil {
			if restoreErr := os.Rename(backupPath, path); restoreErr != nil {
				return fmt.Errorf("replace %s: %w (restore failed: %v)", path, err, restoreErr)
			}
			return fmt.Errorf("replace %s: %w", path, err)
		}

		_ = os.Remove(backupPath)
	}

	cleanup = false
	return nil
}
