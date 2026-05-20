package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupAddTestOpts controls what setupAddTestProject creates.
type setupAddTestOpts struct {
	Module          string
	ManifestVersion int
	TemplateRoot    string
	Mode            string
	Integrations    *manifestIntegrations
	UseGlobal       *bool
	Files           []scaffoldManifestFile
	GoModExtra      string
	ExtraFiles      map[string]string
}

// setupAddTestProject creates a minimal structured project dir for add tests.
// Returns the temp directory path.
func setupAddTestProject(t *testing.T, opts setupAddTestOpts) string {
	t.Helper()
	dir := t.TempDir()

	// Write go.mod
	goModContent := fmt.Sprintf("module %s\n\ngo 1.26\n", opts.Module)
	if opts.GoModExtra != "" {
		goModContent += opts.GoModExtra
	}
	writeTestFile(t, dir, "go.mod", goModContent)

	// Write manifest
	manifest := scaffoldManifest{
		Version:      opts.ManifestVersion,
		Generator:    "gokart",
		TemplateRoot: opts.TemplateRoot,
		Files:        opts.Files,
	}
	if opts.Integrations != nil {
		manifest.Integrations = opts.Integrations
	}
	if opts.Mode != "" {
		manifest.Mode = opts.Mode
	}
	if opts.Module != "" {
		manifest.Module = opts.Module
	}
	if opts.UseGlobal != nil {
		manifest.UseGlobal = opts.UseGlobal
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	writeTestFile(t, dir, scaffoldManifestPath, string(data)+"\n")

	// Write any extra files
	for path, content := range opts.ExtraFiles {
		writeTestFile(t, dir, path, content)
	}

	return dir
}

func writeTestFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatalf("mkdir for %s: %v", relPath, err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", relPath, err)
	}
}

func TestAddRejectsNoManifest(t *testing.T) {
	dir := t.TempDir()
	_, err := readAddManifest(dir)
	if err == nil {
		t.Fatal("expected error for missing manifest")
	}
	if !strings.Contains(err.Error(), "no manifest found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddRejectsFlat(t *testing.T) {
	dir := setupAddTestProject(t, setupAddTestOpts{
		Module:          "example.com/myapp",
		ManifestVersion: scaffoldManifestV1,
		TemplateRoot:    "templates/flat",
	})

	manifest, err := readAddManifest(dir)
	if err != nil {
		t.Fatalf("readAddManifest: %v", err)
	}

	if manifest.TemplateRoot != "templates/flat" {
		t.Fatal("expected flat template root")
	}
	// The check in runAddCommand would reject this
	if !isFlatProject(manifest) {
		t.Fatal("expected flat detection to be true")
	}
}

func TestAddRejectsUnknownIntegration(t *testing.T) {
	if validIntegrations["mysql"] {
		t.Fatal("mysql should not be a valid integration")
	}
	if !validIntegrations["sqlite"] {
		t.Fatal("sqlite should be valid")
	}
	if !validIntegrations["postgres"] {
		t.Fatal("postgres should be valid")
	}
	if !validIntegrations["ai"] {
		t.Fatal("ai should be valid")
	}
	if !validIntegrations["redis"] {
		t.Fatal("redis should be valid")
	}
}

func TestAddRejectsDuplicate(t *testing.T) {
	current := &manifestIntegrations{SQLite: true}
	if !integrationEnabled(current, "sqlite") {
		t.Fatal("sqlite should be detected as already enabled")
	}
	if integrationEnabled(current, "ai") {
		t.Fatal("ai should not be detected as already enabled")
	}
}
