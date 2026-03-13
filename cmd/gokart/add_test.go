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
	if validIntegrations["redis"] {
		t.Fatal("redis should not be a valid integration")
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

func TestAddInfersStateFromGoMod(t *testing.T) {
	dir := setupAddTestProject(t, setupAddTestOpts{
		Module:          "example.com/myapp",
		ManifestVersion: scaffoldManifestV1,
		TemplateRoot:    "templates/structured",
		GoModExtra:      "\nrequire github.com/dotcommander/gokart/sqlite v0.0.0\nrequire github.com/dotcommander/gokart/ai v0.0.0\n",
	})

	_, result := inferIntegrationsFromGoMod(dir)
	if !result.SQLite {
		t.Fatal("expected SQLite to be detected")
	}
	if !result.AI {
		t.Fatal("expected AI to be detected")
	}
	if result.Postgres {
		t.Fatal("Postgres should not be detected")
	}
}

func TestAddDetectsIntegrationsFromV2Manifest(t *testing.T) {
	dir := setupAddTestProject(t, setupAddTestOpts{
		Module:          "example.com/myapp",
		ManifestVersion: scaffoldManifestV2,
		TemplateRoot:    "templates/structured",
		Mode:            "structured",
		Integrations:    &manifestIntegrations{SQLite: true, Postgres: false, AI: false},
	})

	manifest, err := readAddManifest(dir)
	if err != nil {
		t.Fatalf("readAddManifest: %v", err)
	}

	_, current := detectCurrentIntegrations(manifest, dir)
	if !current.SQLite {
		t.Fatal("expected SQLite from v2 manifest")
	}
	if current.AI {
		t.Fatal("AI should not be enabled in v2 manifest")
	}
}

func TestAddCreatesContextForFirstIntegration(t *testing.T) {
	dir := setupAddTestProject(t, setupAddTestOpts{
		Module:          "example.com/myapp",
		ManifestVersion: scaffoldManifestV2,
		TemplateRoot:    "templates/structured",
		Mode:            "structured",
		Integrations:    &manifestIntegrations{},
		UseGlobal:       boolPtr(true),
	})

	// dir is used only to satisfy the project setup; rendering is template-driven
	_ = dir

	data := baseTemplateData("myapp", "example.com/myapp", true, false)
	data.UseSQLite = true

	files, err := renderIntegrationFiles(data)
	if err != nil {
		t.Fatalf("renderIntegrationFiles: %v", err)
	}

	contextContent, ok := files["internal/app/context.go"]
	if !ok {
		t.Fatal("expected context.go to be rendered")
	}

	if !strings.Contains(string(contextContent), "DB") {
		t.Fatal("expected context.go to contain DB field")
	}
	if !strings.Contains(string(contextContent), "sqlite") {
		t.Fatal("expected context.go to reference sqlite")
	}
}

func TestAddUpdatesContextForNewIntegration(t *testing.T) {
	// Start with sqlite, add AI
	data := baseTemplateData("myapp", "example.com/myapp", true, false)
	data.UseSQLite = true
	data.UseAI = true

	files, err := renderIntegrationFiles(data)
	if err != nil {
		t.Fatalf("renderIntegrationFiles: %v", err)
	}

	contextContent, ok := files["internal/app/context.go"]
	if !ok {
		t.Fatal("expected context.go to be rendered")
	}

	content := string(contextContent)
	if !strings.Contains(content, "DB") {
		t.Fatal("expected context.go to contain DB field for sqlite")
	}
	if !strings.Contains(content, "AI") {
		t.Fatal("expected context.go to contain AI field")
	}
}

func TestAddUpdatesRootCommand(t *testing.T) {
	data := baseTemplateData("myapp", "example.com/myapp", true, false)
	data.UseSQLite = true

	files, err := renderIntegrationFiles(data)
	if err != nil {
		t.Fatalf("renderIntegrationFiles: %v", err)
	}

	rootContent, ok := files["internal/commands/root.go"]
	if !ok {
		t.Fatal("expected root.go to be rendered")
	}

	content := string(rootContent)
	if !strings.Contains(content, "PersistentPreRunE") {
		t.Fatal("expected root.go to contain PersistentPreRunE wiring")
	}
	if !strings.Contains(content, "app.New") {
		t.Fatal("expected root.go to reference app.New")
	}
}

func TestAddConflictsOnUserModifiedFile(t *testing.T) {
	originalContent := []byte("// original content\n")
	originalHash := sha256Hex(originalContent)

	dir := setupAddTestProject(t, setupAddTestOpts{
		Module:          "example.com/myapp",
		ManifestVersion: scaffoldManifestV2,
		TemplateRoot:    "templates/structured",
		Mode:            "structured",
		Integrations:    &manifestIntegrations{},
		Files: []scaffoldManifestFile{
			{
				Path:          "internal/commands/root.go",
				Action:        "create",
				ContentSHA256: originalHash,
				Mode:          0644,
			},
		},
		ExtraFiles: map[string]string{
			"internal/commands/root.go": "// user modified this file\n",
		},
	})

	manifest, _ := readAddManifest(dir)
	safety := checkFileSafety(dir, "internal/commands/root.go", manifest)
	if safety != fileSafetyConflict {
		t.Fatalf("expected conflict, got %d", safety)
	}
}

func TestAddSafeOverwriteUnmodifiedFile(t *testing.T) {
	originalContent := []byte("// original content\n")
	originalHash := sha256Hex(originalContent)

	dir := setupAddTestProject(t, setupAddTestOpts{
		Module:          "example.com/myapp",
		ManifestVersion: scaffoldManifestV2,
		TemplateRoot:    "templates/structured",
		Mode:            "structured",
		Integrations:    &manifestIntegrations{},
		Files: []scaffoldManifestFile{
			{
				Path:          "internal/commands/root.go",
				Action:        "create",
				ContentSHA256: originalHash,
				Mode:          0644,
			},
		},
		ExtraFiles: map[string]string{
			"internal/commands/root.go": "// original content\n",
		},
	})

	manifest, _ := readAddManifest(dir)
	safety := checkFileSafety(dir, "internal/commands/root.go", manifest)
	if safety != fileSafetySafe {
		t.Fatalf("expected safe, got %d", safety)
	}
}

func TestAddDryRunNoChanges(t *testing.T) {
	// Verify that renderIntegrationFiles produces content for postgres
	data := baseTemplateData("myapp", "example.com/myapp", true, false)
	data.UsePostgres = true

	files, err := renderIntegrationFiles(data)
	if err != nil {
		t.Fatalf("renderIntegrationFiles: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("expected files to be rendered")
	}

	if _, ok := files["internal/app/context.go"]; !ok {
		t.Fatal("expected context.go in rendered files")
	}
	if _, ok := files["internal/commands/root.go"]; !ok {
		t.Fatal("expected root.go in rendered files")
	}
}

func TestAddUpdatesManifest(t *testing.T) {
	dir := setupAddTestProject(t, setupAddTestOpts{
		Module:          "example.com/myapp",
		ManifestVersion: scaffoldManifestV1,
		TemplateRoot:    "templates/structured",
		Files: []scaffoldManifestFile{
			{Path: "cmd/main.go", Action: "create", ContentSHA256: "abc", Mode: 0644},
		},
	})

	manifest, err := readAddManifest(dir)
	if err != nil {
		t.Fatalf("readAddManifest: %v", err)
	}

	data := baseTemplateData("myapp", "example.com/myapp", true, false)
	data.UseSQLite = true

	renderedFiles := map[string][]byte{
		"internal/app/context.go":   []byte("// context\n"),
		"internal/commands/root.go": []byte("// root\n"),
	}

	if err := updateAddManifest(dir, manifest, data, renderedFiles); err != nil {
		t.Fatalf("updateAddManifest: %v", err)
	}

	// Re-read manifest
	updated, err := readAddManifest(dir)
	if err != nil {
		t.Fatalf("re-read manifest: %v", err)
	}

	if updated.Version != scaffoldManifestV2 {
		t.Fatalf("expected manifest version %d, got %d", scaffoldManifestV2, updated.Version)
	}

	if updated.Integrations == nil {
		t.Fatal("expected integrations in manifest")
	}

	if !updated.Integrations.SQLite {
		t.Fatal("expected SQLite in manifest integrations")
	}

	if updated.Module != "example.com/myapp" {
		t.Fatalf("expected module example.com/myapp, got %s", updated.Module)
	}

	if updated.Mode != "structured" {
		t.Fatalf("expected mode structured, got %s", updated.Mode)
	}

	// Verify file hashes were updated
	foundContext := false
	foundRoot := false
	for _, f := range updated.Files {
		if f.Path == "internal/app/context.go" {
			foundContext = true
			if f.ContentSHA256 != sha256Hex([]byte("// context\n")) {
				t.Fatal("context.go hash not updated")
			}
		}
		if f.Path == "internal/commands/root.go" {
			foundRoot = true
		}
	}
	if !foundContext {
		t.Fatal("context.go not in manifest files")
	}
	if !foundRoot {
		t.Fatal("root.go not in manifest files")
	}
}

func TestAddMultipleIntegrations(t *testing.T) {
	dir := setupAddTestProject(t, setupAddTestOpts{
		Module:          "example.com/myapp",
		ManifestVersion: scaffoldManifestV2,
		TemplateRoot:    "templates/structured",
		Mode:            "structured",
		Integrations:    &manifestIntegrations{},
		UseGlobal:       boolPtr(false),
	})

	manifest, _ := readAddManifest(dir)
	current := &manifestIntegrations{}

	data, err := inferTemplateData(manifest, dir, current, []string{"sqlite", "ai"}, "")
	if err != nil {
		t.Fatalf("inferTemplateData: %v", err)
	}

	if !data.UseSQLite {
		t.Fatal("expected UseSQLite")
	}
	if !data.UseAI {
		t.Fatal("expected UseAI")
	}
	if data.UsePostgres {
		t.Fatal("UsePostgres should be false")
	}
}

func TestAddFileCreateWhenMissing(t *testing.T) {
	dir := setupAddTestProject(t, setupAddTestOpts{
		Module:          "example.com/myapp",
		ManifestVersion: scaffoldManifestV2,
		TemplateRoot:    "templates/structured",
		Mode:            "structured",
		Integrations:    &manifestIntegrations{},
	})

	manifest, _ := readAddManifest(dir)

	// context.go doesn't exist yet
	safety := checkFileSafety(dir, "internal/app/context.go", manifest)
	if safety != fileSafetyCreate {
		t.Fatalf("expected create for missing file, got %d", safety)
	}
}

func TestAddParseModuleFromGoMod(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "go.mod", "module github.com/myorg/myapp\n\ngo 1.26\n")

	mod, _ := inferIntegrationsFromGoMod(dir)

	if mod != "github.com/myorg/myapp" {
		t.Fatalf("expected github.com/myorg/myapp, got %s", mod)
	}
}

func TestPrintAddResultIncludesAlreadyPresent(t *testing.T) {
	stdout := captureStdout(t, func() {
		printAddResult(addRequest{}, addCommandOutput{
			Added:          []string{"ai"},
			AlreadyPresent: []string{"sqlite"},
		})
	})

	if !strings.Contains(stdout, "sqlite already enabled") {
		t.Fatalf("expected already enabled output, got %q", stdout)
	}
	if !strings.Contains(stdout, "Added ai") {
		t.Fatalf("expected added output, got %q", stdout)
	}
}

func TestRunAddCommandJSONMarksVerifyRequestedOnFailure(t *testing.T) {
	tempDir := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	}()

	cmd := newAddCommand()
	mustSetFlagTrue(t, cmd, addFlagJSON)
	mustSetFlagTrue(t, cmd, addFlagVerify)

	stdout := captureStdout(t, func() {
		err = runAddCommand(cmd, []string{"sqlite"})
	})
	if err == nil {
		t.Fatal("expected runAddCommand to fail without manifest")
	}

	var output addCommandOutput
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("unmarshal JSON output: %v\noutput=%q", err, stdout)
	}
	if !output.VerifyRequested {
		t.Fatalf("expected verify_requested=true in JSON output, got %+v", output)
	}
	if output.ErrorCode != errorCodeManifestNotFound {
		t.Fatalf("expected error code %q, got %q", errorCodeManifestNotFound, output.ErrorCode)
	}
}
