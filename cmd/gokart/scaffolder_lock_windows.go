//go:build windows

package main

// processIsRunning is the Windows stub for the POSIX signal-0 liveness check.
//
// Windows has no portable equivalent of kill(pid, 0); OpenProcess + GetExitCodeProcess
// would be the analogue but requires golang.org/x/sys/windows. To stay within
// stdlib and avoid taking on a dependency for a corner case, we return
// (true, nil) — conservative: never reclaim the lock automatically. The
// caller's timestamp-based stale-after check still applies, so an abandoned
// lockfile is eventually superseded once StaleAfter elapses.
func processIsRunning(_ int) (bool, error) {
	return true, nil
}
