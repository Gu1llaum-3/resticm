//go:build windows
// +build windows

package security

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/windows"
)

// Acquire attempts to acquire an exclusive lock (Windows implementation)
func (l *Lock) Acquire() error {
	// Ensure lock directory exists
	dir := filepath.Dir(l.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create lock directory: %w", err)
	}

	fmt.Printf("[DEBUG] Attempting to acquire lock: %s\n", l.path)

	// Open or create lock file with exclusive access
	path16, err := syscall.UTF16PtrFromString(l.path)
	if err != nil {
		return fmt.Errorf("failed to convert path: %w", err)
	}

	handle, err := windows.CreateFile(
		path16,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		0, // No sharing
		nil,
		windows.OPEN_ALWAYS,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return fmt.Errorf("another instance is already running (lock file: %s)", l.path)
	}

	file := os.NewFile(uintptr(handle), l.path)

	fmt.Printf("[DEBUG] Lock acquired successfully\n")

	// Write PID to lock file
	file.Truncate(0)
	file.Seek(0, 0)
	fmt.Fprintf(file, "%d\n", os.Getpid())

	l.file = file
	return nil
}

// Release releases the lock
func (l *Lock) Release() error {
	if l.file == nil {
		return nil
	}

	// Close and remove file
	l.file.Close()
	os.Remove(l.path)
	l.file = nil

	return nil
}

// IsLocked checks if the lock file is currently held by another process
func (l *Lock) IsLocked() bool {
	path16, err := syscall.UTF16PtrFromString(l.path)
	if err != nil {
		return false
	}

	handle, err := windows.CreateFile(
		path16,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		0,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return true
	}

	windows.CloseHandle(handle)
	return false
}

// getProcessStatus checks if a process with given PID is running (Windows)
func (l *Lock) getProcessStatus(pid int) string {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return "Process not found (stale lock)"
	}
	windows.CloseHandle(handle)
	return "Process is running"
}
