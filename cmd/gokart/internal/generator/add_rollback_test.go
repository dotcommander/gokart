package generator

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestAddRollbackRevertsWritesOnFailure(t *testing.T) {
	t.Parallel()
	dir := setupAddTestProject(t, setupAddTestOpts{Module: "example.com/myapp", ManifestVersion: scaffoldManifestV2, TemplateRoot: "templates/structured", Mode: modeStructured, ExtraFiles: map[string]string{"internal/existing.go": "package existing // ORIGINAL\n"}})
	req := addRequest{Dir: dir}
	plan := &addPlan{RenderedFiles: map[string][]byte{"internal/new.go": []byte("package generated\n"), "internal/existing.go": []byte("package existing // OVERWRITTEN\n")}, RenderedPaths: []string{"internal/existing.go", "internal/new.go"}}
	journal, applied, err := applyAddFileWrites(req, plan)
	if err != nil {
		t.Fatalf("applyAddFileWrites: %v", err)
	}
	forced := &OperationError{Kind: ErrorScaffoldFailed, Err: errors.New("deliberate failure")}
	if err := rollbackWithError(forced, applied, journal); err == nil {
		t.Fatal("expected rollback error")
	}
	if _, err := os.Stat(filepath.Join(dir, "internal/new.go")); !os.IsNotExist(err) {
		t.Fatalf("created file remains: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "internal/existing.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "package existing // ORIGINAL\n" {
		t.Fatalf("file not restored: %q", content)
	}
}
