//go:build windows

package utils

import (
	"syscall"
)

var (
	kernel32                    = syscall.NewLazyDLL("kernel32.dll")
	procSetThreadExecutionState = kernel32.NewProc("SetThreadExecutionState")
)

const (
	ES_DISPLAY_REQUIRED = 0x00000002
)

// WakeScreen wakes up the display by calling SetThreadExecutionState
func WakeScreen() error {
	ret, _, err := procSetThreadExecutionState.Call(
		uintptr(ES_DISPLAY_REQUIRED),
	)
	if ret == 0 {
		return err
	}
	return nil
}
