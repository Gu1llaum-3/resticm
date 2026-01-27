package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunHook(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a simple test hook script
	hookPath := filepath.Join(tmpDir, "test-hook.sh")
	hookContent := `#!/bin/bash
echo "Hook executed"
exit 0
`
	if err := os.WriteFile(hookPath, []byte(hookContent), 0755); err != nil {
		t.Fatalf("Failed to write hook script: %v", err)
	}

	runner := &Runner{
		DryRun: false,
	}

	output, err := runner.Run(hookPath, nil)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if output == "" {
		t.Error("Expected output from hook script")
	}
}

func TestRunHookWithEnv(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a hook script that echoes an env var
	hookPath := filepath.Join(tmpDir, "env-hook.sh")
	hookContent := `#!/bin/bash
echo "TEST_VAR=$TEST_VAR"
`
	if err := os.WriteFile(hookPath, []byte(hookContent), 0755); err != nil {
		t.Fatalf("Failed to write hook script: %v", err)
	}

	runner := &Runner{
		DryRun: false,
		Env:    []string{"TEST_VAR=hello"},
	}

	output, err := runner.Run(hookPath, nil)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if output != "TEST_VAR=hello\n" {
		t.Errorf("Output = %q, want %q", output, "TEST_VAR=hello\n")
	}
}

func TestRunHookDryRun(t *testing.T) {
	runner := &Runner{
		DryRun: true,
	}

	// Even non-existent hook should not error in dry-run mode
	output, err := runner.Run("/nonexistent/hook.sh", nil)
	if err != nil {
		t.Fatalf("Run() in dry-run mode error = %v", err)
	}

	if output != "" {
		t.Errorf("Expected empty output in dry-run mode, got %q", output)
	}
}

func TestRunHookNonExistent(t *testing.T) {
	runner := &Runner{
		DryRun: false,
	}

	// Non-existent hook should be silently skipped
	output, err := runner.Run("/nonexistent/hook.sh", nil)
	if err != nil {
		t.Fatalf("Run() error = %v, expected nil for non-existent hook", err)
	}

	if output != "" {
		t.Errorf("Expected empty output for non-existent hook, got %q", output)
	}
}

func TestRunHookFailure(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a hook script that fails
	hookPath := filepath.Join(tmpDir, "fail-hook.sh")
	hookContent := `#!/bin/bash
echo "About to fail"
exit 1
`
	if err := os.WriteFile(hookPath, []byte(hookContent), 0755); err != nil {
		t.Fatalf("Failed to write hook script: %v", err)
	}

	runner := &Runner{
		DryRun: false,
	}

	_, err := runner.Run(hookPath, nil)
	if err == nil {
		t.Error("Expected error from failing hook script")
	}
}

func TestRunHookEmptyPath(t *testing.T) {
	runner := &Runner{
		DryRun: false,
	}

	output, err := runner.Run("", nil)
	if err != nil {
		t.Fatalf("Run() error = %v, expected nil for empty path", err)
	}

	if output != "" {
		t.Errorf("Expected empty output for empty path, got %q", output)
	}
}

func TestRunPreBackup(t *testing.T) {
	tmpDir := t.TempDir()

	hookPath := filepath.Join(tmpDir, "pre-backup.sh")
	hookContent := `#!/bin/bash
echo "Pre-backup hook"
`
	if err := os.WriteFile(hookPath, []byte(hookContent), 0755); err != nil {
		t.Fatalf("Failed to write hook script: %v", err)
	}

	runner := &Runner{
		DryRun:    false,
		PreBackup: hookPath,
	}

	err := runner.RunPreBackup()
	if err != nil {
		t.Fatalf("RunPreBackup() error = %v", err)
	}
}

func TestRunPostBackup(t *testing.T) {
	tmpDir := t.TempDir()

	hookPath := filepath.Join(tmpDir, "post-backup.sh")
	hookContent := `#!/bin/bash
echo "Post-backup hook"
`
	if err := os.WriteFile(hookPath, []byte(hookContent), 0755); err != nil {
		t.Fatalf("Failed to write hook script: %v", err)
	}

	runner := &Runner{
		DryRun:     false,
		PostBackup: hookPath,
	}

	err := runner.RunPostBackup(true, nil)
	if err != nil {
		t.Fatalf("RunPostBackup() error = %v", err)
	}
}

func TestNewRunner(t *testing.T) {
	runner := NewRunner()
	if runner == nil {
		t.Error("NewRunner() returned nil")
	}
}

func TestRunHookNotExecutable(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a hook script without execute permission
	hookPath := filepath.Join(tmpDir, "not-executable.sh")
	hookContent := `#!/bin/bash
echo "This should not run"
`
	// Write with 0644 (no execute permission)
	if err := os.WriteFile(hookPath, []byte(hookContent), 0644); err != nil {
		t.Fatalf("Failed to write hook script: %v", err)
	}

	runner := &Runner{
		DryRun: false,
	}

	_, err := runner.Run(hookPath, nil)
	if err == nil {
		t.Error("Expected error for non-executable hook")
	}

	// Check that error message mentions chmod
	if err != nil && !strings.Contains(err.Error(), "not executable") {
		t.Errorf("Expected error to mention 'not executable', got: %v", err)
	}
}
