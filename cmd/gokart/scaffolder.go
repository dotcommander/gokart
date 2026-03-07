package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"text/template"
	"time"
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
	SkipManifest       bool
	ManifestMetadata   *scaffoldManifestMetadata
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

// ApplyLockError indicates another scaffolding operation already holds the target lock.
type ApplyLockError struct {
	TargetDir string
	Reason    string
	PID       int
	CreatedAt time.Time
}

func (e *ApplyLockError) Error() string {
	if e == nil {
		return "another gokart scaffold is already running"
	}

	var parts []string
	if e.PID > 0 {
		parts = append(parts, fmt.Sprintf("PID %d", e.PID))
	}
	if !e.CreatedAt.IsZero() {
		parts = append(parts, fmt.Sprintf("age %s", time.Since(e.CreatedAt).Truncate(time.Second)))
	}
	if strings.TrimSpace(e.Reason) != "" {
		parts = append(parts, e.Reason)
	}

	detail := strings.Join(parts, ", ")
	if detail == "" {
		return fmt.Sprintf("another gokart scaffold is already running for %s", e.TargetDir)
	}
	return fmt.Sprintf("another gokart scaffold is already running for %s (%s)", e.TargetDir, detail)
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
	TargetRoot   string
	DestPath     string
	Rendered     []byte
	Action       planAction
	Existing     []byte
	ExistingMode fs.FileMode
	ExistingInfo fs.FileInfo
	Fingerprint  fileFingerprint
}

type fileFingerprint struct {
	ContentHash [sha256.Size]byte
	Mode        fs.FileMode
	Size        int64
	ModTimeNano int64
}

type rollbackKind uint8

const (
	rollbackRemove rollbackKind = iota
	rollbackRestore
)

type rollbackAction struct {
	Kind           rollbackKind
	Root           string
	Path           string
	Content        []byte
	Mode           fs.FileMode
	ExpectedExists bool
	ExpectedHash   string
	ExpectedSize   int64
	ExpectedMode   fs.FileMode
}

const (
	applyJournalVersion  = 2
	scaffoldManifestPath = ".gokart-manifest.json"
	scaffoldManifestV1   = 1
	scaffoldManifestV2   = 2
	applyLockStaleAfter  = 30 * time.Minute
)

type applyJournal struct {
	Path  string
	State applyJournalState
}

type applyJournalState struct {
	Version    int                 `json:"version"`
	ID         string              `json:"id"`
	TargetRoot string              `json:"target_root"`
	CreatedAt  time.Time           `json:"created_at"`
	Completed  bool                `json:"completed"`
	Actions    []applyJournalEntry `json:"actions,omitempty"`
}

type applyJournalEntry struct {
	Kind           string `json:"kind"`
	RelPath        string `json:"rel_path"`
	Content        []byte `json:"content,omitempty"`
	Mode           uint32 `json:"mode,omitempty"`
	Applied        bool   `json:"applied,omitempty"`
	ExpectedExists bool   `json:"expected_exists,omitempty"`
	ExpectedSHA256 string `json:"expected_sha256,omitempty"`
	ExpectedSize   int64  `json:"expected_size,omitempty"`
	ExpectedMode   uint32 `json:"expected_mode,omitempty"`
}

type scaffoldManifest struct {
	Version            int                    `json:"version"`
	Generator          string                 `json:"generator"`
	TemplateRoot       string                 `json:"template_root"`
	ExistingFilePolicy ExistingFilePolicy     `json:"existing_file_policy"`
	GeneratedAt        *time.Time             `json:"generated_at,omitempty"`
	Files              []scaffoldManifestFile `json:"files"`
	Integrations       *manifestIntegrations  `json:"integrations,omitempty"`
	Mode               string                 `json:"mode,omitempty"`
	Module             string                 `json:"module,omitempty"`
	UseGlobal          *bool                  `json:"use_global,omitempty"`
}

type scaffoldManifestFile struct {
	Path           string `json:"path"`
	Action         string `json:"action"`
	TemplateSHA256 string `json:"template_sha256"`
	ContentSHA256  string `json:"content_sha256"`
	Mode           uint32 `json:"mode"`
}

type manifestIntegrations struct {
	SQLite   bool `json:"sqlite"`
	Postgres bool `json:"postgres"`
	AI       bool `json:"ai"`
}

type scaffoldManifestMetadata struct {
	Integrations *manifestIntegrations
	Mode         string
	Module       string
	UseGlobal    *bool
}

type journalRecoveryMismatchError struct {
	RelPath string
	Reason  string
}

func (e *journalRecoveryMismatchError) Error() string {
	if e == nil {
		return ""
	}
	if e.RelPath == "" {
		return e.Reason
	}
	return fmt.Sprintf("%s: %s", e.RelPath, e.Reason)
}

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
	targetRoot, err := filepath.Abs(targetDir)
	if err != nil {
		return nil, fmt.Errorf("resolve target directory %q: %w", targetDir, err)
	}

	plan := make([]plannedFile, 0, 16)
	conflicts := make([]string, 0)

	err = fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
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

		destPath, err := safeDestinationPath(targetRoot, outPath)
		if err != nil {
			return err
		}

		entry := plannedFile{
			RelPath:    filepath.ToSlash(outPath),
			TargetRoot: targetRoot,
			DestPath:   destPath,
			Rendered:   rendered,
		}

		info, err := os.Lstat(entry.DestPath)
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

		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("destination path %s exists and is a symlink", entry.RelPath)
		}

		if !info.Mode().IsRegular() {
			return fmt.Errorf("destination path %s is not a regular file", entry.RelPath)
		}

		existing, err := os.ReadFile(entry.DestPath)
		if err != nil {
			return fmt.Errorf("read destination %s: %w", entry.RelPath, err)
		}

		entry.Existing = existing
		entry.ExistingMode = info.Mode().Perm()
		entry.ExistingInfo = info
		entry.Fingerprint = fingerprintForFile(info, existing)

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

func safeDestinationPath(targetRoot, relPath string) (string, error) {
	cleanRel := filepath.Clean(relPath)
	if cleanRel == "." || cleanRel == ".." || filepath.IsAbs(cleanRel) || strings.HasPrefix(cleanRel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid destination path %q", relPath)
	}

	destPath := filepath.Join(targetRoot, cleanRel)
	withinRoot, err := filepath.Rel(targetRoot, destPath)
	if err != nil {
		return "", fmt.Errorf("resolve destination path %q: %w", relPath, err)
	}
	if withinRoot == ".." || strings.HasPrefix(withinRoot, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("destination path %q escapes target directory", relPath)
	}

	if err := ensureNoSymlinkFromRoot(targetRoot, destPath); err != nil {
		return "", fmt.Errorf("destination path %s is unsafe: %w", filepath.ToSlash(relPath), err)
	}

	return destPath, nil
}

func ensureNoSymlinkFromRoot(rootPath, path string) error {
	rootAbs, err := filepath.Abs(rootPath)
	if err != nil {
		return fmt.Errorf("resolve absolute root path: %w", err)
	}

	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve absolute path: %w", err)
	}

	rel, err := filepath.Rel(rootAbs, pathAbs)
	if err != nil {
		return fmt.Errorf("resolve relative destination path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path %s escapes target root %s", pathAbs, rootAbs)
	}

	info, err := os.Lstat(rootAbs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("inspect path %s: %w", rootAbs, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("path component %s is a symlink", rootAbs)
	}
	if !info.IsDir() {
		return fmt.Errorf("path component %s is not a directory", rootAbs)
	}

	current := rootAbs
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if part == "" {
			continue
		}

		if part == "." {
			continue
		}

		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("inspect path %s: %w", current, err)
		}

		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("path component %s is a symlink", current)
		}
	}

	return nil
}

func fingerprintForFile(info fs.FileInfo, content []byte) fileFingerprint {
	return fileFingerprint{
		ContentHash: sha256.Sum256(content),
		Mode:        info.Mode(),
		Size:        info.Size(),
		ModTimeNano: info.ModTime().UnixNano(),
	}
}

func (f fileFingerprint) equal(other fileFingerprint) bool {
	return f.ContentHash == other.ContentHash &&
		f.Mode == other.Mode &&
		f.Size == other.Size &&
		f.ModTimeNano == other.ModTimeNano
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

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:])
}

func ensureCreateStateUnchanged(file plannedFile) error {
	if err := ensureNoSymlinkFromRoot(file.TargetRoot, file.DestPath); err != nil {
		return fmt.Errorf("destination path %s became unsafe after planning: %w", file.RelPath, err)
	}

	info, err := os.Lstat(file.DestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("check destination %s before write: %w", file.RelPath, err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("destination path %s became a symlink after planning", file.RelPath)
	}

	if info.IsDir() {
		return fmt.Errorf("destination path %s became a directory after planning", file.RelPath)
	}

	return fmt.Errorf("destination file %s appeared after planning; rerun command (or use --force/--skip-existing)", file.RelPath)
}

func ensureOverwriteStateUnchanged(file plannedFile) error {
	if err := ensureNoSymlinkFromRoot(file.TargetRoot, file.DestPath); err != nil {
		return fmt.Errorf("destination path %s became unsafe after planning: %w", file.RelPath, err)
	}

	info, err := os.Lstat(file.DestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("destination file %s was removed after planning; rerun command", file.RelPath)
		}
		return fmt.Errorf("check destination %s before write: %w", file.RelPath, err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("destination path %s became a symlink after planning", file.RelPath)
	}

	if info.IsDir() {
		return fmt.Errorf("destination path %s became a directory after planning", file.RelPath)
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf("destination path %s is not a regular file", file.RelPath)
	}

	current, err := os.ReadFile(file.DestPath)
	if err != nil {
		return fmt.Errorf("read destination %s before write: %w", file.RelPath, err)
	}

	if !fingerprintForFile(info, current).equal(file.Fingerprint) {
		return fmt.Errorf("destination file %s changed after planning; rerun command", file.RelPath)
	}

	if file.ExistingInfo != nil && !os.SameFile(info, file.ExistingInfo) {
		return fmt.Errorf("destination file %s changed after planning; rerun command", file.RelPath)
	}

	return nil
}

func writePlannedFile(file plannedFile, perm fs.FileMode) error {
	if err := ensureNoSymlinkFromRoot(file.TargetRoot, file.DestPath); err != nil {
		return fmt.Errorf("destination path %s became unsafe before write: %w", file.RelPath, err)
	}

	if err := os.MkdirAll(filepath.Dir(file.DestPath), 0755); err != nil {
		return fmt.Errorf("create directory for %s: %w", file.RelPath, err)
	}

	if err := writeFileAtomic(file.DestPath, file.Rendered, perm); err != nil {
		return fmt.Errorf("write file %s: %w", file.RelPath, err)
	}

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

func beginApplyJournal(targetRoot string) (*applyJournal, error) {
	journalDir := filepath.Join(targetRoot, ".gokart", "tx")
	if err := ensureNoSymlinkFromRoot(targetRoot, journalDir); err != nil {
		return nil, fmt.Errorf("prepare journal directory: %w", err)
	}

	if err := os.MkdirAll(journalDir, 0755); err != nil {
		return nil, fmt.Errorf("create journal directory %s: %w", filepath.ToSlash(journalDir), err)
	}

	id := fmt.Sprintf("apply-%d-%d", time.Now().UnixNano(), os.Getpid())
	path := filepath.Join(journalDir, id+".json")

	journal := &applyJournal{
		Path: path,
		State: applyJournalState{
			Version:    applyJournalVersion,
			ID:         id,
			TargetRoot: targetRoot,
			CreatedAt:  time.Now().UTC(),
			Completed:  false,
		},
	}

	if err := journal.save(); err != nil {
		return nil, fmt.Errorf("initialize scaffold journal: %w", err)
	}

	return journal, nil
}

func recoverPendingJournals(targetRoot string) error {
	txDir := filepath.Join(targetRoot, ".gokart", "tx")
	entries, err := os.ReadDir(txDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("list scaffold journals in %s: %w", filepath.ToSlash(txDir), err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		journalPath := filepath.Join(txDir, entry.Name())
		journal, err := loadApplyJournal(journalPath)
		if err != nil {
			return fmt.Errorf("load scaffold journal %s: %w", entry.Name(), err)
		}
		journal.State.TargetRoot = targetRoot

		if journal.State.Completed {
			if err := journal.cleanup(); err != nil {
				return fmt.Errorf("cleanup completed scaffold journal %s: %w", entry.Name(), err)
			}
			continue
		}

		actions, err := journal.State.rollbackActions()
		if err != nil {
			var mismatchErr *journalRecoveryMismatchError
			if errors.As(err, &mismatchErr) {
				return fmt.Errorf("recovery blocked for unfinished scaffold journal %s: %s (target files changed after failure; resolve manually or remove %s if safe)", entry.Name(), mismatchErr.Error(), filepath.ToSlash(journal.Path))
			}
			return fmt.Errorf("decode rollback actions for scaffold journal %s: %w", entry.Name(), err)
		}

		if err := rollbackWrites(actions); err != nil {
			return fmt.Errorf("recover unfinished scaffold journal %s: %w", entry.Name(), err)
		}

		journal.State.Completed = true
		if err := journal.save(); err != nil {
			return fmt.Errorf("mark recovered scaffold journal %s complete: %w", entry.Name(), err)
		}

		if err := journal.cleanup(); err != nil {
			return fmt.Errorf("cleanup recovered scaffold journal %s: %w", entry.Name(), err)
		}
	}

	return nil
}

func loadApplyJournal(path string) (*applyJournal, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read journal file: %w", err)
	}

	var state applyJournalState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("decode journal file: %w", err)
	}

	if state.Version == 1 {
		for i := range state.Actions {
			state.Actions[i].Applied = true
		}
		state.Version = applyJournalVersion
	}

	if state.Version != applyJournalVersion {
		return nil, fmt.Errorf("unsupported journal version %d", state.Version)
	}

	if strings.TrimSpace(state.TargetRoot) == "" {
		txDir := filepath.Dir(path)
		gokartDir := filepath.Dir(txDir)
		state.TargetRoot = filepath.Dir(gokartDir)
	}

	return &applyJournal{Path: path, State: state}, nil
}

func (j *applyJournal) appendAction(action rollbackAction) (int, error) {
	if j == nil {
		return -1, nil
	}

	entry, err := rollbackActionToJournalEntry(j.State.TargetRoot, action)
	if err != nil {
		return -1, err
	}

	j.State.Actions = append(j.State.Actions, entry)
	idx := len(j.State.Actions) - 1
	if err := j.save(); err != nil {
		return -1, err
	}

	return idx, nil
}

func (j *applyJournal) markActionApplied(index int) error {
	if j == nil {
		return nil
	}

	if index < 0 || index >= len(j.State.Actions) {
		return fmt.Errorf("journal action index %d out of range", index)
	}

	j.State.Actions[index].Applied = true
	return j.save()
}

func (j *applyJournal) markCompleted() error {
	if j == nil {
		return nil
	}

	j.State.Completed = true
	return j.save()
}

func (j *applyJournal) cleanup() error {
	if j == nil {
		return nil
	}

	if err := os.Remove(j.Path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove journal file %s: %w", filepath.ToSlash(j.Path), err)
	}

	cleanupJournalDirsBestEffort(j.State.TargetRoot)
	return nil
}

func (j *applyJournal) save() error {
	data, err := json.MarshalIndent(j.State, "", "  ")
	if err != nil {
		return fmt.Errorf("encode journal data: %w", err)
	}

	data = append(data, '\n')
	if err := writeFileAtomic(j.Path, data, 0600); err != nil {
		return fmt.Errorf("write journal file %s: %w", filepath.ToSlash(j.Path), err)
	}

	return nil
}

func rollbackActionToJournalEntry(targetRoot string, action rollbackAction) (applyJournalEntry, error) {
	root := action.Root
	if root == "" {
		root = targetRoot
	}

	if root == "" {
		return applyJournalEntry{}, fmt.Errorf("rollback action root is empty")
	}

	if action.Path == "" {
		return applyJournalEntry{}, fmt.Errorf("rollback action path is empty")
	}

	relPath, err := filepath.Rel(root, action.Path)
	if err != nil {
		return applyJournalEntry{}, fmt.Errorf("compute rollback path for journal: %w", err)
	}

	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return applyJournalEntry{}, fmt.Errorf("rollback path %s escapes root %s", action.Path, root)
	}

	entry := applyJournalEntry{RelPath: filepath.ToSlash(relPath)}

	switch action.Kind {
	case rollbackRemove:
		entry.Kind = "remove"
	case rollbackRestore:
		entry.Kind = "restore"
		entry.Content = action.Content
		entry.Mode = uint32(action.Mode)
	default:
		return applyJournalEntry{}, fmt.Errorf("unsupported rollback action kind %d", action.Kind)
	}

	entry.ExpectedExists = action.ExpectedExists
	entry.ExpectedSHA256 = action.ExpectedHash
	entry.ExpectedSize = action.ExpectedSize
	entry.ExpectedMode = uint32(action.ExpectedMode)

	return entry, nil
}

func (state applyJournalState) rollbackActions() ([]rollbackAction, error) {
	if strings.TrimSpace(state.TargetRoot) == "" {
		return nil, fmt.Errorf("journal target root is empty")
	}

	actions := make([]rollbackAction, 0, len(state.Actions))
	for i, entry := range state.Actions {
		if !entry.Applied {
			continue
		}

		relPath := filepath.FromSlash(entry.RelPath)
		destPath, err := safeDestinationPath(state.TargetRoot, relPath)
		if err != nil {
			return nil, fmt.Errorf("entry %d path %q: %w", i, entry.RelPath, err)
		}

		if err := verifyJournalActionExpectedState(destPath, entry); err != nil {
			return nil, err
		}

		action := rollbackAction{Root: state.TargetRoot, Path: destPath}
		switch entry.Kind {
		case "remove":
			action.Kind = rollbackRemove
		case "restore":
			action.Kind = rollbackRestore
			action.Content = entry.Content
			action.Mode = fs.FileMode(entry.Mode)
			if action.Mode == 0 {
				action.Mode = 0644
			}
		default:
			return nil, fmt.Errorf("entry %d has unknown action kind %q", i, entry.Kind)
		}

		actions = append(actions, action)
	}

	return actions, nil
}

func verifyJournalActionExpectedState(destPath string, entry applyJournalEntry) error {
	if !entry.ExpectedExists {
		return nil
	}

	info, err := os.Lstat(destPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &journalRecoveryMismatchError{RelPath: entry.RelPath, Reason: "expected generated file is missing"}
		}
		return fmt.Errorf("inspect expected state for %s: %w", entry.RelPath, err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return &journalRecoveryMismatchError{RelPath: entry.RelPath, Reason: "expected generated file became a symlink"}
	}

	if info.IsDir() {
		return &journalRecoveryMismatchError{RelPath: entry.RelPath, Reason: "expected generated file became a directory"}
	}

	if !info.Mode().IsRegular() {
		return &journalRecoveryMismatchError{RelPath: entry.RelPath, Reason: "expected generated file is not regular"}
	}

	content, err := os.ReadFile(destPath)
	if err != nil {
		return fmt.Errorf("read expected state for %s: %w", entry.RelPath, err)
	}

	if entry.ExpectedSize >= 0 && int64(len(content)) != entry.ExpectedSize {
		return &journalRecoveryMismatchError{RelPath: entry.RelPath, Reason: "generated file size changed after failure"}
	}

	if entry.ExpectedSHA256 != "" && sha256Hex(content) != entry.ExpectedSHA256 {
		return &journalRecoveryMismatchError{RelPath: entry.RelPath, Reason: "generated file content changed after failure"}
	}

	if entry.ExpectedMode != 0 && info.Mode().Perm() != fs.FileMode(entry.ExpectedMode) {
		return &journalRecoveryMismatchError{RelPath: entry.RelPath, Reason: "generated file mode changed after failure"}
	}

	return nil
}

func cleanupJournalDirsBestEffort(targetRoot string) {
	if strings.TrimSpace(targetRoot) == "" {
		return
	}

	txDir := filepath.Join(targetRoot, ".gokart", "tx")
	_ = os.Remove(txDir)
	_ = os.Remove(filepath.Dir(txDir))
}

func acquireApplyLock(targetDir string) (func() error, error) {
	targetRoot, err := filepath.Abs(targetDir)
	if err != nil {
		return nil, fmt.Errorf("resolve target directory %q for lock: %w", targetDir, err)
	}

	createdTargetDir := false
	info, err := os.Stat(targetRoot)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("check target directory %q for lock: %w", targetDir, err)
		}

		if err := os.MkdirAll(targetRoot, 0755); err != nil {
			return nil, fmt.Errorf("create target directory %q for lock: %w", targetDir, err)
		}
		createdTargetDir = true
	} else if !info.IsDir() {
		return nil, fmt.Errorf("target path %q exists and is not a directory", targetDir)
	}

	lockPath := filepath.Join(targetRoot, ".gokart.lock")
	if err := ensureNoSymlinkFromRoot(targetRoot, lockPath); err != nil {
		if createdTargetDir {
			_ = os.Remove(targetRoot)
		}
		return nil, fmt.Errorf("cannot lock target directory %q: %w", targetDir, err)
	}

	if err := createApplyLockFile(lockPath); err != nil {
		if !os.IsExist(err) {
			if createdTargetDir {
				_ = os.Remove(targetRoot)
			}
			return nil, fmt.Errorf("create lock file %s: %w", filepath.ToSlash(lockPath), err)
		}

		isStale, reason, staleErr := shouldReclaimStaleLock(lockPath)
		if staleErr != nil {
			if createdTargetDir {
				_ = os.Remove(targetRoot)
			}
			return nil, &ApplyLockError{TargetDir: targetDir, Reason: fmt.Sprintf("existing lock unreadable: %v", staleErr)}
		}

		if !isStale {
			if createdTargetDir {
				_ = os.Remove(targetRoot)
			}
			return nil, &ApplyLockError{TargetDir: targetDir}
		}

		if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
			if createdTargetDir {
				_ = os.Remove(targetRoot)
			}
			return nil, fmt.Errorf("remove stale lock file %s: %w", filepath.ToSlash(lockPath), err)
		}

		if err := createApplyLockFile(lockPath); err != nil {
			if createdTargetDir {
				_ = os.Remove(targetRoot)
			}
			if os.IsExist(err) {
				return nil, &ApplyLockError{TargetDir: targetDir}
			}
			return nil, fmt.Errorf("recreate lock file %s after stale cleanup (%s): %w", filepath.ToSlash(lockPath), reason, err)
		}
	}

	return func() error {
		var errs []error

		if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("remove lock file %s: %w", filepath.ToSlash(lockPath), err))
		}

		if createdTargetDir {
			_ = os.Remove(targetRoot)
		}

		if len(errs) == 0 {
			return nil
		}

		return errors.Join(errs...)
	}, nil
}

func createApplyLockFile(lockPath string) error {
	lockFile, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}

	metadata := applyLockMetadata{
		PID:        os.Getpid(),
		CreatedAt:  time.Now().UTC(),
		StaleAfter: int64(applyLockStaleAfter / time.Second),
	}

	if _, err := lockFile.Write(metadata.encode()); err != nil {
		_ = lockFile.Close()
		_ = os.Remove(lockPath)
		return fmt.Errorf("write lock metadata: %w", err)
	}

	if err := lockFile.Sync(); err != nil {
		_ = lockFile.Close()
		_ = os.Remove(lockPath)
		return fmt.Errorf("sync lock metadata: %w", err)
	}

	if err := lockFile.Close(); err != nil {
		_ = os.Remove(lockPath)
		return fmt.Errorf("close lock file: %w", err)
	}

	return nil
}

type applyLockMetadata struct {
	PID        int
	CreatedAt  time.Time
	StaleAfter int64
}

func (m applyLockMetadata) encode() []byte {
	createdAt := m.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	staleAfter := m.StaleAfter
	if staleAfter <= 0 {
		staleAfter = int64(applyLockStaleAfter / time.Second)
	}

	return []byte(fmt.Sprintf("pid=%d\ncreated_unix=%d\nstale_after_seconds=%d\n", m.PID, createdAt.Unix(), staleAfter))
}

func shouldReclaimStaleLock(lockPath string) (bool, string, error) {
	info, err := os.Lstat(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return true, "lock vanished", nil
		}
		return false, "", err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return false, "", fmt.Errorf("lock path is a symlink")
	}

	if info.IsDir() {
		return false, "", fmt.Errorf("lock path is a directory")
	}

	if !info.Mode().IsRegular() {
		return false, "", fmt.Errorf("lock path is not a regular file")
	}

	content, err := os.ReadFile(lockPath)
	if err != nil {
		return false, "", fmt.Errorf("read lock file: %w", err)
	}

	metadata, parseErr := parseApplyLockMetadata(content)
	if parseErr != nil {
		if lockAge := time.Since(info.ModTime()); lockAge > applyLockStaleAfter {
			return true, fmt.Sprintf("unparseable lock older than %s", applyLockStaleAfter), nil
		}
		return false, "", parseErr
	}

	lockAge := time.Since(info.ModTime())
	maxAge := applyLockStaleAfter
	if metadata.StaleAfter > 0 {
		maxAge = time.Duration(metadata.StaleAfter) * time.Second
	}

	if metadata.PID > 0 {
		running, err := processIsRunning(metadata.PID)
		if err != nil {
			return false, "", fmt.Errorf("check lock pid %d: %w", metadata.PID, err)
		}
		if !running {
			return true, fmt.Sprintf("pid %d is not running", metadata.PID), nil
		}

		if lockAge > maxAge {
			return true, fmt.Sprintf("pid %d is running but lock is older than %s", metadata.PID, maxAge), nil
		}

		return false, "", nil
	}

	if lockAge > maxAge {
		return true, fmt.Sprintf("lock older than %s", maxAge), nil
	}

	return false, "", nil
}

func parseApplyLockMetadata(content []byte) (applyLockMetadata, error) {
	metadata := applyLockMetadata{}
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return applyLockMetadata{}, fmt.Errorf("invalid lock metadata line %q", line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "pid":
			pid, err := parsePositiveInt(value)
			if err != nil {
				return applyLockMetadata{}, fmt.Errorf("invalid pid %q", value)
			}
			metadata.PID = pid
		case "created_unix":
			createdUnix, err := parsePositiveInt64(value)
			if err != nil {
				return applyLockMetadata{}, fmt.Errorf("invalid created_unix %q", value)
			}
			metadata.CreatedAt = time.Unix(createdUnix, 0).UTC()
		case "stale_after_seconds":
			seconds, err := parsePositiveInt64(value)
			if err != nil {
				return applyLockMetadata{}, fmt.Errorf("invalid stale_after_seconds %q", value)
			}
			metadata.StaleAfter = seconds
		}
	}

	if metadata.PID <= 0 {
		return applyLockMetadata{}, fmt.Errorf("missing pid metadata")
	}

	return metadata, nil
}

func processIsRunning(pid int) (bool, error) {
	if pid <= 0 {
		return false, nil
	}

	err := syscall.Kill(pid, 0)
	if err == nil {
		return true, nil
	}

	if errors.Is(err, syscall.ESRCH) {
		return false, nil
	}

	if errors.Is(err, syscall.EPERM) {
		return true, nil
	}

	return false, err
}

func parsePositiveInt(value string) (int, error) {
	parsed, err := parsePositiveInt64(value)
	if err != nil {
		return 0, err
	}
	return int(parsed), nil
}

func parsePositiveInt64(value string) (int64, error) {
	if value == "" {
		return 0, fmt.Errorf("empty numeric value")
	}

	var n int64
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid numeric value %q", value)
		}
		n = n*10 + int64(ch-'0')
	}

	return n, nil
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

	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("sync temp file for %s: %w", path, err)
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

	if err := syncDirBestEffort(dir); err != nil {
		return fmt.Errorf("sync directory for %s: %w", path, err)
	}

	cleanup = false
	return nil
}

func syncDirBestEffort(dir string) error {
	fd, err := os.Open(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer fd.Close()

	if err := fd.Sync(); err != nil {
		if errors.Is(err, syscall.EINVAL) || errors.Is(err, syscall.ENOTSUP) || errors.Is(err, syscall.EPERM) {
			return nil
		}
		return err
	}

	return nil
}
