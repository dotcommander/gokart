package postgres

import (
	"context"
	"strings"
	"testing"
	"time"
)

// unreachableURL points at a closed port: connections refuse immediately,
// so these failure-path tests never need a live database or a new dependency.
const unreachableURL = "postgres://u:p@127.0.0.1:1/db?sslmode=disable"

func TestNewPoolRejectsInvalidDSN(t *testing.T) {
	t.Parallel()
	cfg := Config{URL: "://invalid"}
	_, err := cfg.NewPool(context.Background())
	if err == nil {
		t.Fatal("NewPool with invalid DSN error = nil")
	}
}

func TestNewPoolWithoutPingSucceedsUnreachable(t *testing.T) {
	t.Parallel()
	cfg := Config{URL: unreachableURL}
	pool, err := cfg.NewPool(context.Background())
	if err != nil {
		t.Fatalf("NewPool (no ping) err = %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(context.Background()); err == nil {
		t.Fatal("ping of unreachable host must fail")
	}
}

func TestOpenWithConfigInvalidDSNFails(t *testing.T) {
	t.Parallel()
	_, err := OpenWithConfig(context.Background(), Config{URL: "://invalid"})
	if err == nil {
		t.Fatal("OpenWithConfig with invalid DSN error = nil")
	}
}

func TestOpenWithConfigUnreachablePingFails(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	_, err := OpenWithConfig(ctx, Config{URL: unreachableURL})
	if err == nil || !strings.Contains(err.Error(), "ping") {
		t.Fatalf("OpenWithConfig unreachable err = %v, want ping error", err)
	}
}

func TestOpenUnreachablePingFails(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	_, err := Open(ctx, unreachableURL)
	if err == nil || !strings.Contains(err.Error(), "ping") {
		t.Fatalf("Open unreachable err = %v, want ping error", err)
	}
}

func TestFromEnvValidButUnreachablePingFails(t *testing.T) {
	t.Setenv("DATABASE_URL", unreachableURL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	_, err := FromEnv(ctx)
	if err == nil || !strings.Contains(err.Error(), "ping") {
		t.Fatalf("FromEnv unreachable err = %v, want ping error", err)
	}
}
