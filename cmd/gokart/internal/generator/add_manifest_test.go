package generator

import (
	"testing"
)

func TestAddInfersStateFromGoMod(t *testing.T) {
	t.Parallel()
	dir := setupAddTestProject(t, setupAddTestOpts{
		Module:          "example.com/myapp",
		ManifestVersion: scaffoldManifestV1,
		TemplateRoot:    "templates/structured",
		GoModExtra:      "\nrequire github.com/dotcommander/gokart/sqlite v0.0.0\nrequire github.com/openai/openai-go/v3 v3.41.0\n",
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
	t.Parallel()
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

func TestAddUpdatesManifest(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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

func TestAddParseModuleFromGoMod(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeTestFile(t, dir, "go.mod", "module github.com/myorg/myapp\n\ngo 1.26\n")

	mod, _ := inferIntegrationsFromGoMod(dir)

	if mod != "github.com/myorg/myapp" {
		t.Fatalf("expected github.com/myorg/myapp, got %s", mod)
	}
}
