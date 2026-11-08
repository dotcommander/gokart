package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Stats struct {
	Database                       sql.DBStats
	PageCount, PageSize, FreePages int64
	WALBytes                       *int64
}

func Inspect(ctx context.Context, db *sql.DB, path string) (Stats, error) {
	if db == nil {
		return Stats{}, fmt.Errorf("inspect: nil db")
	}
	stats := Stats{Database: db.Stats()}
	for _, q := range []struct {
		name, query string
		value       *int64
	}{{"page count", "PRAGMA page_count", &stats.PageCount}, {"page size", "PRAGMA page_size", &stats.PageSize}, {"free pages", "PRAGMA freelist_count", &stats.FreePages}} {
		if err := db.QueryRowContext(ctx, q.query).Scan(q.value); err != nil {
			return Stats{}, fmt.Errorf("inspect %s: %w", q.name, err)
		}
	}
	if path != "" && path != ":memory:" {
		if info, err := os.Stat(path + "-wal"); err == nil {
			size := info.Size()
			stats.WALBytes = &size
		} else if !errors.Is(err, os.ErrNotExist) {
			return Stats{}, fmt.Errorf("inspect wal: %w", err)
		}
	}
	return stats, nil
}

func VacuumInto(ctx context.Context, db *sql.DB, destination string) error {
	if db == nil {
		return fmt.Errorf("vacuum into: nil db")
	}
	if destination == "" {
		return fmt.Errorf("vacuum into: destination is required")
	}
	if _, err := db.ExecContext(ctx, "VACUUM INTO ?", destination); err != nil {
		return fmt.Errorf("vacuum into %q: %w", destination, err)
	}
	return nil
}

type BackupOptions struct{ Overwrite bool }

func Backup(ctx context.Context, db *sql.DB, destination string, opts BackupOptions) error {
	if db == nil {
		return fmt.Errorf("backup: nil db")
	}
	if destination == "" {
		return fmt.Errorf("backup: destination is required")
	}
	if !opts.Overwrite {
		if _, err := os.Stat(destination); err == nil {
			return fmt.Errorf("backup: destination exists: %s", destination)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("backup: inspect destination: %w", err)
		}
	}
	temporary, err := os.CreateTemp(filepath.Dir(destination), ".gokart-backup-*")
	if err != nil {
		return fmt.Errorf("backup: create temporary sibling: %w", err)
	}
	tempPath := temporary.Name()
	temporary.Close()
	os.Remove(tempPath)
	defer os.Remove(tempPath)
	if err := VacuumInto(ctx, db, tempPath); err != nil {
		return fmt.Errorf("backup: %w", err)
	}
	checkDB, err := OpenImmutable(ctx, tempPath)
	if err != nil {
		return fmt.Errorf("backup: verify open: %w", err)
	}
	result, checkErr := QuickCheck(ctx, checkDB)
	closeErr := checkDB.Close()
	if checkErr != nil {
		return fmt.Errorf("backup: verify: %w", checkErr)
	}
	if closeErr != nil {
		return fmt.Errorf("backup: verify close: %w", closeErr)
	}
	if !result.OK {
		return fmt.Errorf("backup: quick check failed: %v", result.Messages)
	}
	if opts.Overwrite {
		if err := os.Rename(tempPath, destination); err != nil {
			return fmt.Errorf("backup: publish: %w", err)
		}
		return nil
	}
	if err := os.Link(tempPath, destination); err != nil {
		return fmt.Errorf("backup: publish: %w", err)
	}
	return os.Remove(tempPath)
}
