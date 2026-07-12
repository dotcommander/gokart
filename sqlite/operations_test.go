package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestHealthInspectAndMaintenance(t *testing.T) {
	db, err := InMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	for name, check := range map[string]func(context.Context, *sql.DB) (CheckResult, error){"quick": QuickCheck, "integrity": IntegrityCheck} {
		t.Run(name, func(t *testing.T) {
			result, err := check(ctx, db)
			if err != nil || !result.OK {
				t.Fatalf("result=%+v err=%v", result, err)
			}
		})
	}
	violations, err := ForeignKeyCheck(ctx, db)
	if err != nil || len(violations) != 0 {
		t.Fatalf("violations=%v err=%v", violations, err)
	}
	stats, err := Inspect(ctx, db, ":memory:")
	if err != nil || stats.PageSize == 0 {
		t.Fatalf("stats=%+v err=%v", stats, err)
	}
	if err := Optimize(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := Vacuum(ctx, db); err != nil {
		t.Fatal(err)
	}
}

func TestSavepointRollsBackOnlyNestedWork(t *testing.T) {
	db, err := InMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if _, err := db.Exec("CREATE TABLE item (value TEXT)"); err != nil {
		t.Fatal(err)
	}
	err = Transaction(ctx, db, func(tx *sql.Tx) error {
		if _, err := tx.Exec("INSERT INTO item VALUES ('kept')"); err != nil {
			return err
		}
		expected := errors.New("rollback nested")
		got := Savepoint(ctx, tx, "nested_1", func() error {
			if _, err := tx.Exec("INSERT INTO item VALUES ('removed')"); err != nil {
				return err
			}
			return expected
		})
		if !errors.Is(got, expected) {
			t.Fatalf("got %v", got)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow("SELECT count(*) FROM item").Scan(&count); err != nil || count != 1 {
		t.Fatalf("count=%d err=%v", count, err)
	}
}

func TestRetryValidationCancellationAndNonSQLiteError(t *testing.T) {
	if err := Retry(context.Background(), RetryPolicy{}, func() error { return nil }); err == nil {
		t.Fatal("expected attempts validation")
	}
	want := errors.New("stop")
	calls := 0
	if got := Retry(context.Background(), RetryPolicy{Attempts: 3}, func() error { calls++; return want }); !errors.Is(got, want) || calls != 1 {
		t.Fatalf("got=%v calls=%d", got, calls)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls = 0
	if got := Retry(ctx, RetryPolicy{Attempts: 2, InitialDelay: time.Millisecond}, func() error { calls++; return nil }); !errors.Is(got, context.Canceled) || calls != 0 {
		t.Fatalf("got=%v calls=%d", got, calls)
	}
}

func TestVacuumIntoAndBackup(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	source := filepath.Join(dir, "source.db")
	db, err := Open(source)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE item (value TEXT); INSERT INTO item VALUES ('copied')"); err != nil {
		t.Fatal(err)
	}
	vacuumCopy := filepath.Join(dir, "vacuum.db")
	if err := VacuumInto(ctx, db, vacuumCopy); err != nil {
		t.Fatal(err)
	}
	assertCopiedRow(t, vacuumCopy)
	backup := filepath.Join(dir, "backup.db")
	if err := Backup(ctx, db, backup, BackupOptions{}); err != nil {
		t.Fatal(err)
	}
	assertCopiedRow(t, backup)
	if err := Backup(ctx, db, backup, BackupOptions{}); err == nil {
		t.Fatal("expected existing destination error")
	}
	if err := Backup(ctx, db, backup, BackupOptions{Overwrite: true}); err != nil {
		t.Fatal(err)
	}
}

func assertCopiedRow(t *testing.T, path string) {
	t.Helper()
	db, err := OpenImmutable(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var value string
	if err := db.QueryRow("SELECT value FROM item").Scan(&value); err != nil || value != "copied" {
		t.Fatalf("value=%q err=%v", value, err)
	}
}

func TestWALCheckpoint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wal.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE item (id INTEGER)"); err != nil {
		t.Fatal(err)
	}
	if _, err := WALCheckpointTruncate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if _, err := WALCheckpoint(context.Background(), db, "INVALID"); err == nil {
		t.Fatal("expected invalid mode")
	}
}

func TestOperationNilInputs(t *testing.T) {
	ctx := context.Background()
	if _, err := QuickCheck(ctx, nil); err == nil {
		t.Fatal("quick check nil")
	}
	if _, err := Inspect(ctx, nil, ""); err == nil {
		t.Fatal("inspect nil")
	}
	if err := Optimize(ctx, nil); err == nil {
		t.Fatal("optimize nil")
	}
	if err := Savepoint(ctx, nil, "ok", func() error { return nil }); err == nil {
		t.Fatal("savepoint nil")
	}
	if err := Backup(ctx, nil, "x", BackupOptions{}); err == nil {
		t.Fatal("backup nil")
	}
	if err := VacuumInto(ctx, nil, ""); err == nil {
		t.Fatal("vacuum nil")
	}
}
