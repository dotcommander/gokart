package gokart_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/dotcommander/gokart"
)

func TestConfigDirRejectsUnsafeAppNames(t *testing.T) {
	t.Parallel()

	for _, appName := range []string{"", ".", "..", "../escape", filepath.Join(string(filepath.Separator), "absolute")} {
		t.Run(fmt.Sprintf("ConfigDir_%q", appName), func(t *testing.T) {
			t.Parallel()

			if _, err := gokart.ConfigDir(appName); err == nil {
				t.Fatalf("ConfigDir(%q) succeeded, want error", appName)
			}
		})
		t.Run(fmt.Sprintf("EnsureConfigDir_%q", appName), func(t *testing.T) {
			t.Parallel()

			if err := gokart.EnsureConfigDir(appName, []byte("unsafe\n")); err == nil {
				t.Fatalf("EnsureConfigDir(%q) succeeded, want error", appName)
			}
		})
	}
}

func TestConfigDirAllowsNestedLocalAppName(t *testing.T) {
	t.Parallel()

	rootName := fmt.Sprintf("gokart-config-nested-test-%d", os.Getpid())
	appName := filepath.Join(rootName, "child")
	dir, err := gokart.ConfigDir(appName)
	if err != nil {
		t.Fatalf("ConfigDir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(filepath.Dir(dir)); err != nil {
			t.Errorf("RemoveAll(%q): %v", filepath.Dir(dir), err)
		}
	})

	if err := gokart.EnsureConfigDir(appName, []byte("nested\n")); err != nil {
		t.Fatalf("EnsureConfigDir: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "nested\n" {
		t.Fatalf("config.yaml = %q, want nested content", got)
	}
}

func TestEnsureConfigDirPreservesExistingConfig(t *testing.T) {
	t.Parallel()
	appName := fmt.Sprintf("gokart-config-test-%d", os.Getpid())
	dir, err := gokart.ConfigDir(appName)
	if err != nil {
		t.Fatalf("ConfigDir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Errorf("RemoveAll(%q): %v", dir, err)
		}
	})

	if err := gokart.EnsureConfigDir(appName, []byte("first\n")); err != nil {
		t.Fatalf("EnsureConfigDir(first): %v", err)
	}
	if err := gokart.EnsureConfigDir(appName, []byte("second\n")); err != nil {
		t.Fatalf("EnsureConfigDir(second): %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "first\n" {
		t.Fatalf("config.yaml = %q, want first content", got)
	}
}

func TestEnsureConfigDirConcurrentCallersPreserveWinner(t *testing.T) {
	appName := fmt.Sprintf("gokart-config-race-test-%d", os.Getpid())
	dir, err := gokart.ConfigDir(appName)
	if err != nil {
		t.Fatalf("ConfigDir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Errorf("RemoveAll(%q): %v", dir, err)
		}
	})

	const callers = 16
	start := make(chan struct{})
	errCh := make(chan error, callers)
	var wg sync.WaitGroup
	for i := range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errCh <- gokart.EnsureConfigDir(appName, []byte(fmt.Sprintf("candidate-%d\n", i)))
		}()
	}
	close(start)
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Errorf("EnsureConfigDir: %v", err)
		}
	}
	got, err := os.ReadFile(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	valid := false
	for i := range callers {
		if string(got) == fmt.Sprintf("candidate-%d\n", i) {
			valid = true
			break
		}
	}
	if !valid {
		t.Fatalf("config.yaml contains no complete candidate: %q", got)
	}
}
