package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestApplyWritesManifestWithFileProvenance(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"templates/basic/main.go.tmpl": {Data: []byte("package main\n")},
	}

	targetDir := t.TempDir()
	result, err := Apply(fsys, "templates/basic", targetDir, TemplateData{}, ApplyOptions{ExistingFilePolicy: ExistingFilePolicyOverwrite})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if len(result.Created) != 1 || result.Created[0] != "main.go" {
		t.Fatalf("unexpected created result: %#v", result.Created)
	}

	manifest := mustReadManifest(t, targetDir)
	if manifest.Version != scaffoldManifestV1 {
		t.Fatalf("unexpected manifest version: %d", manifest.Version)
	}
	if manifest.Generator != "gokart" {
		t.Fatalf("unexpected manifest generator: %q", manifest.Generator)
	}
	if manifest.GeneratorVersion != gokartVersion {
		t.Fatalf("unexpected manifest generator version: got %q, want %q", manifest.GeneratorVersion, gokartVersion)
	}
	if manifest.TemplateRoot != "templates/basic" {
		t.Fatalf("unexpected template root: %q", manifest.TemplateRoot)
	}
	if manifest.ExistingFilePolicy != ExistingFilePolicyOverwrite {
		t.Fatalf("unexpected manifest policy: %q", manifest.ExistingFilePolicy)
	}

	if len(manifest.Files) != 1 {
		t.Fatalf("expected 1 manifest file entry, got %d", len(manifest.Files))
	}

	entry := manifest.Files[0]
	if entry.Path != "main.go" {
		t.Fatalf("unexpected manifest path: %q", entry.Path)
	}
	if entry.Action != "create" {
		t.Fatalf("unexpected manifest action: %q", entry.Action)
	}

	expectedHash := sha256Hex([]byte("package main\n"))
	if entry.TemplateSHA256 != expectedHash {
		t.Fatalf("unexpected template hash: %q", entry.TemplateSHA256)
	}
	if entry.ContentSHA256 != expectedHash {
		t.Fatalf("unexpected content hash: %q", entry.ContentSHA256)
	}
	if entry.Mode != uint32(0644) {
		t.Fatalf("unexpected manifest mode: %o", entry.Mode)
	}
}

func TestApplyManifestTracksSkippedContent(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"templates/basic/main.go.tmpl": {Data: []byte("package main\n")},
	}

	targetDir := t.TempDir()
	dest := filepath.Join(targetDir, "main.go")
	seedContent := []byte("package stale\n")
	if err := os.WriteFile(dest, seedContent, 0644); err != nil {
		t.Fatalf("seed destination file: %v", err)
	}

	result, err := Apply(fsys, "templates/basic", targetDir, TemplateData{}, ApplyOptions{ExistingFilePolicy: ExistingFilePolicySkip})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if len(result.Skipped) != 1 || result.Skipped[0] != "main.go" {
		t.Fatalf("unexpected skipped result: %#v", result.Skipped)
	}

	manifest := mustReadManifest(t, targetDir)
	if len(manifest.Files) != 1 {
		t.Fatalf("expected 1 manifest file entry, got %d", len(manifest.Files))
	}

	entry := manifest.Files[0]
	if entry.Action != "skip" {
		t.Fatalf("unexpected manifest action: %q", entry.Action)
	}

	templateHash := sha256Hex([]byte("package main\n"))
	contentHash := sha256Hex(seedContent)
	if entry.TemplateSHA256 != templateHash {
		t.Fatalf("unexpected template hash: %q", entry.TemplateSHA256)
	}
	if entry.ContentSHA256 != contentHash {
		t.Fatalf("unexpected skipped content hash: %q", entry.ContentSHA256)
	}
	if entry.ContentSHA256 == entry.TemplateSHA256 {
		t.Fatal("expected skipped file content hash to differ from template hash")
	}
}

func TestApplyDryRunDoesNotWriteManifest(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"templates/basic/main.go.tmpl": {Data: []byte("package main\n")},
	}

	targetDir := t.TempDir()
	_, err := Apply(fsys, "templates/basic", targetDir, TemplateData{}, ApplyOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	manifestPath := filepath.Join(targetDir, scaffoldManifestPath)
	if _, err := os.Stat(manifestPath); !os.IsNotExist(err) {
		t.Fatalf("expected no manifest in dry-run mode, stat err = %v", err)
	}
}

func TestApplySkipManifestOptionDoesNotWriteManifest(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"templates/basic/main.go.tmpl": {Data: []byte("package main\n")},
	}

	targetDir := t.TempDir()
	result, err := Apply(fsys, "templates/basic", targetDir, TemplateData{}, ApplyOptions{ExistingFilePolicy: ExistingFilePolicyOverwrite, SkipManifest: true})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if len(result.Created) != 1 || result.Created[0] != "main.go" {
		t.Fatalf("unexpected created result: %#v", result.Created)
	}

	manifestPath := filepath.Join(targetDir, scaffoldManifestPath)
	if _, err := os.Stat(manifestPath); !os.IsNotExist(err) {
		t.Fatalf("expected no manifest when SkipManifest=true, stat err = %v", err)
	}
}

func TestManifestIsDeterministicWithoutGeneratedAtTimestamp(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"templates/basic/main.go.tmpl": {Data: []byte("package main\n")},
	}

	targetDir := t.TempDir()
	if _, err := Apply(fsys, "templates/basic", targetDir, TemplateData{}, ApplyOptions{ExistingFilePolicy: ExistingFilePolicyOverwrite}); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	manifestPath := filepath.Join(targetDir, scaffoldManifestPath)
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest file: %v", err)
	}

	if strings.Contains(string(manifestBytes), "generated_at") {
		t.Fatalf("expected deterministic manifest without generated_at, got: %s", string(manifestBytes))
	}
}

func mustReadManifest(t *testing.T, targetDir string) scaffoldManifest {
	t.Helper()

	manifestPath := filepath.Join(targetDir, scaffoldManifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest file: %v", err)
	}

	var manifest scaffoldManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("decode manifest file: %v", err)
	}

	return manifest
}
