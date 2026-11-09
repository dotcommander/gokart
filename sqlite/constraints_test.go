package sqlite

import (
	"context"
	"path/filepath"
	"testing"
)

func TestIsConstraintDetectsPrimaryKeyViolation(t *testing.T) {
	t.Parallel()
	db, err := InMemory()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO t (id, v) VALUES (1, 'a')`); err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `INSERT INTO t (id, v) VALUES (1, 'b')`)
	if !IsConstraint(err) {
		t.Fatalf("IsConstraint(%v) = false, want true", err)
	}
	if IsBusy(err) || IsLocked(err) {
		t.Errorf("constraint error must not classify as busy/locked: %v", err)
	}
}

func TestOpenReadOnly(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "ro.db")
	db, err := OpenContext(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(context.Background(), `CREATE TABLE t (id INTEGER PRIMARY KEY)`); err != nil {
		db.Close()
		t.Fatal(err)
	}
	db.Close()

	ro, err := OpenReadOnly(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ro.Close() })
	if _, err := ro.ExecContext(context.Background(), `INSERT INTO t (id) VALUES (1)`); err == nil {
		t.Fatal("write to read-only database must fail")
	}
}
