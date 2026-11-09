package migrate

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	if cfg.Dir != "migrations" {
		t.Errorf("Dir = %q, want migrations", cfg.Dir)
	}
	if cfg.Table != "goose_db_version" {
		t.Errorf("Table = %q, want goose_db_version", cfg.Table)
	}
}

func TestCreateWritesMigrationFile(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "migrations")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := Create(dir, "add_things", "sql"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("dir entries = %d, want 1", len(entries))
	}
	if !strings.HasSuffix(entries[0].Name(), "_add_things.sql") {
		t.Errorf("file = %q", entries[0].Name())
	}
}

func TestStatusAndSQLiteConvenience(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	migrations := fstest.MapFS{
		"migrations/00001_items.sql": {Data: []byte("-- +goose Up\nCREATE TABLE items (id INTEGER PRIMARY KEY);\n-- +goose Down\nDROP TABLE items;\n")},
	}
	cfg := Config{Dir: "migrations", Dialect: "sqlite3", FS: migrations}
	db := openSQLiteForMigrationTest(t, filepath.Join(t.TempDir(), "status.db"))

	if err := Up(ctx, db, cfg); err != nil {
		t.Fatalf("Up: %v", err)
	}
	if err := Status(ctx, db, cfg); err != nil {
		t.Fatalf("Status: %v", err)
	}

	// SQLite convenience wrapper exercises the same path.
	db2 := openSQLiteForMigrationTest(t, filepath.Join(t.TempDir(), "convenience.db"))
	dir := writeTestMigrations(t)
	if err := SQLite(ctx, db2, dir); err != nil {
		t.Fatalf("SQLite: %v", err)
	}
	if got, err := Version(ctx, db2, Config{Dir: dir, Dialect: "sqlite3"}); err != nil || got != 2 {
		t.Errorf("after SQLite: version=%d err=%v, want 2/nil", got, err)
	}
}
