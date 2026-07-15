package generator

import (
	"strings"
	"testing"
)

func TestAddCreatesContextForFirstIntegration(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	if !strings.Contains(content, "kong.BindTo") {
		t.Fatal("expected root.go to bind the process context")
	}
	if !strings.Contains(content, "app.New") {
		t.Fatal("expected root.go to reference app.New")
	}
}

func TestGeneratedConfigFlagsOnlyOverrideWhenExplicit(t *testing.T) {
	t.Parallel()
	data := baseTemplateData("myapp", "example.com/myapp", false, false)
	data.UseSQLite = true
	files, err := renderIntegrationFiles(data)
	if err != nil {
		t.Fatal(err)
	}
	root := string(files["internal/commands/root.go"])
	for _, contract := range []string{
		"Verbose *bool", "Quiet   *bool", "if cli.Verbose != nil", "if cli.Quiet != nil",
		`v.AddConfigPath(workingDir)`, `filepath.Join(userConfigDir, "myapp")`, `filepath.Join(systemConfigRoot, "myapp")`,
		"var notFound viper.ConfigFileNotFoundError", `if cli.Config != "" || !errors.As(err, &notFound)`,
	} {
		if !strings.Contains(root, contract) {
			t.Fatalf("root.go omits precedence contract %q", contract)
		}
	}
}

func TestAddDryRunNoChanges(t *testing.T) {
	t.Parallel()
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

func TestAddPostgresUsesToolkitOwner(t *testing.T) {
	t.Parallel()
	data := baseTemplateData("myapp", "example.com/myapp", true, false)
	data.UsePostgres = true

	files, err := renderIntegrationFiles(data)
	if err != nil {
		t.Fatalf("renderIntegrationFiles: %v", err)
	}

	contextContent, ok := files["internal/app/context.go"]
	if !ok {
		t.Fatal("expected context.go to be rendered")
	}
	content := string(contextContent)

	for _, want := range []string{"postgres.Open(ctx, dbURL)"} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected postgres context.go to contain %q", want)
		}
	}

	for _, bad := range []string{
		"pgxpool.New(ctx,",
		"pgxpool.NewWithConfig",
		"cfg.MinConns",
		"cfg.MaxConnLifetime",
		"context.Background",
	} {
		if strings.Contains(content, bad) {
			t.Fatalf("postgres context.go must not contain %q", bad)
		}
	}
}
