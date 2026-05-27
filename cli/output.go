// Package cli — writer plumbing for package-level print helpers.
//
// Tests and embedders swap the destination via SetOutput / SetErrOutput; the
// helpers in cli.go (Success/Warning/Info/Dim/Bold/Error) and table.go
// (KeyValue/List/NumberedList) read these writers on every call. Reads are
// guarded by an RWMutex so a concurrent SetOutput cannot race with an
// in-flight print. See cli/table.go SetWriter for the per-instance equivalent
// on Table.
package cli

import (
	"io"
	"os"
	"sync"
)

// Override writers are nil by default — Output/ErrOutput fall back to the live
// os.Stdout/os.Stderr at call time. This keeps `os.Stdout = pipe` test patterns
// working without requiring callers to SetOutput before every redirect.
var (
	writerMu  sync.RWMutex
	outWriter io.Writer // nil → os.Stdout (re-read each call)
	errWriter io.Writer // nil → os.Stderr (re-read each call)
)

// SetOutput redirects Success/Warning/Info/Dim/Bold and the package-level
// KeyValue/List/NumberedList helpers. Pass nil to clear the override and fall
// back to os.Stdout at call time.
func SetOutput(w io.Writer) {
	writerMu.Lock()
	defer writerMu.Unlock()
	outWriter = w
}

// SetErrOutput redirects Error/Fatal/FatalErr. Pass nil to clear the override
// and fall back to os.Stderr at call time.
func SetErrOutput(w io.Writer) {
	writerMu.Lock()
	defer writerMu.Unlock()
	errWriter = w
}

// Output returns the configured stdout-replacement writer, falling back to
// the current os.Stdout when no override is set.
func Output() io.Writer {
	writerMu.RLock()
	defer writerMu.RUnlock()
	if outWriter != nil {
		return outWriter
	}
	return os.Stdout
}

// ErrOutput returns the configured stderr-replacement writer, falling back to
// the current os.Stderr when no override is set.
func ErrOutput() io.Writer {
	writerMu.RLock()
	defer writerMu.RUnlock()
	if errWriter != nil {
		return errWriter
	}
	return os.Stderr
}
