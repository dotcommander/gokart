package components_test

import (
	"os"
	"strings"
	"testing"
)

func TestMigrateDocumentation_FunctionSignatures(t *testing.T) {
	content, err := os.ReadFile("docs/components/migrate.md")
	if err != nil {
		t.Fatalf("Failed to read migrate.md: %v", err)
	}

	text := string(content)

	// Test for function signatures documented
	signatures := []string{
		"func Migrate",
		"func MigrateUp",
		"func MigrateDown",
		"func MigrateDownTo",
		"func MigrateReset",
		"func MigrateStatus",
		"func MigrateVersion",
		"func MigrateCreate",
		"func PostgresMigrate",
		"func SQLiteMigrate",
	}

	for _, sig := range signatures {
		if !strings.Contains(text, sig) {
			t.Errorf("Missing function signature documentation for %s", sig)
		}
	}

	// Test for code examples with function signatures
	examples := []string{
		"gokart.Migrate(ctx, db, gokart.MigrateConfig",
		"gokart.MigrateUp(ctx, db, gokart.MigrateConfig",
		"gokart.MigrateDown(ctx, db, gokart.MigrateConfig",
		"gokart.MigrateDownTo(ctx, db, gokart.MigrateConfig",
		"gokart.MigrateReset(ctx, db, gokart.MigrateConfig",
		"gokart.MigrateStatus(ctx, db, gokart.MigrateConfig",
		"gokart.MigrateVersion(ctx, db, gokart.MigrateConfig",
		"gokart.MigrateCreate(",
		"gokart.PostgresMigrate(ctx, db,",
		"gokart.SQLiteMigrate(ctx, db,",
	}

	for _, example := range examples {
		if !strings.Contains(text, example) {
			t.Errorf("Missing code example for %s", example)
		}
	}
}

func TestMigrateDocumentation_EmbedFSUsage(t *testing.T) {
	content, err := os.ReadFile("docs/components/migrate.md")
	if err != nil {
		t.Fatalf("Failed to read migrate.md: %v", err)
	}

	text := string(content)

	// Test for embed.FS documentation
	embedTerms := []string{
		"embed.FS",
		"//go:embed",
		"embed.FS",
		"var migrations embed.FS",
	}

	for _, term := range embedTerms {
		if !strings.Contains(text, term) {
			t.Errorf("Missing embed.FS documentation term: %s", term)
		}
	}

	// Test for embed.FS usage example
	embedExample := `//go:embed migrations/*.sql
var migrations embed.FS`
	if !strings.Contains(text, embedExample) {
		t.Error("Missing embed.FS usage example")
	}

	// Test for FS config field
	fsConfig := `FS:      migrations,`
	if !strings.Contains(text, fsConfig) {
		t.Error("Missing FS field in MigrateConfig example")
	}

	// Test for benefits explanation
	benefitsSection := "Benefits of embedded migrations"
	if !strings.Contains(text, benefitsSection) {
		t.Error("Missing benefits section for embedded migrations")
	}
}

func TestMigrateDocumentation_Commands(t *testing.T) {
	content, err := os.ReadFile("docs/components/migrate.md")
	if err != nil {
		t.Fatalf("Failed to read migrate.md: %v", err)
	}

	text := string(content)

	// Test for operation commands (up, down, status)
	operations := []string{
		"Migrate",
		"MigrateUp",
		"MigrateDown",
		"MigrateDownTo",
		"MigrateReset",
		"MigrateStatus",
		"MigrateVersion",
	}

	for _, op := range operations {
		// Check for heading
		if !strings.Contains(text, "### "+op) {
			t.Errorf("Missing section heading for %s operation", op)
		}

		// Check for code example
		if !strings.Contains(text, "gokart."+op+"(ctx, db,") {
			t.Errorf("Missing code example for %s operation", op)
		}
	}

	// Test for operation descriptions
	descriptions := []string{
		"Runs all pending migrations",
		"Rolls back the most recently applied migration",
		"Rolls back to a specific version",
		"Rolls back all migrations",
		"Prints the status of all migrations",
		"Returns the current migration version",
	}

	for _, desc := range descriptions {
		if !strings.Contains(text, desc) {
			t.Errorf("Missing description: %s", desc)
		}
	}

	// Test for MigrateStatus output example
	statusOutput := "Applied At"
	if !strings.Contains(text, statusOutput) {
		t.Error("Missing MigrateStatus output example")
	}
}

func TestMigrateDocumentation_MigrateConfigStruct(t *testing.T) {
	content, err := os.ReadFile("docs/components/migrate.md")
	if err != nil {
		t.Fatalf("Failed to read migrate.md: %v", err)
	}

	text := string(content)

	// Test for MigrateConfig struct documentation
	configFields := []string{
		"Dir",
		"Table",
		"Dialect",
		"FS",
		"AllowMissing",
		"NoVersioning",
	}

	for _, field := range configFields {
		if !strings.Contains(text, "| `"+field+"`") {
			t.Errorf("Missing MigrateConfig field documentation for %s", field)
		}
	}
}

func TestMigrateDocumentation_FileFormat(t *testing.T) {
	content, err := os.ReadFile("docs/components/migrate.md")
	if err != nil {
		t.Fatalf("Failed to read migrate.md: %v", err)
	}

	text := string(content)

	// Test for migration file format documentation
	fileFormatTerms := []string{
		"<version>_<name>_<type>.<ext>",
		"YYYYMMDDHHMMSS",
		"-- +goose Up",
		"-- +goose Down",
	}

	for _, term := range fileFormatTerms {
		if !strings.Contains(text, term) {
			t.Errorf("Missing file format documentation: %s", term)
		}
	}
}
