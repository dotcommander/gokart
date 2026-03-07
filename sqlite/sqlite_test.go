package sqlite

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestDefaultConfigMaxOpenConns(t *testing.T) {
	cfg := DefaultConfig("test.db")
	if cfg.MaxOpenConns != 4 {
		t.Errorf("expected MaxOpenConns=4, got %d", cfg.MaxOpenConns)
	}
}

func TestBuildDSN(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		wantSubs []string
		wantNot  []string
	}{
		{
			name: "file db with defaults",
			cfg: Config{
				Path:        "app.db",
				WALMode:     true,
				BusyTimeout: 5 * time.Second,
				ForeignKeys: true,
			},
			wantSubs: []string{
				"_txlock=immediate",
				"_pragma=busy_timeout(5000)",
				"_pragma=foreign_keys(1)",
				"_pragma=journal_mode(WAL)",
				"_pragma=synchronous(NORMAL)",
				"_pragma=cache_size(-2000)",
				"_pragma=temp_store(MEMORY)",
			},
		},
		{
			name: "no WAL no foreign keys",
			cfg: Config{
				Path:        "plain.db",
				WALMode:     false,
				ForeignKeys: false,
			},
			wantSubs: []string{
				"_txlock=immediate",
				"_pragma=cache_size(-2000)",
				"_pragma=temp_store(MEMORY)",
			},
			wantNot: []string{
				"_pragma=journal_mode(WAL)",
				"_pragma=foreign_keys(1)",
			},
		},
		{
			name: "memory db skips txlock",
			cfg: Config{
				Path:        ":memory:",
				WALMode:     false,
				ForeignKeys: true,
			},
			wantSubs: []string{
				"_pragma=foreign_keys(1)",
				"_pragma=cache_size(-2000)",
			},
			wantNot: []string{
				"_txlock=immediate",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsn := buildDSN(tt.cfg)
			for _, sub := range tt.wantSubs {
				if !strings.Contains(dsn, sub) {
					t.Errorf("DSN %q missing %q", dsn, sub)
				}
			}
			for _, sub := range tt.wantNot {
				if strings.Contains(dsn, sub) {
					t.Errorf("DSN %q should not contain %q", dsn, sub)
				}
			}
		})
	}
}

func TestPerConnectionPragmas(t *testing.T) {
	cfg := DefaultConfig(":memory:")
	cfg.WALMode = false
	cfg.ForeignKeys = true
	// Use 2 connections so the pool can issue separate physical connections.
	cfg.MaxOpenConns = 2
	cfg.MaxIdleConns = 2

	db, err := OpenWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	conn1, err := db.Conn(ctx)
	if err != nil {
		t.Fatalf("conn1: %v", err)
	}
	defer conn1.Close()

	conn2, err := db.Conn(ctx)
	if err != nil {
		t.Fatalf("conn2: %v", err)
	}
	defer conn2.Close()

	var fk1 int
	if err := conn1.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&fk1); err != nil {
		t.Fatalf("conn1 pragma scan: %v", err)
	}
	if fk1 != 1 {
		t.Errorf("conn1: expected foreign_keys=1, got %d", fk1)
	}

	var fk2 int
	if err := conn2.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&fk2); err != nil {
		t.Fatalf("conn2 pragma scan: %v", err)
	}
	if fk2 != 1 {
		t.Errorf("conn2: expected foreign_keys=1, got %d", fk2)
	}
}
