package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestRunNow(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	if err := run(t.Context(), []string{"now"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"WGBH", "Nature: Wild Coast", "WGBX", "The Great British Bake Off", "WCVB", "Chronicle"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("output omits %q:\n%s", want, stdout.String())
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunFiltersByChannel(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	if err := run(t.Context(), []string{"now", "--channel", "WGBH"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	const want = "CHANNEL  STARTS   PROGRAM\nWGBH     8:00 PM  Nature: Wild Coast\n"
	if got := stdout.String(); got != want {
		t.Fatalf("output mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunWithoutArgumentsPrintsUsage(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	if err := run(t.Context(), nil, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Usage: tvguide <command>") {
		t.Fatalf("usage missing from stdout:\n%s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunRejectsInvalidCommand(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	if err := run(t.Context(), []string{"later"}, &stdout, &stderr); err == nil {
		t.Fatal("expected invalid command error")
	}
}

func TestRunReportsUnknownChannel(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	err := run(t.Context(), []string{"now", "--channel", "missing"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), `no programs found for channel "missing"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWriteProgramsPropagatesWriterFailure(t *testing.T) {
	t.Parallel()

	want := errors.New("write failed")
	if err := writePrograms(errorWriter{err: want}, schedule); !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
}

type errorWriter struct {
	err error
}

func (w errorWriter) Write([]byte) (int, error) {
	return 0, w.err
}
