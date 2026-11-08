//go:build ignore

// Example: Run goose migrations with gokart/migrate.
//
// The migration files are written to a temp directory so this example runs
// anywhere. In a real application, keep them in ./migrations or embed them
// (see the --go:embed note below). The dialect is never auto-detected.
//
// Run with: go run docs/examples/migrate/main.go
package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/dotcommander/gokart/migrate"
	"github.com/dotcommander/gokart/sqlite"
)

const migration = `-- +goose Up
CREATE TABLE notes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    body TEXT NOT NULL
);

-- +goose Down
DROP TABLE notes;
`

func main() {
	ctx := context.Background()
	dir, err := os.MkdirTemp("", "migrations")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	if err := os.WriteFile(filepath.Join(dir, "00001_create_notes.sql"), []byte(migration), 0o644); err != nil {
		log.Fatal(err)
	}

	db, err := sqlite.OpenContext(ctx, filepath.Join(os.TempDir(), "example-migrate.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	cfg := migrate.Config{Dir: dir, Dialect: "sqlite3"}
	if err := migrate.Up(ctx, db, cfg); err != nil {
		log.Fatal(err)
	}
	log.Println("migrated up")

	if _, err := db.ExecContext(ctx, "INSERT INTO notes(body) VALUES (?)", "hello"); err != nil {
		log.Fatal(err)
	}

	statuses, err := migrate.MigrationStatuses(ctx, db, cfg)
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range statuses {
		log.Printf("version %d applied=%v at %v", s.Version, s.Applied, s.AppliedAt)
	}

	// For a binary that runs anywhere, embed migrations and set Config.FS:
	//   //go:embed migrations/*.sql
	//   var files embed.FS
	//   migrate.Up(ctx, db, migrate.Config{FS: files, Dir: "migrations", Dialect: "sqlite3"})
}
