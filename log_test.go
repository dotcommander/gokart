package gokart_test

import (
	"os"
	"strings"
	"testing"

	"github.com/dotcommander/gokart"
)

func TestNewFileLogger(t *testing.T) {
	t.Parallel()

	appName := "gokart-test-logger"

	// Clean up after test
	defer os.Remove(gokart.LogPath(appName))

	logger, cleanup, err := gokart.NewFileLogger(appName)
	if err != nil {
		t.Fatalf("NewFileLogger failed: %v", err)
	}
	defer cleanup()

	if logger == nil {
		t.Fatal("expected logger, got nil")
	}

	// Write a log entry
	logger.Info("test message", "key", "value")

	// Verify file exists
	path := gokart.LogPath(appName)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("log file should exist at %s", path)
	}
}

func TestLogPath(t *testing.T) {
	t.Parallel()

	path := gokart.LogPath("myapp")

	// Should be in temp directory
	if !strings.HasPrefix(path, os.TempDir()) {
		t.Errorf("path should be in temp dir: %s", path)
	}

	// Should end with .log
	if !strings.HasSuffix(path, ".log") {
		t.Errorf("path should end with .log: %s", path)
	}

	// Should contain app name
	if !strings.Contains(path, "myapp") {
		t.Errorf("path should contain app name: %s", path)
	}
}

func TestNewFileLogger_Append(t *testing.T) {
	t.Parallel()

	appName := "gokart-test-append"
	path := gokart.LogPath(appName)

	// Clean up
	defer os.Remove(path)

	// First logger
	logger1, cleanup1, err := gokart.NewFileLogger(appName)
	if err != nil {
		t.Fatalf("first NewFileLogger failed: %v", err)
	}
	logger1.Info("first message")
	cleanup1()

	// Second logger (should append)
	logger2, cleanup2, err := gokart.NewFileLogger(appName)
	if err != nil {
		t.Fatalf("second NewFileLogger failed: %v", err)
	}
	logger2.Info("second message")
	cleanup2()

	// Verify both messages are in file
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}

	if !strings.Contains(string(content), "first message") {
		t.Error("log should contain first message")
	}
	if !strings.Contains(string(content), "second message") {
		t.Error("log should contain second message")
	}
}
