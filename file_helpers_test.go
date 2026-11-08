package gokart

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestAtomicWriteFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "a", "b", "f.txt")
	if err := AtomicWriteFile(path, []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := AtomicWriteFile(path, []byte("two"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "two" {
		t.Fatalf("got %q, %v", got, err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("perm = %v", info.Mode().Perm())
		}
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 1 {
		t.Fatalf("leftover temp files: %v, %v", entries, err)
	}
}
