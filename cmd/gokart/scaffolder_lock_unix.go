//go:build !windows

package main

import (
	"errors"
	"syscall"
)

// processIsRunning reports whether the given pid is alive on this host.
//
// Uses the POSIX signal-0 trick: kill(pid, 0) is a no-op delivery that only
// reports whether the target is a valid signal recipient.
//   - nil      → process exists and we have permission to signal it → alive
//   - ESRCH    → no such process → dead
//   - EPERM    → process exists but we lack permission → still alive
//   - other    → propagate the error to the caller
func processIsRunning(pid int) (bool, error) {
	if pid <= 0 {
		return false, nil
	}

	err := syscall.Kill(pid, 0)
	if err == nil {
		return true, nil
	}

	if errors.Is(err, syscall.ESRCH) {
		return false, nil
	}

	if errors.Is(err, syscall.EPERM) {
		return true, nil
	}

	return false, err
}
