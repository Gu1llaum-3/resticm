// Package security provides security-related functionality
package security

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// DefaultLockFile is the default lock file path for root
	DefaultLockFile = "/var/lock/resticm.lock"
)

// Lock represents a file lock
type Lock struct {
	path string
	file *os.File
}

// NewLock creates a new lock with the given path
// If path is empty, it chooses an appropriate default based on user privileges
func NewLock(path string) *Lock {
	if path == "" {
		path = getDefaultLockPath()
	}
	return &Lock{path: path}
}

// getDefaultLockPath returns the appropriate lock file path
func getDefaultLockPath() string {
	// If running as root, use system lock directory
	if os.Geteuid() == 0 {
		return DefaultLockFile
	}

	// For non-root users, use a lock file in their home directory
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share", "resticm", "resticm.lock")
	}

	// Fallback to /tmp if home directory is not available
	return "/tmp/resticm.lock"
}

// ForceUnlock removes a stale lock file
func (l *Lock) ForceUnlock() error {
	if err := os.Remove(l.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove lock file: %w", err)
	}
	return nil
}

// GetPID returns the PID from the lock file
func (l *Lock) GetPID() (int, error) {
	data, err := os.ReadFile(l.path)
	if err != nil {
		return 0, err
	}

	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return 0, err
	}

	return pid, nil
}

// PrintLockInfo prints information about the current lock
func (l *Lock) PrintLockInfo() {
	if !l.IsLocked() {
		fmt.Println("No active lock")
		return
	}

	pid, err := l.GetPID()
	if err != nil {
		fmt.Printf("Lock file exists: %s (could not read PID)\n", l.path)
		return
	}

	fmt.Printf("⚠️  Lock file: %s\n", l.path)
	fmt.Printf("   PID: %d\n", pid)
	fmt.Printf("   Status: %s\n", l.getProcessStatus(pid))
}
