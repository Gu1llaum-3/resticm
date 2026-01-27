package config

import (
	"fmt"
	"os"
	"runtime"
)

// PermissionError represents a file permission error
type PermissionError struct {
	Path     string
	Got      os.FileMode
	Expected string
	Message  string
}

func (e *PermissionError) Error() string {
	return fmt.Sprintf("%s: %s (got %04o, expected %s)\n\n"+
		"🔒 To fix this issue:\n"+
		"   sudo chown root:root %s\n"+
		"   sudo chmod 600 %s\n\n"+
		"💡 Permissions must be 600 (rw-------) or 400 (r--------)",
		e.Message, e.Path, e.Got, e.Expected, e.Path, e.Path)
}

// OwnerError represents a file ownership error
type OwnerError struct {
	Path         string
	FileOwnerUID uint32
	ExpectedUID  uint32
	Message      string
}

func (e *OwnerError) Error() string {
	return fmt.Sprintf("%s: %s (owned by UID %d, expected root or UID %d)\n\n"+
		"🔒 To fix this issue:\n"+
		"   sudo chown root:root %s\n"+
		"   sudo chmod 600 %s",
		e.Message, e.Path, e.FileOwnerUID, e.ExpectedUID, e.Path, e.Path)
}

// ValidateFilePermissions checks that a file has secure permissions
func ValidateFilePermissions(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Get file permissions
	perms := info.Mode().Perm()

	// Check permissions are 600 or 400
	if perms != 0600 && perms != 0400 {
		return &PermissionError{
			Path:     path,
			Got:      perms,
			Expected: "0600 or 0400",
			Message:  "configuration file has insecure permissions",
		}
	}

	// On Unix systems, also check owner
	if runtime.GOOS != "windows" {
		if err := checkFileOwner(info, path); err != nil {
			return err
		}
	}

	return nil
}

// EnsureSecureFile creates or updates a file with secure permissions
func EnsureSecureFile(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

// IsRoot returns true if the current process is running as root
func IsRoot() bool {
	return os.Geteuid() == 0
}

// RequireRoot returns an error if not running as root
func RequireRoot() error {
	if !IsRoot() {
		return fmt.Errorf("this command must be run as root or with sudo")
	}
	return nil
}
