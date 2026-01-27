package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLockAcquireRelease(t *testing.T) {
	tmpDir := t.TempDir()
	lockFile := filepath.Join(tmpDir, "test.lock")

	lock := NewLock(lockFile)

	// Acquire lock
	if err := lock.Acquire(); err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}

	// Verify lock file exists
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		t.Error("Lock file was not created")
	}

	// Release lock
	if err := lock.Release(); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
}

func TestLockDoubleAcquire(t *testing.T) {
	tmpDir := t.TempDir()
	lockFile := filepath.Join(tmpDir, "test.lock")

	lock1 := NewLock(lockFile)
	lock2 := NewLock(lockFile)

	// First lock should succeed
	if err := lock1.Acquire(); err != nil {
		t.Fatalf("First Acquire() error = %v", err)
	}
	defer func() { _ = lock1.Release() }()

	// Second lock should fail immediately (non-blocking)
	err := lock2.Acquire()
	if err == nil {
		t.Error("Expected error for second Acquire(), got nil")
		_ = lock2.Release()
	}
}

func TestForceUnlock(t *testing.T) {
	tmpDir := t.TempDir()
	lockFile := filepath.Join(tmpDir, "test.lock")

	lock := NewLock(lockFile)

	// Acquire lock
	if err := lock.Acquire(); err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}

	// Force unlock
	if err := lock.ForceUnlock(); err != nil {
		t.Fatalf("ForceUnlock() error = %v", err)
	}

	// File should be deleted
	if _, err := os.Stat(lockFile); !os.IsNotExist(err) {
		t.Error("Lock file should be deleted after ForceUnlock")
	}
}

func TestForceUnlockNonExistent(t *testing.T) {
	lock := NewLock("/nonexistent/lock/file")

	// Should not error on non-existent file
	err := lock.ForceUnlock()
	if err != nil {
		t.Errorf("ForceUnlock() on non-existent file error = %v", err)
	}
}

func TestNewLockDefault(t *testing.T) {
	lock := NewLock("")

	// Check that a default path was set (either root or user path)
	if lock.path == "" {
		t.Error("NewLock(\"\").path should not be empty")
	}

	// Verify the path is one of the valid defaults based on current user
	isRoot := os.Getuid() == 0
	if isRoot && lock.path != "/var/lock/resticm.lock" {
		t.Errorf("Running as root, expected /var/lock/resticm.lock, got %q", lock.path)
	}
	if !isRoot && !strings.Contains(lock.path, ".local/share/resticm") {
		t.Errorf("Running as non-root, expected path containing .local/share/resticm, got %q", lock.path)
	}
}

func TestLockGetPID(t *testing.T) {
	tmpDir := t.TempDir()
	lockFile := filepath.Join(tmpDir, "test.lock")

	lock := NewLock(lockFile)

	if err := lock.Acquire(); err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	defer func() { _ = lock.Release() }()

	pid, err := lock.GetPID()
	if err != nil {
		t.Fatalf("GetPID() error = %v", err)
	}

	if pid != os.Getpid() {
		t.Errorf("GetPID() = %d, want %d", pid, os.Getpid())
	}
}

func TestLockIsLocked(t *testing.T) {
	tmpDir := t.TempDir()
	lockFile := filepath.Join(tmpDir, "test.lock")

	lock := NewLock(lockFile)

	// Not locked initially
	if lock.IsLocked() {
		t.Error("IsLocked() = true before Acquire(), want false")
	}

	if err := lock.Acquire(); err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}

	// Should be locked now (from another lock's perspective)
	lock2 := NewLock(lockFile)
	if !lock2.IsLocked() {
		t.Error("IsLocked() = false after Acquire(), want true")
	}

	_ = lock.Release()
}
