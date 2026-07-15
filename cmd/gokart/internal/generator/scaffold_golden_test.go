package generator

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Golden-tree tests assert that the REAL embedded templates (scaffold.go's
// `templates embed.FS`) render to a stable, committed snapshot. A bad template
// edit that still satisfies the Apply()-mechanics tests will break these.
//
// Two values are intentionally normalized/excluded so goldens stay stable
// across Go toolchains:
//   - the `go X.Y.Z` directive in go.mod (sourced from runtime.Version()).
//   - .gokart-manifest.json, whose SHA256 entries hash files containing the
//     Go version. Manifest content is covered by scaffolder_manifest_test.go.
//
// Regenerate goldens: GOKART_UPDATE_GOLDEN=1 go test ./cmd/gokart/internal/generator -run Golden

const goldenModule = "github.com/example/demo"
const goldenName = "demo"

// goDirectiveLine matches the `go X[.Y[.Z]]` directive in a rendered go.mod.
var goDirectiveLine = regexp.MustCompile(`(?m)^go [0-9]+(\.[0-9]+)*$`)

// normalizeGolden replaces the volatile go-version directive so snapshots are
// toolchain-independent. Applied to both produced and golden bytes.
func normalizeGolden(b []byte) []byte {
	return goDirectiveLine.ReplaceAll(b, []byte("go GOVERSION"))
}

func goldenUpdateEnabled() bool {
	return os.Getenv("GOKART_UPDATE_GOLDEN") == "1"
}

// collectTree walks dir and returns a map of slash-relative path -> file bytes,
// excluding the scaffold manifest (non-deterministic; tested elsewhere).
func collectTree(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if rel == scaffoldManifestPath {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		out[rel] = data
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	return out
}

func sortedKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// assertGolden compares the produced tree against testdata/golden/<variant>,
// normalizing volatile lines. With GOKART_UPDATE_GOLDEN=1 it (re)writes the
// golden tree instead of asserting.
func assertGolden(t *testing.T, variant, producedDir string) {
	t.Helper()
	goldenDir := filepath.Join("testdata", "golden", variant)
	produced := collectTree(t, producedDir)

	if goldenUpdateEnabled() {
		if err := os.RemoveAll(goldenDir); err != nil {
			t.Fatalf("clear golden dir %s: %v", goldenDir, err)
		}
		for rel, data := range produced {
			dest := filepath.Join(goldenDir, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				t.Fatalf("mkdir for golden %s: %v", dest, err)
			}
			if err := os.WriteFile(dest, normalizeGolden(data), 0o644); err != nil {
				t.Fatalf("write golden %s: %v", dest, err)
			}
		}
		t.Logf("updated golden tree: %s (%d files)", goldenDir, len(produced))
		return
	}

	golden := collectTree(t, goldenDir)

	producedKeys := sortedKeys(produced)
	goldenKeys := sortedKeys(golden)
	if strings.Join(producedKeys, "\n") != strings.Join(goldenKeys, "\n") {
		t.Fatalf("file set mismatch for %s\nproduced:\n  %s\ngolden:\n  %s\n(run GOKART_UPDATE_GOLDEN=1 go test ./cmd/gokart/internal/generator -run Golden to refresh)",
			variant, strings.Join(producedKeys, "\n  "), strings.Join(goldenKeys, "\n  "))
	}

	for _, rel := range producedKeys {
		got := normalizeGolden(produced[rel])
		want := golden[rel]
		if string(got) != string(want) {
			t.Errorf("content mismatch for %s/%s\n--- got ---\n%s\n--- want ---\n%s\n(run GOKART_UPDATE_GOLDEN=1 go test ./cmd/gokart/internal/generator -run Golden to refresh)",
				variant, rel, string(got), string(want))
		}
	}
}

func TestScaffoldStructuredGolden(t *testing.T) {
	t.Parallel()
	// Variant matrix: each integration flag exercised independently so a drift
	// in any conditional template block is isolated to one golden subtree.
	// Combination variants are intentionally omitted: every conditional in the
	// templates keys off a single flag (verified: go.mod.tmpl uses per-flag
	// `if`/`if or`), so single-flag coverage plus the all-off baseline exercises
	// every branch without combinatorial golden bloat.
	cases := []struct {
		variant     string
		useSQLite   bool
		usePostgres bool
		useAI       bool
		useRedis    bool
		useGlobal   bool
	}{
		{variant: "structured"},
		{variant: "structured-sqlite", useSQLite: true},
		{variant: "structured-postgres", usePostgres: true},
		{variant: "structured-ai", useAI: true},
		{variant: "structured-redis", useRedis: true},
		{variant: "structured-global", useGlobal: true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.variant, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if _, err := ScaffoldStructured(dir, goldenName, goldenModule,
				tc.useSQLite, tc.usePostgres, tc.useAI, tc.useRedis,
				tc.useGlobal, true,
				ApplyOptions{ExistingFilePolicy: ExistingFilePolicyOverwrite}); err != nil {
				t.Fatalf("ScaffoldStructured(%s) error = %v", tc.variant, err)
			}
			assertGolden(t, tc.variant, dir)
		})
	}
}

func TestScaffoldFlatGolden(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if _, err := ScaffoldFlat(dir, goldenName, goldenModule,
		false, true,
		ApplyOptions{ExistingFilePolicy: ExistingFilePolicyOverwrite}); err != nil {
		t.Fatalf("ScaffoldFlat error = %v", err)
	}
	assertGolden(t, "flat", dir)
}

func TestGlobalScaffoldsOnlyEmitProductDocumentation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		scaffold func(string) error
	}{
		{
			name: "flat",
			scaffold: func(dir string) error {
				_, err := ScaffoldFlat(dir, goldenName, goldenModule, true, true,
					ApplyOptions{ExistingFilePolicy: ExistingFilePolicyOverwrite})
				return err
			},
		},
		{
			name: "structured",
			scaffold: func(dir string) error {
				_, err := ScaffoldStructured(dir, goldenName, goldenModule,
					false, false, false, false, true, true,
					ApplyOptions{ExistingFilePolicy: ExistingFilePolicyOverwrite})
				return err
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if err := tc.scaffold(dir); err != nil {
				t.Fatalf("scaffold: %v", err)
			}
			for rel := range collectTree(t, dir) {
				if strings.HasSuffix(strings.ToLower(rel), ".md") && rel != "README.md" {
					t.Errorf("unexpected generated documentation: %s", rel)
				}
			}
		})
	}
}
