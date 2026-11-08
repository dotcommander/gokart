package gokart

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestReadAllLimit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		in      string
		max     int64
		wantLen int
		wantErr bool
		tooBig  bool
	}{
		{"under", "abc", 10, 3, false, false},
		{"exact", "abcde", 5, 5, false, false},
		{"over", "abcdefgh", 5, 6, true, true},
		{"zero empty", "", 0, 0, false, false},
		{"negative", "x", -1, 0, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := ReadAllLimit(strings.NewReader(tt.in), tt.max)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if errors.Is(err, ErrTooLarge) != tt.tooBig {
				t.Fatalf("ErrTooLarge = %v, want %v", errors.Is(err, ErrTooLarge), tt.tooBig)
			}
			if len(data) != tt.wantLen {
				t.Fatalf("len = %d, want %d", len(data), tt.wantLen)
			}
		})
	}
	t.Run("reader error", func(t *testing.T) {
		t.Parallel()
		if _, err := ReadAllLimit(errReader{}, 5); err == nil || errors.Is(err, ErrTooLarge) {
			t.Fatalf("err = %v, want reader error", err)
		}
	})
}

func TestReadFileLimit(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	file := filepath.Join(dir, "f")
	if err := os.WriteFile(file, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Run("under", func(t *testing.T) {
		t.Parallel()
		data, err := ReadFileLimit(file, 10)
		if err != nil || string(data) != "hello" {
			t.Fatalf("got %q, %v", data, err)
		}
	})
	t.Run("over", func(t *testing.T) {
		t.Parallel()
		if _, err := ReadFileLimit(file, 2); !errors.Is(err, ErrTooLarge) {
			t.Fatalf("err = %v, want ErrTooLarge", err)
		}
	})
	t.Run("missing", func(t *testing.T) {
		t.Parallel()
		if _, err := ReadFileLimit(filepath.Join(dir, "nope"), 2); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("err = %v, want not exist", err)
		}
	})
	t.Run("directory", func(t *testing.T) {
		t.Parallel()
		if _, err := ReadFileLimit(dir, 10); err == nil {
			t.Fatal("expected error reading directory")
		}
	})
}
