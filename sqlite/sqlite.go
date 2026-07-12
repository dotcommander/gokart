// Package sqlite provides SQLite database utilities wrapping modernc.org/sqlite.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/dotcommander/gokart/internal/sqltx"
	_ "modernc.org/sqlite"
)

type Mode string

const (
	ModeReadWrite Mode = "read-write"
	ModeReadOnly  Mode = "read-only"
	ModeImmutable Mode = "immutable"
	ModeMemory    Mode = "memory"
)

type JournalMode string

const (
	JournalModeDelete   JournalMode = "DELETE"
	JournalModeTruncate JournalMode = "TRUNCATE"
	JournalModePersist  JournalMode = "PERSIST"
	JournalModeMemory   JournalMode = "MEMORY"
	JournalModeWAL      JournalMode = "WAL"
	JournalModeOff      JournalMode = "OFF"
)

type SynchronousMode string

const (
	SynchronousOff    SynchronousMode = "OFF"
	SynchronousNormal SynchronousMode = "NORMAL"
	SynchronousFull   SynchronousMode = "FULL"
	SynchronousExtra  SynchronousMode = "EXTRA"
)

const (
	DefaultBusyTimeout     = 5 * time.Second
	DefaultCacheSizeKB     = 2_000
	DefaultMaxOpenConns    = 1
	DefaultMaxIdleConns    = 1
	ReadHeavyCacheSizeKB   = 20_000
	ReadHeavyMmapSizeBytes = int64(30_000_000_000)
	ReadHeavyMaxOpenConns  = 10
	ReadHeavyMaxIdleConns  = 5
)

type Config struct {
	Path            string
	Mode            Mode
	WALMode         bool
	JournalMode     JournalMode
	Synchronous     SynchronousMode
	BusyTimeout     time.Duration
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ForeignKeys     bool
	CacheSizeKB     int
	MmapSizeBytes   int64
}

type EffectiveConfig struct {
	Mode          Mode
	BusyTimeout   time.Duration
	ForeignKeys   bool
	JournalMode   JournalMode
	Synchronous   SynchronousMode
	CacheSizeKB   int
	MmapSizeBytes int64
	MaxOpenConns  int
	MaxIdleConns  int
}

func DefaultConfig(path string) Config {
	mode := modeForPath(path)
	return Config{Path: path, Mode: mode, WALMode: mode == ModeReadWrite, BusyTimeout: DefaultBusyTimeout, MaxOpenConns: 1, MaxIdleConns: 1, ConnMaxLifetime: time.Hour, ForeignKeys: true, CacheSizeKB: DefaultCacheSizeKB}
}

func ReadOnlyConfig() Config {
	cfg := DefaultConfig("")
	cfg.Mode = ModeReadOnly
	cfg.WALMode = false
	return cfg
}
func ImmutableConfig() Config {
	cfg := ReadOnlyConfig()
	cfg.Mode = ModeImmutable
	return cfg
}
func ReadHeavyConfig(path string) Config {
	cfg := DefaultConfig(path)
	cfg.CacheSizeKB = ReadHeavyCacheSizeKB
	cfg.MmapSizeBytes = ReadHeavyMmapSizeBytes
	cfg.MaxOpenConns = ReadHeavyMaxOpenConns
	cfg.MaxIdleConns = ReadHeavyMaxIdleConns
	return cfg
}

func ResolveConfig(cfg Config) (EffectiveConfig, error) {
	if cfg.Path == "" {
		return EffectiveConfig{}, fmt.Errorf("resolve sqlite config: empty path")
	}
	mode := cfg.Mode
	if mode == "" {
		mode = modeForPath(cfg.Path)
	}
	if mode != ModeReadWrite && mode != ModeReadOnly && mode != ModeImmutable && mode != ModeMemory {
		return EffectiveConfig{}, fmt.Errorf("resolve sqlite config: unsupported mode %q", mode)
	}
	if cfg.BusyTimeout < 0 || cfg.ConnMaxLifetime < 0 {
		return EffectiveConfig{}, fmt.Errorf("resolve sqlite config: durations must not be negative")
	}
	if cfg.MaxOpenConns < 0 || cfg.MaxIdleConns < 0 || cfg.CacheSizeKB < 0 || cfg.MmapSizeBytes < 0 {
		return EffectiveConfig{}, fmt.Errorf("resolve sqlite config: limits must not be negative")
	}
	if cfg.MaxOpenConns > 0 && cfg.MaxIdleConns > cfg.MaxOpenConns {
		return EffectiveConfig{}, fmt.Errorf("resolve sqlite config: MaxIdleConns exceeds MaxOpenConns")
	}
	if !validJournalMode(cfg.JournalMode) || !validSynchronousMode(cfg.Synchronous) {
		return EffectiveConfig{}, fmt.Errorf("resolve sqlite config: unsupported pragma value")
	}
	isMemory := cfg.Path == ":memory:" || strings.HasPrefix(cfg.Path, "file:") && strings.Contains(cfg.Path, "mode=memory")
	if (mode == ModeMemory) != isMemory {
		return EffectiveConfig{}, fmt.Errorf("resolve sqlite config: memory mode and path disagree")
	}
	journal := cfg.JournalMode
	if cfg.WALMode {
		if journal != "" && journal != JournalModeWAL {
			return EffectiveConfig{}, fmt.Errorf("resolve sqlite config: WALMode conflicts with journal mode %q", journal)
		}
		journal = JournalModeWAL
	}
	if (mode == ModeReadOnly || mode == ModeImmutable) && (journal != "" || cfg.Synchronous != "") {
		return EffectiveConfig{}, fmt.Errorf("resolve sqlite config: read-only modes cannot set write pragmas")
	}
	if mode == ModeMemory && journal == JournalModeWAL {
		return EffectiveConfig{}, fmt.Errorf("resolve sqlite config: memory mode does not support WAL")
	}
	syncMode := cfg.Synchronous
	if journal == JournalModeWAL && syncMode == "" {
		syncMode = SynchronousNormal
	}
	cache := cfg.CacheSizeKB
	if cache == 0 {
		cache = DefaultCacheSizeKB
	}
	open, idle := cfg.MaxOpenConns, cfg.MaxIdleConns
	if open == 0 {
		open = 1
	}
	if idle == 0 {
		idle = 1
	}
	if cfg.Path == ":memory:" && (open != 1 || idle != 1) {
		return EffectiveConfig{}, fmt.Errorf("resolve sqlite config: :memory: requires one open and idle connection")
	}
	return EffectiveConfig{mode, cfg.BusyTimeout, cfg.ForeignKeys, journal, syncMode, cache, cfg.MmapSizeBytes, open, idle}, nil
}

func Open(path string) (*sql.DB, error) { return OpenContext(context.Background(), path) }
func OpenContext(ctx context.Context, path string) (*sql.DB, error) {
	return OpenWithConfig(ctx, DefaultConfig(path))
}
func OpenReadOnly(ctx context.Context, path string) (*sql.DB, error) {
	cfg := ReadOnlyConfig()
	cfg.Path = path
	return OpenWithConfig(ctx, cfg)
}
func OpenImmutable(ctx context.Context, path string) (*sql.DB, error) {
	cfg := ImmutableConfig()
	cfg.Path = path
	return OpenWithConfig(ctx, cfg)
}

func OpenWithConfig(ctx context.Context, cfg Config) (*sql.DB, error) {
	effective, err := ResolveConfig(cfg)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", buildEffectiveDSN(cfg, effective))
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(effective.MaxOpenConns)
	db.SetMaxIdleConns(effective.MaxIdleConns)
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return db, nil
}

func buildDSN(cfg Config) string {
	effective, err := ResolveConfig(cfg)
	if err != nil {
		return ""
	}
	return buildEffectiveDSN(cfg, effective)
}
func buildEffectiveDSN(cfg Config, e EffectiveConfig) string {
	var p []string
	if e.Mode == ModeReadWrite {
		p = append(p, "_txlock=immediate")
	}
	if e.Mode == ModeReadOnly || e.Mode == ModeImmutable {
		p = append(p, "mode=ro")
	}
	if e.Mode == ModeImmutable {
		p = append(p, "immutable=1")
	}
	if e.BusyTimeout > 0 {
		p = append(p, fmt.Sprintf("_pragma=busy_timeout(%d)", e.BusyTimeout.Milliseconds()))
	}
	if e.ForeignKeys {
		p = append(p, "_pragma=foreign_keys(1)")
	}
	if e.JournalMode != "" {
		p = append(p, fmt.Sprintf("_pragma=journal_mode(%s)", e.JournalMode))
	}
	if e.Synchronous != "" {
		p = append(p, fmt.Sprintf("_pragma=synchronous(%s)", e.Synchronous))
	}
	p = append(p, fmt.Sprintf("_pragma=cache_size(-%d)", e.CacheSizeKB))
	if e.MmapSizeBytes > 0 {
		p = append(p, fmt.Sprintf("_pragma=mmap_size(%d)", e.MmapSizeBytes))
	}
	p = append(p, "_pragma=temp_store(MEMORY)")
	if e.Mode == ModeMemory && strings.HasPrefix(cfg.Path, "file:") {
		sep := "?"
		if strings.Contains(cfg.Path, "?") {
			sep = "&"
		}
		return cfg.Path + sep + strings.Join(p, "&")
	}
	return "file:" + escapeSQLitePath(cfg.Path) + "?" + strings.Join(p, "&")
}
func escapeSQLitePath(path string) string {
	if path == ":memory:" {
		return path
	}
	parts := strings.Split(filepath.ToSlash(path), "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.Join(parts, "/")
}
func modeForPath(path string) Mode {
	if path == ":memory:" || strings.HasPrefix(path, "file:") && strings.Contains(path, "mode=memory") {
		return ModeMemory
	}
	return ModeReadWrite
}
func validJournalMode(v JournalMode) bool {
	switch v {
	case "", JournalModeDelete, JournalModeTruncate, JournalModePersist, JournalModeMemory, JournalModeWAL, JournalModeOff:
		return true
	}
	return false
}
func validSynchronousMode(v SynchronousMode) bool {
	switch v {
	case "", SynchronousOff, SynchronousNormal, SynchronousFull, SynchronousExtra:
		return true
	}
	return false
}

func InMemory() (*sql.DB, error) { return InMemoryContext(context.Background()) }
func InMemoryContext(ctx context.Context) (*sql.DB, error) {
	cfg := DefaultConfig(":memory:")
	cfg.WALMode = false
	return OpenWithConfig(ctx, cfg)
}

func TransactionWithOptions(ctx context.Context, db *sql.DB, opts *sql.TxOptions, fn func(*sql.Tx) error) error {
	if db == nil {
		return fmt.Errorf("transaction: nil db")
	}
	if fn == nil {
		return fmt.Errorf("transaction: nil callback")
	}
	return sqltx.Run(func() (*sql.Tx, error) { return db.BeginTx(ctx, opts) }, func(tx *sql.Tx) error { return tx.Rollback() }, func(tx *sql.Tx) error { return tx.Commit() }, fn)
}
func Transaction(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	return TransactionWithOptions(ctx, db, nil, fn)
}
