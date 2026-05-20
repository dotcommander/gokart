package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

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
