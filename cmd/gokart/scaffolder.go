package main

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// Apply walks the embedded filesystem and renders templates to targetDir.
// Files that render to empty content (whitespace only) are skipped.
func Apply(efs embed.FS, root string, targetDir string, data TemplateData) error {
	return fs.WalkDir(efs, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		// Calculate relative path and strip .tmpl extension
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		outPath := strings.TrimSuffix(relPath, ".tmpl")
		dest := filepath.Join(targetDir, outPath)

		// Read and parse template
		content, err := efs.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read template %s: %w", path, err)
		}

		tmpl, err := template.New(filepath.Base(path)).Parse(string(content))
		if err != nil {
			return fmt.Errorf("parse template %s: %w", path, err)
		}

		// Execute template
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return fmt.Errorf("execute template %s: %w", path, err)
		}

		// Skip-empty pattern: don't write files with empty content
		if strings.TrimSpace(buf.String()) == "" {
			return nil
		}

		// Create directory and write file
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return fmt.Errorf("create directory for %s: %w", outPath, err)
		}
		if err := os.WriteFile(dest, buf.Bytes(), 0644); err != nil {
			return fmt.Errorf("write file %s: %w", outPath, err)
		}

		return nil
	})
}
