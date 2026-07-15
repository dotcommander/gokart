package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// setupAddTestOpts controls what setupAddTestProject creates.
type setupAddTestOpts struct {
	Module           string
	ManifestVersion  int
	TemplateRoot     string
	Mode             string
	Integrations     *manifestIntegrations
	UseGlobal        *bool
	Files            []scaffoldManifestFile
	GoModExtra       string
	ExtraFiles       map[string]string
	GeneratorVersion string
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
		Version:          opts.ManifestVersion,
		Generator:        "gokart",
		GeneratorVersion: opts.GeneratorVersion,
		TemplateRoot:     opts.TemplateRoot,
		Files:            opts.Files,
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	if _, ok := integrationRegistry["mysql"]; ok {
		t.Fatal("mysql should not be a valid integration")
	}
	for _, name := range []string{integrationSQLite, integrationPostgres, integrationAI, integrationRedis} {
		if _, ok := integrationRegistry[name]; !ok {
			t.Fatalf("%s should be valid", name)
		}
	}
}

func TestAddRejectsDuplicate(t *testing.T) {
	t.Parallel()
	current := &manifestIntegrations{SQLite: true}
	if !integrationEnabled(current, "sqlite") {
		t.Fatal("sqlite should be detected as already enabled")
	}
	if integrationEnabled(current, "ai") {
		t.Fatal("ai should not be detected as already enabled")
	}
}

func TestIntegrationRegistryIsSingleSource(t *testing.T) { //nolint:gocyclo // one contract test validates every registry-owned projection
	t.Parallel()
	wantDeps := map[string][]string{
		integrationSQLite:   {"github.com/dotcommander/gokart/sqlite@" + defaultGokartSQLiteVersion},
		integrationPostgres: {"github.com/dotcommander/gokart/postgres@" + defaultGokartPostgresVersion, "github.com/jackc/pgx/v5@" + defaultPGXVersion},
		integrationAI:       {"github.com/openai/openai-go/v3@" + defaultOpenAIVersion},
		integrationRedis:    {"github.com/dotcommander/gokart/cache@" + defaultGokartCacheVersion, "github.com/redis/go-redis/v9@" + defaultRedisVersion},
	}
	if len(integrationRegistry) != len(wantDeps) {
		t.Fatalf("integrationRegistry has %d entries, want %d", len(integrationRegistry), len(wantDeps))
	}
	for name, entry := range integrationRegistry {
		if entry.template == nil || entry.setTemplate == nil || entry.get == nil || entry.set == nil || entry.description == "" || entry.flag == "" || entry.golden == "" || entry.environment == "" || entry.upstream == "" || entry.recipe == "" {
			t.Fatalf("%s has incomplete registry metadata", name)
		}
		if !slices.Equal(entry.deps, wantDeps[name]) {
			t.Fatalf("%s deps = %v, want %v", name, entry.deps, wantDeps[name])
		}
		if _, err := os.Stat(filepath.Join("testdata", "golden", entry.golden)); err != nil {
			t.Fatalf("%s golden coverage: %v", name, err)
		}
		readme, err := os.ReadFile(filepath.Join("testdata", "golden", entry.golden, "README.md"))
		if err != nil {
			t.Fatalf("%s README coverage: %v", name, err)
		}
		wantEnvironment := strings.ReplaceAll(entry.environment, "{{APP}}", strings.ToUpper(goldenName))
		if !strings.Contains(string(readme), entry.description) || !strings.Contains(string(readme), wantEnvironment) {
			t.Fatalf("%s generated README omits registry metadata", name)
		}
		docs, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "docs", "components", "generator.md"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(docs), entry.flag) || !strings.Contains(string(docs), name) {
			t.Fatalf("%s missing generator help/docs coverage", name)
		}
		ownership, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "docs", "generated-code.md"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(ownership), entry.upstream) {
			t.Fatalf("%s missing ownership/upstream coverage", name)
		}
		recipes, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "docs", "recipes.md"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(recipes), entry.recipe) {
			t.Fatalf("%s missing recipe coverage", name)
		}
		m := &manifestIntegrations{}
		entry.set(m, true)
		if !integrationEnabled(m, name) {
			t.Fatalf("set+integrationEnabled disagree for %s", name)
		}
		setIntegration(m, name, false)
		if integrationEnabled(m, name) {
			t.Fatalf("setIntegration(false) did not disable %s", name)
		}
	}
	if integrationEnabled(&manifestIntegrations{}, "mysql") {
		t.Fatal("integrationEnabled must reject unknown name")
	}
}
