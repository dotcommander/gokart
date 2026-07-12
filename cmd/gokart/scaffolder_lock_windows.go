//go:build windows

package main

import (
	"errors"
	"syscall"
)

const (
	windowsStillActive           = 259
	windowsErrorInvalidParameter = syscall.Errno(87)
)

func processIsRunning(pid int) (bool, error) {
	handle, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		if errors.Is(err, windowsErrorInvalidParameter) {
			return false, nil
		}
		return false, err
	}
	defer syscall.CloseHandle(handle)

	var exitCode uint32
	if err := syscall.GetExitCodeProcess(handle, &exitCode); err != nil {
		return false, err
	}
	return exitCode == windowsStillActive, nil
}
