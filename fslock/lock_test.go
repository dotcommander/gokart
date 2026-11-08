//go:build unix

package fslock

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestAcquireUnlock(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "l")
	l, err := Acquire(path, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Unlock(); err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	l2, err := Acquire(path, true)
	if err != nil {
		t.Fatalf("reacquire: %v", err)
	}
	t.Cleanup(func() { _ = l2.Close() })
}

func TestNonblockingContention(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "l")
	l, err := Acquire(path, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	if _, err := Acquire(path, true); !errors.Is(err, ErrLocked) {
		t.Fatalf("err = %v, want ErrLocked", err)
	}
}
