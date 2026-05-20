package logger

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew_JSONFormatDefault(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	log := New(Config{Output: &buf})
	if log == nil {
		t.Fatal("New returned nil logger")
	}

	log.Info("hello", "k", "v")

	line := strings.TrimSpace(buf.String())
	if line == "" {
		t.Fatal("expected log output, got empty buffer")
	}

	var rec map[string]any
	if err := json.Unmarshal([]byte(line), &rec); err != nil {
		t.Fatalf("expected JSON output, got %q: %v", line, err)
	}
	if rec["msg"] != "hello" {
		t.Errorf("msg = %v, want %q", rec["msg"], "hello")
	}
	if rec["k"] != "v" {
		t.Errorf("k = %v, want %q", rec["k"], "v")
	}
	if rec["level"] != "INFO" {
		t.Errorf("level = %v, want INFO", rec["level"])
	}
}

func TestNew_TextFormat(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	log := New(Config{Format: "text", Output: &buf})

	log.Info("hello", "k", "v")

	out := buf.String()
	if out == "" {
		t.Fatal("expected log output, got empty buffer")
	}
	// text handler emits key=value pairs, not JSON
	if strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Errorf("expected text format, got JSON-looking output: %q", out)
	}
	if !strings.Contains(out, "msg=hello") {
		t.Errorf("expected msg=hello in text output, got %q", out)
	}
	if !strings.Contains(out, "k=v") {
		t.Errorf("expected k=v in text output, got %q", out)
	}
}

func TestNew_TextFormatCaseInsensitive(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	log := New(Config{Format: "TEXT", Output: &buf})

	log.Info("hi")

	out := strings.TrimSpace(buf.String())
	if strings.HasPrefix(out, "{") {
		t.Errorf("uppercase TEXT should select text format, got JSON: %q", out)
	}
}

func TestNew_LevelFiltering(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		level     string
		wantDebug bool
		wantInfo  bool
		wantWarn  bool
		wantError bool
	}{
		{"debug", "debug", true, true, true, true},
		{"info", "info", false, true, true, true},
		{"warn", "warn", false, false, true, true},
		{"error", "error", false, false, false, true},
		{"default_empty", "", false, true, true, true},
		{"unknown_falls_back_to_info", "verbose", false, true, true, true},
		{"mixed_case", "DEBUG", true, true, true, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			log := New(Config{Level: tc.level, Output: &buf})

			log.Debug("d")
			log.Info("i")
			log.Warn("w")
			log.Error("e")

			out := buf.String()
			gotDebug := strings.Contains(out, `"msg":"d"`)
			gotInfo := strings.Contains(out, `"msg":"i"`)
			gotWarn := strings.Contains(out, `"msg":"w"`)
			gotError := strings.Contains(out, `"msg":"e"`)

			if gotDebug != tc.wantDebug {
				t.Errorf("debug emitted=%v, want %v (output=%q)", gotDebug, tc.wantDebug, out)
			}
			if gotInfo != tc.wantInfo {
				t.Errorf("info emitted=%v, want %v", gotInfo, tc.wantInfo)
			}
			if gotWarn != tc.wantWarn {
				t.Errorf("warn emitted=%v, want %v", gotWarn, tc.wantWarn)
			}
			if gotError != tc.wantError {
				t.Errorf("error emitted=%v, want %v", gotError, tc.wantError)
			}
		})
	}
}

func TestNew_NilOutputDoesNotPanic(t *testing.T) {
	t.Parallel()
	// Default Output is os.Stderr; we don't capture stderr, just verify
	// construction succeeds and logging is non-panicking.
	log := New(Config{})
	if log == nil {
		t.Fatal("New returned nil")
	}
	// Intentionally do not call log.Info here — we don't want to pollute
	// stderr during the test run. Construction success is the assertion.
}

func TestNewDefault_NonNil(t *testing.T) {
	t.Parallel()
	log := NewDefault()
	if log == nil {
		t.Fatal("NewDefault returned nil")
	}
}

func TestPath_TempDirWithLogSuffix(t *testing.T) {
	t.Parallel()
	got := Path("myapp")
	want := filepath.Join(os.TempDir(), "myapp.log")
	if got != want {
		t.Errorf("Path(%q) = %q, want %q", "myapp", got, want)
	}
}

func TestPath_PreservesAppName(t *testing.T) {
	t.Parallel()
	got := Path("weird-app_1")
	if !strings.HasSuffix(got, "weird-app_1.log") {
		t.Errorf("Path(%q) = %q, missing expected suffix", "weird-app_1", got)
	}
	if filepath.Dir(got) != filepath.Clean(os.TempDir()) {
		t.Errorf("Path dir = %q, want %q", filepath.Dir(got), filepath.Clean(os.TempDir()))
	}
}

func TestNewFile_WritesJSONToTempPath(t *testing.T) {
	t.Parallel()
	// Use a unique app name per test run to avoid cross-test contamination
	// of the shared /tmp/<app>.log file.
	appName := "gokart-logger-test-" + t.Name()
	path := Path(appName)

	// Clean any stale file from prior runs.
	_ = os.Remove(path)
	t.Cleanup(func() { _ = os.Remove(path) })

	log, cleanup, err := NewFile(appName)
	if err != nil {
		t.Fatalf("NewFile error: %v", err)
	}
	if log == nil {
		t.Fatal("NewFile returned nil logger")
	}
	if cleanup == nil {
		t.Fatal("NewFile returned nil cleanup")
	}

	log.Info("test-event", "key", "value")
	log.Debug("debug-event") // NewFile is fixed at LevelDebug, must appear

	cleanup()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log file %s: %v", path, err)
	}
	if len(data) == 0 {
		t.Fatal("log file is empty after writes")
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 log lines, got %d: %q", len(lines), string(data))
	}

	var first map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("first line not JSON: %v (line=%q)", err, lines[0])
	}
	if first["msg"] != "test-event" {
		t.Errorf("first msg = %v, want test-event", first["msg"])
	}
	if first["key"] != "value" {
		t.Errorf("first key = %v, want value", first["key"])
	}

	var second map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &second); err != nil {
		t.Fatalf("second line not JSON: %v (line=%q)", err, lines[1])
	}
	if second["msg"] != "debug-event" {
		t.Errorf("second msg = %v, want debug-event", second["msg"])
	}
	if second["level"] != "DEBUG" {
		t.Errorf("second level = %v, want DEBUG (NewFile must accept debug)", second["level"])
	}
}

func TestNewFile_CleanupIsIdempotentSafe(t *testing.T) {
	t.Parallel()
	appName := "gokart-logger-cleanup-" + t.Name()
	path := Path(appName)
	_ = os.Remove(path)
	t.Cleanup(func() { _ = os.Remove(path) })

	_, cleanup, err := NewFile(appName)
	if err != nil {
		t.Fatalf("NewFile error: %v", err)
	}

	// First cleanup closes the file; second close returns an error from the
	// OS but the cleanup func itself must not panic.
	cleanup()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("second cleanup panicked: %v", r)
		}
	}()
	cleanup()
}

func TestNewFile_AppendsToExistingFile(t *testing.T) {
	t.Parallel()
	appName := "gokart-logger-append-" + t.Name()
	path := Path(appName)
	_ = os.Remove(path)
	t.Cleanup(func() { _ = os.Remove(path) })

	// First session writes one event.
	log1, cleanup1, err := NewFile(appName)
	if err != nil {
		t.Fatalf("NewFile #1 error: %v", err)
	}
	log1.Info("first")
	cleanup1()

	// Second session must append, not truncate.
	log2, cleanup2, err := NewFile(appName)
	if err != nil {
		t.Fatalf("NewFile #2 error: %v", err)
	}
	log2.Info("second")
	cleanup2()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	out := string(data)
	if !strings.Contains(out, `"msg":"first"`) {
		t.Errorf("expected first event preserved (append mode), got %q", out)
	}
	if !strings.Contains(out, `"msg":"second"`) {
		t.Errorf("expected second event present, got %q", out)
	}
}
