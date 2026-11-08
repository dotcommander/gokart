package generator

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type noopRunner struct{}

func (noopRunner) Run(context.Context, Command) error { return nil }

// legacyProject scaffolds a managed structured sqlite project, then rewrites
// context.go to import the old gokart wrapper so it looks pre-detach.
func legacyProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	data := baseTemplateData("demo", "example.com/demo", false, false)
	data.UseSQLite = true
	data.derive(true)
	files, err := renderIntegrationFiles(data)
	if err != nil {
		t.Fatal(err)
	}
	manifest := &scaffoldManifest{Module: "example.com/demo", Mode: modeStructured, Integrations: &manifestIntegrations{SQLite: true}}
	legacy := []byte("package app\n\nimport _ \"github.com/dotcommander/gokart/sqlite\"\n")
	for rel, content := range files {
		if rel == appContextPath {
			content = legacy
		}
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatal(err)
		}
		manifest.Files = append(manifest.Files, scaffoldManifestFile{Path: rel, ContentSHA256: sha256Hex(content)})
	}
	mustWrite(t, filepath.Join(dir, "go.mod"), "module example.com/demo\n\ngo 1.26\n\nrequire github.com/dotcommander/gokart/sqlite v0.13.0\n")
	if err := updateAddManifest(dir, manifest, data, nil); err != nil {
		t.Fatal(err)
	}
	return dir
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDetachDryRunThenApply(t *testing.T) {
	t.Parallel()
	dir := legacyProject(t)
	svc := New(Dependencies{Runner: noopRunner{}})
	res, err := svc.Detach(context.Background(), DetachRequest{Dir: dir, DryRun: true}, Runtime{})
	if err != nil || len(res.FilesOverwritten) != 1 || res.FilesOverwritten[0] != appContextPath {
		t.Fatalf("dry run res=%+v err=%v", res, err)
	}
	if _, err := os.Stat(filepath.Join(dir, scaffoldManifestPath)); err != nil {
		t.Fatal("dry run removed manifest")
	}
	res, err = svc.Detach(context.Background(), DetachRequest{Dir: dir}, Runtime{})
	if err != nil || !res.ManifestRemoved {
		t.Fatalf("apply res=%+v err=%v", res, err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, appContextPath))
	if strings.Contains(string(got), "dotcommander/gokart") || !strings.Contains(string(got), "modernc.org/sqlite") {
		t.Fatalf("context.go not detached:\n%s", got)
	}
}

func TestDetachAlreadyDetached(t *testing.T) {
	t.Parallel()
	dir := legacyProject(t)
	data := baseTemplateData("demo", "example.com/demo", false, false)
	data.UseSQLite = true
	files, _ := renderIntegrationFiles(data)
	mustWrite(t, filepath.Join(dir, appContextPath), string(files[appContextPath]))
	mustWrite(t, filepath.Join(dir, "go.mod"), "module example.com/demo\n\ngo 1.26\n")
	res, err := New(Dependencies{Runner: noopRunner{}}).Detach(context.Background(), DetachRequest{Dir: dir}, Runtime{})
	if err != nil || !res.AlreadyDetached {
		t.Fatalf("res=%+v err=%v", res, err)
	}
}

func TestDetachConflictRequiresForce(t *testing.T) {
	t.Parallel()
	dir := legacyProject(t)
	mustWrite(t, filepath.Join(dir, appContextPath), "package app\n\nimport _ \"github.com/dotcommander/gokart/sqlite\"\n\n// edited\n")
	svc := New(Dependencies{Runner: noopRunner{}})
	_, err := svc.Detach(context.Background(), DetachRequest{Dir: dir}, Runtime{})
	var op *OperationError
	if !errors.As(err, &op) || op.Kind != ErrorExistingFileConflict {
		t.Fatalf("err=%v", err)
	}
	if _, err := svc.Detach(context.Background(), DetachRequest{Dir: dir, Force: true}, Runtime{}); err != nil {
		t.Fatal(err)
	}
}

func TestDetachRejectsFlat(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "main.go"), "package main\n")
	mustWrite(t, filepath.Join(dir, "go.mod"), "module x\n")
	_, err := New(Dependencies{Runner: noopRunner{}}).Detach(context.Background(), DetachRequest{Dir: dir, DryRun: true}, Runtime{})
	var op *OperationError
	if !errors.As(err, &op) || op.Kind != ErrorFlatModeUnsupported {
		t.Fatalf("err=%v", err)
	}
}

func TestDetachIdempotentAfterManifestRemoved(t *testing.T) {
	t.Parallel()
	dir := legacyProject(t)
	svc := New(Dependencies{Runner: noopRunner{}})
	if _, err := svc.Detach(context.Background(), DetachRequest{Dir: dir}, Runtime{}); err != nil {
		t.Fatal(err)
	}
	// noopRunner skips `go mod tidy`, so simulate its effect on go.mod.
	mustWrite(t, filepath.Join(dir, "go.mod"), "module example.com/demo\n\ngo 1.26\n")
	os.Remove(filepath.Join(dir, scaffoldManifestPath))
	res, err := svc.Detach(context.Background(), DetachRequest{Dir: dir}, Runtime{})
	if err != nil || !res.AlreadyDetached {
		t.Fatalf("res=%+v err=%v", res, err)
	}
}

func TestDetachMissingManifestWithGokartImportStillErrors(t *testing.T) {
	t.Parallel()
	dir := legacyProject(t)
	if err := os.Remove(filepath.Join(dir, scaffoldManifestPath)); err != nil {
		t.Fatal(err)
	}
	_, err := New(Dependencies{Runner: noopRunner{}}).Detach(context.Background(), DetachRequest{Dir: dir}, Runtime{})
	var op *OperationError
	if !errors.As(err, &op) || op.Kind != ErrorManifestNotFound {
		t.Fatalf("err=%v", err)
	}
}
