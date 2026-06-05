package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var errDeliberateAddFailure = errors.New("deliberate add failure for rollback test")

// TestAddRollbackRevertsWritesOnFailure exercises the journaled add write loop
// and asserts that a downstream failure reverts the filesystem to its pre-call
// state: newly created files are removed and overwritten files are restored to
// their original content. This is the exact rollback path applyAddChanges runs
// when addGoDependencies or updateAddManifest fails.
func TestAddRollbackRevertsWritesOnFailure(t *testing.T) {
	t.Parallel()

	dir := setupAddTestProject(t, setupAddTestOpts{
		Module:          "example.com/myapp",
		ManifestVersion: scaffoldManifestV2,
		TemplateRoot:    "templates/structured",
		Mode:            modeStructured,
		ExtraFiles: map[string]string{
			"internal/existing.go": "package existing // ORIGINAL\n",
		},
	})

	req := addRequest{Dir: dir, JSONOutput: true}
	plan := &addPlan{
		RenderedFiles: map[string][]byte{
			"internal/new_a.go":    []byte("package a // GENERATED\n"),
			"internal/new_b.go":    []byte("package b // GENERATED\n"),
			"internal/existing.go": []byte("package existing // OVERWRITTEN\n"),
		},
		RenderedPaths: []string{
			"internal/existing.go",
			"internal/new_a.go",
			"internal/new_b.go",
		},
	}

	journal, applied, err := applyAddFileWrites(req, plan)
	if err != nil {
		t.Fatalf("applyAddFileWrites: %v", err)
	}
	if len(applied) != len(plan.RenderedPaths) {
		t.Fatalf("expected %d journaled actions, got %d", len(plan.RenderedPaths), len(applied))
	}

	// Happy-path sanity: files landed before we simulate the downstream failure.
	if got, _ := os.ReadFile(filepath.Join(dir, "internal/new_a.go")); string(got) != "package a // GENERATED\n" {
		t.Fatalf("new_a.go not written, got %q", string(got))
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "internal/existing.go")); string(got) != "package existing // OVERWRITTEN\n" {
		t.Fatalf("existing.go not overwritten, got %q", string(got))
	}

	// Simulate addGoDependencies / updateAddManifest failing after the writes.
	forced := wrapAddFlowError(errDeliberateAddFailure, errorCodeScaffoldFailed, exitCodeScaffoldFailed)
	if rbErr := rollbackWithError(forced, applied, journal); rbErr == nil {
		t.Fatal("rollbackWithError returned nil; expected the original failure to be propagated")
	}

	// Created files must be gone.
	for _, rel := range []string{"internal/new_a.go", "internal/new_b.go"} {
		if _, statErr := os.Stat(filepath.Join(dir, rel)); !os.IsNotExist(statErr) {
			t.Fatalf("expected %s to be removed by rollback, stat err = %v", rel, statErr)
		}
	}

	// Overwritten file must be restored to its original content (modify-in-place
	// distinguished from create-new).
	restored, readErr := os.ReadFile(filepath.Join(dir, "internal/existing.go"))
	if readErr != nil {
		t.Fatalf("read restored existing.go: %v", readErr)
	}
	if string(restored) != "package existing // ORIGINAL\n" {
		t.Fatalf("existing.go not restored to original, got %q", string(restored))
	}

	// Journal must be cleaned up after rollback.
	if _, statErr := os.Stat(filepath.Join(dir, ".gokart", "tx")); !os.IsNotExist(statErr) {
		t.Fatalf("expected journal tx dir to be cleaned up, stat err = %v", statErr)
	}
}

func TestApplyAddChangesRollsBackDependencyFilesOnManifestFailure(t *testing.T) {
	dir := setupAddTestProject(t, setupAddTestOpts{
		Module:          "example.com/myapp",
		ManifestVersion: scaffoldManifestV2,
		TemplateRoot:    "templates/structured",
		Mode:            modeStructured,
	})
	originalGoMod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatalf("read original go.mod: %v", err)
	}

	req := addRequest{Dir: dir, JSONOutput: true}
	plan := &addPlan{
		Manifest: &scaffoldManifest{Version: scaffoldManifestV2},
		ToAdd:    []string{integrationSQLite},
		Data:     TemplateData{Module: "example.com/myapp", UseSQLite: true},
		RenderedFiles: map[string][]byte{
			"internal/app/context.go": []byte("package app // generated\n"),
		},
		RenderedPaths: []string{"internal/app/context.go"},
	}
	output := &addCommandOutput{}

	originalAddDeps := addGoDependenciesFunc
	originalUpdateManifest := updateAddManifestFunc
	addGoDependenciesFunc = func(_ context.Context, dir string, _ []string, _ bool) error {
		mutatedGoMod := append([]byte(nil), originalGoMod...)
		mutatedGoMod = append(mutatedGoMod, []byte("\nrequire example.com/dep v1.2.3\n")...)
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), mutatedGoMod, 0644); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, "go.sum"), []byte("example.com/dep v1.2.3 h1:test\n"), 0644)
	}
	updateAddManifestFunc = func(string, *scaffoldManifest, TemplateData, map[string][]byte) error {
		return errDeliberateAddFailure
	}
	t.Cleanup(func() {
		addGoDependenciesFunc = originalAddDeps
		updateAddManifestFunc = originalUpdateManifest
	})

	err = applyAddChanges(context.Background(), req, plan, output)
	if err == nil {
		t.Fatal("expected applyAddChanges to fail")
	}
	if !strings.Contains(err.Error(), errDeliberateAddFailure.Error()) {
		t.Fatalf("expected deliberate failure in error, got %v", err)
	}

	gotGoMod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatalf("read rolled-back go.mod: %v", err)
	}
	if string(gotGoMod) != string(originalGoMod) {
		t.Fatalf("go.mod was not restored:\n got %q\nwant %q", gotGoMod, originalGoMod)
	}
	if _, err := os.Stat(filepath.Join(dir, "go.sum")); !os.IsNotExist(err) {
		t.Fatalf("go.sum should have been removed by rollback, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "internal/app/context.go")); !os.IsNotExist(err) {
		t.Fatalf("rendered file should have been removed by rollback, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".gokart", "tx")); !os.IsNotExist(err) {
		t.Fatalf("expected journal tx dir to be cleaned up, stat err = %v", err)
	}
}
