//go:build unix

package fslock

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func lockFile(f *os.File, nonblocking bool) error {
	how := syscall.LOCK_EX
	if nonblocking {
		how |= syscall.LOCK_NB
	}
	for {
		err := syscall.Flock(int(f.Fd()), how)
		if errors.Is(err, syscall.EINTR) {
			continue
		}
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return ErrLocked
		}
		if err != nil {
			return fmt.Errorf("flock: %w", err)
		}
		return nil
	}
}

func unlockFile(f *os.File) error {
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_UN); err != nil {
		return fmt.Errorf("unlock: %w", err)
	}
	return nil
}
