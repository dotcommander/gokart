package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

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

		rendered, err := renderTemplate(fsys, path, data)
		if err != nil {
			return err
		}

		if len(bytes.TrimSpace(rendered)) == 0 {
			return nil
		}

		entry, conflict, err := planFileAction(root, targetRoot, path, rendered, policy)
		if err != nil {
			return err
		}

		if conflict != "" {
			conflicts = append(conflicts, conflict)
			return nil
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

// planFileAction determines the planned action for a single template file.
// It returns the populated plannedFile and an empty conflict string on success.
// When the file conflicts with an existing file under ExistingFilePolicyFail,
// it returns a zero plannedFile and the conflict path string instead.
func planFileAction(root, targetRoot, path string, rendered []byte, policy ExistingFilePolicy) (plannedFile, string, error) {
	outPath, err := templateOutputPath(root, path)
	if err != nil {
		return plannedFile{}, "", err
	}

	destPath, err := safeDestinationPath(targetRoot, outPath)
	if err != nil {
		return plannedFile{}, "", err
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
			return entry, "", nil
		}
		return plannedFile{}, "", fmt.Errorf("check destination %s: %w", entry.RelPath, err)
	}

	if info.IsDir() {
		return plannedFile{}, "", fmt.Errorf("destination path %s exists and is a directory", entry.RelPath)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return plannedFile{}, "", fmt.Errorf("destination path %s exists and is a symlink", entry.RelPath)
	}

	if !info.Mode().IsRegular() {
		return plannedFile{}, "", fmt.Errorf("destination path %s is not a regular file", entry.RelPath)
	}

	existing, err := os.ReadFile(entry.DestPath)
	if err != nil {
		return plannedFile{}, "", fmt.Errorf("read destination %s: %w", entry.RelPath, err)
	}

	entry.Existing = existing
	entry.ExistingMode = info.Mode().Perm()
	entry.ExistingInfo = info
	entry.Fingerprint = fingerprintForFile(info, existing)

	if bytes.Equal(existing, rendered) {
		entry.Action = planActionUnchanged
		return entry, "", nil
	}

	switch policy {
	case ExistingFilePolicyFail:
		return plannedFile{}, entry.RelPath, nil
	case ExistingFilePolicySkip:
		entry.Action = planActionSkip
	case ExistingFilePolicyOverwrite:
		entry.Action = planActionOverwrite
	default:
		return plannedFile{}, "", fmt.Errorf("invalid existing file policy %q", policy)
	}

	return entry, "", nil
}

func safeDestinationPath(targetRoot, relPath string) (string, error) {
	cleanRel := filepath.Clean(relPath)
	if cleanRel == "." || cleanRel == ".." || filepath.IsAbs(cleanRel) || strings.HasPrefix(cleanRel, ".."+string(filepath.Separator)) { //nolint:goconst // ".." is a well-known path literal used in path-safety checks across multiple files
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

//nolint:gocognit,gocyclo // security check, iterates path components
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
		if part == "" || part == "." {
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

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func checkRegularFileAfterPlan(err error, relPath string) error {
	var nrf *notRegularFileError
	if errors.As(err, &nrf) {
		return fmt.Errorf("destination path %s %s after planning", relPath, nrf.Reason)
	}
	return nil
}

func ensureCreateStateUnchanged(file plannedFile) error {
	if err := ensureNoSymlinkFromRoot(file.TargetRoot, file.DestPath); err != nil {
		return fmt.Errorf("destination path %s became unsafe after planning: %w", file.RelPath, err)
	}

	_, err := assertRegularFile(file.DestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		if msg := checkRegularFileAfterPlan(err, file.RelPath); msg != nil {
			return msg
		}
		return fmt.Errorf("check destination %s before write: %w", file.RelPath, err)
	}

	return fmt.Errorf("destination file %s appeared after planning; rerun command (or use --force/--skip-existing)", file.RelPath)
}

func ensureOverwriteStateUnchanged(file plannedFile) error {
	if err := ensureNoSymlinkFromRoot(file.TargetRoot, file.DestPath); err != nil {
		return fmt.Errorf("destination path %s became unsafe after planning: %w", file.RelPath, err)
	}

	info, err := assertRegularFile(file.DestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("destination file %s was removed after planning; rerun command", file.RelPath)
		}
		if msg := checkRegularFileAfterPlan(err, file.RelPath); msg != nil {
			return msg
		}
		return fmt.Errorf("check destination %s before write: %w", file.RelPath, err)
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
