package generator

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

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
		Mode:        info.Mode(), Size: info.Size(), ModTimeNano: info.ModTime().UnixNano(),
	}
}

func (f fileFingerprint) equal(other fileFingerprint) bool {
	return f.ContentHash == other.ContentHash && f.Mode == other.Mode &&
		f.Size == other.Size && f.ModTimeNano == other.ModTimeNano
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
