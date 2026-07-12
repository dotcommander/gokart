package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type failingAddJSONWriter struct {
	err error
}

func (w failingAddJSONWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func TestRunAddCommandAcquiresLockBeforePlanning(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, scaffoldManifestPath)
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		t.Fatalf("create manifest dir: %v", err)
	}
	if err := os.WriteFile(manifestPath, []byte("not json"), 0644); err != nil {
		t.Fatalf("seed malformed manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gokart.lock"), []byte(fmt.Sprintf("pid=%d\n", os.Getpid())), 0600); err != nil {
		t.Fatalf("seed lock: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalWD) })

	err = runAddCommand(newAddCommand(), []string{"sqlite"})
	var lockErr *ApplyLockError
	if !errors.As(err, &lockErr) {
		t.Fatalf("expected lock error before malformed manifest was planned, got %v", err)
	}
}

func TestAddConflictsOnUserModifiedFile(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	// Sequential: uses os.Chdir which mutates process-wide working directory.
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

func TestRunAddCommandJSONReturnsWriterFailure(t *testing.T) {
	// Sequential: uses os.Chdir.
	dir := setupAddTestProject(t, setupAddTestOpts{
		Module:          "example.com/myapp",
		ManifestVersion: scaffoldManifestV2,
		TemplateRoot:    "templates/structured",
		Mode:            modeStructured,
		Integrations:    &manifestIntegrations{SQLite: true},
	})
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})

	writeErr := errors.New("write JSON")
	cmd := newAddCommand()
	cmd.SetOut(failingAddJSONWriter{err: writeErr})
	mustSetFlagTrue(t, cmd, addFlagJSON)

	err = runAddCommand(cmd, []string{integrationSQLite})
	if err == nil {
		t.Fatal("expected JSON writer failure")
	}
	var cmdErr *commandError
	if !errors.As(err, &cmdErr) {
		t.Fatalf("error type = %T, want *commandError", err)
	}
	if cmdErr.Code != errorCodeJSONEncodeFailed {
		t.Fatalf("error code = %q, want %q", cmdErr.Code, errorCodeJSONEncodeFailed)
	}
	if cmdErr.ExitCode != exitCodeJSONEncodeFailed {
		t.Fatalf("exit code = %d, want %d", cmdErr.ExitCode, exitCodeJSONEncodeFailed)
	}
	if !errors.Is(err, writeErr) {
		t.Fatalf("error %v does not wrap writer failure", err)
	}
}

// TestAddWarnsOnGeneratorVersionSkew covers the version-skew warning path:
// a project scaffolded by an older gokart version must produce an operator-visible
// WARN from planAddChanges, and the skew alone must not cause a fatal error.
func TestAddWarnsOnGeneratorVersionSkew(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name             string
		generatorVersion string
		wantWarn         bool
	}{
		{
			name:             "skew warns",
			generatorVersion: "v0.0.0-test-old",
			wantWarn:         true,
		},
		{
			name:             "matching version is silent",
			generatorVersion: gokartVersion,
			wantWarn:         false,
		},
		{
			name:             "empty version is silent (pre-existing manifest)",
			generatorVersion: "",
			wantWarn:         false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dir := setupAddTestProject(t, setupAddTestOpts{
				Module:           "example.com/myapp",
				ManifestVersion:  scaffoldManifestV2,
				TemplateRoot:     "templates/structured",
				Mode:             "structured",
				Integrations:     &manifestIntegrations{},
				GeneratorVersion: tc.generatorVersion,
			})

			// planAddChanges is the real code path for the skew check.
			// Request an integration that is already present so we can isolate
			// the warning without needing full template rendering.
			output := &addCommandOutput{}
			_, _ = planAddChanges(addRequest{Dir: dir, Integrations: []string{"sqlite"}}, output)

			got := strings.Join(output.Warnings, "\n")
			if tc.wantWarn {
				if !strings.Contains(got, tc.generatorVersion) {
					t.Fatalf("expected warning containing %q, got %q", tc.generatorVersion, got)
				}
				if !strings.Contains(got, gokartVersion) {
					t.Fatalf("expected warning containing running version %q, got %q", gokartVersion, got)
				}
				if !strings.Contains(got, "scaffolded by gokart") {
					t.Fatalf("expected skew warning text, got %q", got)
				}
			} else {
				if strings.Contains(got, "scaffolded by gokart") {
					t.Fatalf("expected no skew warning, got %q", got)
				}
			}
		})
	}
}
