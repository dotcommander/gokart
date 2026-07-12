package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	moderncsqlite "modernc.org/sqlite"
)

// WALCheckpointMode names a supported SQLite WAL checkpoint mode.
type WALCheckpointMode string

const (
	WALCheckpointModePassive  WALCheckpointMode = "PASSIVE"
	WALCheckpointModeFull     WALCheckpointMode = "FULL"
	WALCheckpointModeRestart  WALCheckpointMode = "RESTART"
	WALCheckpointModeTruncate WALCheckpointMode = "TRUNCATE"
)

type WALCheckpointResult struct{ Busy, LogFrames, CheckpointedFrames int }

func WALCheckpoint(ctx context.Context, db *sql.DB, mode WALCheckpointMode) (WALCheckpointResult, error) {
	if db == nil {
		return WALCheckpointResult{}, fmt.Errorf("wal checkpoint: nil db")
	}
	switch mode {
	case WALCheckpointModePassive, WALCheckpointModeFull, WALCheckpointModeRestart, WALCheckpointModeTruncate:
	default:
		return WALCheckpointResult{}, fmt.Errorf("wal checkpoint: unsupported mode %q", mode)
	}
	var result WALCheckpointResult
	if err := db.QueryRowContext(ctx, "PRAGMA wal_checkpoint("+string(mode)+")").Scan(&result.Busy, &result.LogFrames, &result.CheckpointedFrames); err != nil {
		return WALCheckpointResult{}, fmt.Errorf("wal checkpoint %s: %w", mode, err)
	}
	return result, nil
}
func WALCheckpointTruncate(ctx context.Context, db *sql.DB) (WALCheckpointResult, error) {
	return WALCheckpoint(ctx, db, WALCheckpointModeTruncate)
}

type CheckResult struct {
	OK       bool
	Messages []string
}
type ForeignKeyViolation struct {
	Table       string
	RowID       *int64
	ParentTable string
	ForeignKey  int
}

func QuickCheck(ctx context.Context, db *sql.DB) (CheckResult, error) {
	return runCheck(ctx, db, "PRAGMA quick_check", "quick check")
}
func IntegrityCheck(ctx context.Context, db *sql.DB) (CheckResult, error) {
	return runCheck(ctx, db, "PRAGMA integrity_check", "integrity check")
}
func runCheck(ctx context.Context, db *sql.DB, query, name string) (CheckResult, error) {
	if db == nil {
		return CheckResult{}, fmt.Errorf("%s: nil db", name)
	}
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return CheckResult{}, fmt.Errorf("%s: %w", name, err)
	}
	defer rows.Close()
	result := CheckResult{OK: true}
	for rows.Next() {
		var message string
		if err := rows.Scan(&message); err != nil {
			return CheckResult{}, fmt.Errorf("%s result: %w", name, err)
		}
		result.Messages = append(result.Messages, message)
		result.OK = result.OK && message == "ok"
	}
	if err := rows.Err(); err != nil {
		return CheckResult{}, fmt.Errorf("%s rows: %w", name, err)
	}
	if len(result.Messages) == 0 {
		result.OK = false
	}
	return result, nil
}
func ForeignKeyCheck(ctx context.Context, db *sql.DB) ([]ForeignKeyViolation, error) {
	if db == nil {
		return nil, fmt.Errorf("foreign key check: nil db")
	}
	rows, err := db.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return nil, fmt.Errorf("foreign key check: %w", err)
	}
	defer rows.Close()
	var out []ForeignKeyViolation
	for rows.Next() {
		var v ForeignKeyViolation
		var row sql.NullInt64
		if err := rows.Scan(&v.Table, &row, &v.ParentTable, &v.ForeignKey); err != nil {
			return nil, fmt.Errorf("foreign key check result: %w", err)
		}
		if row.Valid {
			value := row.Int64
			v.RowID = &value
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("foreign key check rows: %w", err)
	}
	return out, nil
}

type RetryPolicy struct {
	Attempts               int
	InitialDelay, MaxDelay time.Duration
}

func IsBusy(err error) bool       { return sqlitePrimaryCode(err) == 5 }
func IsLocked(err error) bool     { return sqlitePrimaryCode(err) == 6 }
func IsConstraint(err error) bool { return sqlitePrimaryCode(err) == 19 }
func sqlitePrimaryCode(err error) int {
	var sqliteErr *moderncsqlite.Error
	if !errors.As(err, &sqliteErr) {
		return -1
	}
	return sqliteErr.Code() & 0xff
}
func Retry(ctx context.Context, policy RetryPolicy, fn func() error) error {
	if policy.Attempts < 1 {
		return fmt.Errorf("retry sqlite operation: attempts must be positive")
	}
	if policy.InitialDelay < 0 || policy.MaxDelay < 0 || policy.MaxDelay > 0 && policy.InitialDelay > policy.MaxDelay {
		return fmt.Errorf("retry sqlite operation: invalid delays")
	}
	if fn == nil {
		return fmt.Errorf("retry sqlite operation: nil callback")
	}
	delay := policy.InitialDelay
	for attempt := 1; attempt <= policy.Attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := fn()
		if err == nil {
			return nil
		}
		if (!IsBusy(err) && !IsLocked(err)) || attempt == policy.Attempts {
			return err
		}
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				return ctx.Err()
			case <-timer.C:
			}
		}
		if delay > 0 {
			delay *= 2
			if policy.MaxDelay > 0 && delay > policy.MaxDelay {
				delay = policy.MaxDelay
			}
		}
	}
	panic("unreachable")
}

func Savepoint(ctx context.Context, tx *sql.Tx, name string, fn func() error) (err error) {
	if tx == nil {
		return fmt.Errorf("savepoint: nil transaction")
	}
	if !validSavepointName(name) {
		return fmt.Errorf("savepoint: invalid name %q", name)
	}
	if fn == nil {
		return fmt.Errorf("savepoint: nil callback")
	}
	if _, err := tx.ExecContext(ctx, "SAVEPOINT "+name); err != nil {
		return fmt.Errorf("create savepoint %s: %w", name, err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = rollbackSavepoint(ctx, tx, name)
			panic(p)
		}
	}()
	if err := fn(); err != nil {
		return errors.Join(err, rollbackSavepoint(ctx, tx, name))
	}
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if _, err := tx.ExecContext(cleanup, "RELEASE SAVEPOINT "+name); err != nil {
		return fmt.Errorf("release savepoint %s: %w", name, err)
	}
	return nil
}
func rollbackSavepoint(ctx context.Context, tx *sql.Tx, name string) error {
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	var errs []error
	if _, err := tx.ExecContext(cleanup, "ROLLBACK TO SAVEPOINT "+name); err != nil {
		errs = append(errs, err)
	}
	if _, err := tx.ExecContext(cleanup, "RELEASE SAVEPOINT "+name); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
func validSavepointName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '_' || i > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

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

func Optimize(ctx context.Context, db *sql.DB) error {
	return execMaintenance(ctx, db, "optimize", "PRAGMA optimize")
}
func Vacuum(ctx context.Context, db *sql.DB) error {
	return execMaintenance(ctx, db, "vacuum", "VACUUM")
}
func execMaintenance(ctx context.Context, db *sql.DB, name, query string) error {
	if db == nil {
		return fmt.Errorf("%s: nil db", name)
	}
	if _, err := db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
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
