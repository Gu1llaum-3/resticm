//go:build !windows
// +build !windows

package security

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// Acquire attempts to acquire an exclusive lock
func (l *Lock) Acquire() error {
	// Ensure lock directory exists
	dir := filepath.Dir(l.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create lock directory: %w", err)
	}

	// Open or create lock file with permissions that allow any user to read/write
	// This prevents issues when switching between users (e.g., root vs regular user)
	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("failed to open lock file: %w", err)
	}

	// Try to acquire exclusive lock (non-blocking)
	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		_ = file.Close()
		if err == syscall.EWOULDBLOCK {
			return fmt.Errorf("another instance is already running (lock file: %s)", l.path)
		}
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	// Write PID to lock file
	_ = file.Truncate(0)
	_, _ = file.Seek(0, 0)
	_, _ = fmt.Fprintf(file, "%d\n", os.Getpid())

	l.file = file
	return nil
}

// Release releases the lock
func (l *Lock) Release() error {
	if l.file == nil {
		return nil
	}

	// Release lock
	if err := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN); err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	// Close and remove file
	_ = l.file.Close()
	_ = os.Remove(l.path)
	l.file = nil

	return nil
}

// IsLocked checks if the lock file is currently held by another process
func (l *Lock) IsLocked() bool {
	// First check if file exists
	if _, err := os.Stat(l.path); os.IsNotExist(err) {
		return false
	}

	// Try to open for read/write
	file, err := os.OpenFile(l.path, os.O_RDWR, 0644)
	if err != nil {
		// File exists but we can't open it (permission issue) - consider it locked
		if os.IsPermission(err) {
			return true
		}
		// Try read-only to at least check if file exists with content
		if _, err := os.Stat(l.path); err == nil {
			return true // File exists, assume locked
		}
		return false
	}
	defer func() { _ = file.Close() }()

	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		return true
	}

	_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	return false
}

// getProcessStatus checks if a process with given PID is running
func (l *Lock) getProcessStatus(pid int) string {
	process, err := os.FindProcess(pid)
	if err == nil {
		if err := process.Signal(syscall.Signal(0)); err == nil {
			return "Process is running"
		}
	}
	return "Process not found (stale lock)"
}
