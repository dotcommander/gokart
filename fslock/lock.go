// Package fslock provides advisory whole-file locks for cross-process
// coordination. Unix only; other platforms return ErrUnsupported.
package fslock

import (
	"errors"
	"fmt"
	"os"
)

var (
	// ErrLocked is returned by a nonblocking Acquire when another holder exists.
	ErrLocked = errors.New("fslock: already locked")
	// ErrUnsupported is returned on platforms without lock support.
	ErrUnsupported = errors.New("fslock: not supported on this platform")
)

// Lock is a held advisory lock. Release it with Unlock or Close.
type Lock struct {
	file *os.File
}

// Acquire locks path exclusively, creating it (0600) if needed. With
// nonblocking set it returns ErrLocked instead of waiting.
func Acquire(path string, nonblocking bool) (*Lock, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}
	if err := lockFile(f, nonblocking); err != nil {
		_ = f.Close()
		return nil, err
	}
	return &Lock{file: f}, nil
}

// Unlock releases the lock and closes the file. It is idempotent.
func (l *Lock) Unlock() error {
	if l == nil || l.file == nil {
		return nil
	}
	f := l.file
	l.file = nil
	return errors.Join(unlockFile(f), f.Close())
}

// Close is an alias for Unlock.
func (l *Lock) Close() error { return l.Unlock() }
