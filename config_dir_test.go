package gokart_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/dotcommander/gokart"
)

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
