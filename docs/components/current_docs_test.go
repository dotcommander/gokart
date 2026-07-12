package components_test

import (
	"os"
	"strings"
	"testing"
)

func assertDocContains(t *testing.T, file string, required ...string) string {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)
	for _, text := range required {
		if !strings.Contains(doc, text) {
			t.Errorf("%s omits %q", file, text)
		}
	}
	return doc
}

func TestPostgresDocMatchesPublicSurface(t *testing.T) {
	assertDocContains(t, "postgres.md", "postgres.Open", "OpenWithConfig", "FromEnv", "NewPool", "DSN", "Transaction", "NewPostgresIdentifier", "NewPostgresIndexIdentifier", "25", "5", "one-hour", "30-minute", "one-minute")
}

func TestSQLiteDocMatchesPublicSurface(t *testing.T) {
	assertDocContains(t, "sqlite.md", "pure-Go", "OpenReadOnly", "OpenImmutable", "ReadHeavyConfig", "ResolveConfig", "TransactionWithOptions", "Savepoint", "QuickCheck", "IntegrityCheck", "ForeignKeyCheck", "Inspect", "Optimize", "Vacuum", "Backup", "WALCheckpoint", "Retry", "IsBusy", "IsLocked", "IsConstraint")
}

func TestMigrateDocMatchesPublicSurface(t *testing.T) {
	doc := assertDocContains(t, "migrate.md", "Dialect", "never auto-detected", "migrate.Up", "migrate.Down", "migrate.DownTo", "migrate.Reset", "MigrationStatuses", "migrate.Version", "migrate.Create", "migrate.Postgres", "migrate.SQLite", "embed.FS")
	if strings.Contains(doc, "Status prints") {
		t.Error("migrate docs claim Status prints")
	}
}

func TestValidateDocMatchesPublicSurface(t *testing.T) {
	doc := assertDocContains(t, "validate.md", "NewValidator", "NewStandardValidator", "ValidationErrors", "BindAndValidate", "notblank", "UseJSONNames:false")
	if strings.Contains(doc, "UseJSONNames:false selects") {
		t.Error("validate docs claim false is distinguishable from zero config")
	}
}
