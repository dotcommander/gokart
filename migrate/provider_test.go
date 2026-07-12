package migrate

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	_ "modernc.org/sqlite"
)

func TestEmbeddedMigrationLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	migrations := fstest.MapFS{
		"migrations/00001_items.sql": {Data: []byte(`-- +goose Up
CREATE TABLE items (id INTEGER PRIMARY KEY);
-- +goose Down
DROP TABLE items;
`)},
		"migrations/00002_labels.sql": {Data: []byte(`-- +goose Up
CREATE TABLE labels (id INTEGER PRIMARY KEY);
-- +goose Down
DROP TABLE labels;
`)},
	}
	cfg := Config{Dir: "migrations", Dialect: "sqlite3", FS: migrations}
	db := openSQLiteForMigrationTest(t, filepath.Join(t.TempDir(), "lifecycle.db"))

	if err := Up(ctx, db, cfg); err != nil {
		t.Fatalf("up embedded migrations: %v", err)
	}
	assertMigrationVersion(t, ctx, db, cfg, 2)

	if err := Down(ctx, db, cfg); err != nil {
		t.Fatalf("down embedded migration: %v", err)
	}
	assertMigrationVersion(t, ctx, db, cfg, 1)
	assertTableExists(t, ctx, db, "items", true)
	assertTableExists(t, ctx, db, "labels", false)

	if err := Up(ctx, db, cfg); err != nil {
		t.Fatalf("reapply embedded migration: %v", err)
	}
	if err := Reset(ctx, db, cfg); err != nil {
		t.Fatalf("reset embedded migrations: %v", err)
	}
	assertMigrationVersion(t, ctx, db, cfg, 0)
	assertTableExists(t, ctx, db, "items", false)
	assertTableExists(t, ctx, db, "labels", false)
}

func TestEmbeddedMigrationFailureReportsError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	migrations := fstest.MapFS{
		"migrations/00001_broken.sql": {Data: []byte(`-- +goose Up
CREATE TABL broken (id INTEGER PRIMARY KEY);
`)},
	}
	cfg := Config{Dir: "migrations", Dialect: "sqlite3", FS: migrations}
	db := openSQLiteForMigrationTest(t, filepath.Join(t.TempDir(), "failed.db"))

	err := Up(ctx, db, cfg)
	if err == nil {
		t.Fatal("broken embedded migration unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), "migration failed") {
		t.Fatalf("error = %q, want migration failure context", err)
	}
	assertMigrationVersion(t, ctx, db, cfg, 0)
}

func TestProviderOperationsDoNotLeakConfiguration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	migrationDir := writeTestMigrations(t)

	customDB := openSQLiteForMigrationTest(t, filepath.Join(t.TempDir(), "custom.db"))
	customCfg := Config{Dir: migrationDir, Dialect: "sqlite3", Table: "custom_versions"}
	if err := Up(ctx, customDB, customCfg); err != nil {
		t.Fatalf("up custom table: %v", err)
	}
	if err := Down(ctx, customDB, customCfg); err != nil {
		t.Fatalf("down custom table: %v", err)
	}

	defaultDB := openSQLiteForMigrationTest(t, filepath.Join(t.TempDir(), "default.db"))
	defaultCfg := Config{Dir: migrationDir, Dialect: "sqlite3"}
	if err := Up(ctx, defaultDB, defaultCfg); err != nil {
		t.Fatalf("up default table: %v", err)
	}
	if err := Down(ctx, defaultDB, defaultCfg); err != nil {
		t.Fatalf("down default table after custom rollback: %v", err)
	}
	assertMigrationVersion(t, ctx, defaultDB, defaultCfg, 1)
}

func TestMigrationStatusesReportsAppliedAndPending(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	migrationDir := writeTestMigrations(t)
	db := openSQLiteForMigrationTest(t, filepath.Join(t.TempDir(), "status.db"))
	cfg := Config{Dir: migrationDir, Dialect: "sqlite3"}
	if err := Up(ctx, db, cfg); err != nil {
		t.Fatalf("up: %v", err)
	}
	if err := Down(ctx, db, cfg); err != nil {
		t.Fatalf("down: %v", err)
	}

	statuses, err := MigrationStatuses(ctx, db, cfg)
	if err != nil {
		t.Fatalf("migration statuses: %v", err)
	}
	if len(statuses) != 2 {
		t.Fatalf("status count = %d, want 2", len(statuses))
	}
	if statuses[0].Version != 1 || !statuses[0].Applied || statuses[0].AppliedAt.IsZero() {
		t.Fatalf("first status = %#v", statuses[0])
	}
	if statuses[1].Version != 2 || statuses[1].Applied || !statuses[1].AppliedAt.IsZero() {
		t.Fatalf("second status = %#v", statuses[1])
	}
}

func TestProviderRequiresDialect(t *testing.T) {
	t.Parallel()

	db := openSQLiteForMigrationTest(t, filepath.Join(t.TempDir(), "missing-dialect.db"))
	if _, err := MigrationStatuses(context.Background(), db, Config{Dir: t.TempDir()}); err == nil {
		t.Fatal("migration statuses without dialect unexpectedly succeeded")
	}
}

func writeTestMigrations(t *testing.T) string {
	t.Helper()

	dir := filepath.Join(t.TempDir(), "migrations")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create migration directory: %v", err)
	}
	writeMigration(t, filepath.Join(dir, "00001_init.sql"), `-- +goose Up
CREATE TABLE t1 (id INTEGER PRIMARY KEY);
-- +goose Down
DROP TABLE t1;
`)
	writeMigration(t, filepath.Join(dir, "00002_more.sql"), `-- +goose Up
CREATE TABLE t2 (id INTEGER PRIMARY KEY);
-- +goose Down
DROP TABLE t2;
`)
	return dir
}

func openSQLiteForMigrationTest(t *testing.T, path string) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		db.Close()
		t.Fatalf("ping sqlite db: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close sqlite db: %v", err)
		}
	})
	return db
}

func writeMigration(t *testing.T, path, contents string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write migration %s: %v", path, err)
	}
}

func assertMigrationVersion(t *testing.T, ctx context.Context, db *sql.DB, cfg Config, want int64) {
	t.Helper()

	got, err := Version(ctx, db, cfg)
	if err != nil {
		t.Fatalf("migration version: %v", err)
	}
	if got != want {
		t.Fatalf("migration version = %d, want %d", got, want)
	}
}

func assertTableExists(t *testing.T, ctx context.Context, db *sql.DB, table string, want bool) {
	t.Helper()

	var count int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table,
	).Scan(&count); err != nil {
		t.Fatalf("inspect table %q: %v", table, err)
	}
	if got := count == 1; got != want {
		t.Fatalf("table %q exists = %t, want %t", table, got, want)
	}
}
