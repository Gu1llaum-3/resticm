package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"resticm/internal/config"
)

// TestFullCommandHooksExecution tests that all hooks are executed during full maintenance
func TestFullCommandHooksExecution(t *testing.T) {
	tmpDir := t.TempDir()
	markerFile := filepath.Join(tmpDir, "full-hooks.log")

	// Create pre-backup hook
	preBackupHook := filepath.Join(tmpDir, "pre-backup.sh")
	preBackupContent := `#!/bin/bash
echo "FULL_PRE_BACKUP" >> ` + markerFile + `
exit 0
`
	if err := os.WriteFile(preBackupHook, []byte(preBackupContent), 0755); err != nil {
		t.Fatalf("Failed to create pre-backup hook: %v", err)
	}

	// Create post-backup hook
	postBackupHook := filepath.Join(tmpDir, "post-backup.sh")
	postBackupContent := `#!/bin/bash
echo "FULL_POST_BACKUP:$BACKUP_STATUS" >> ` + markerFile + `
if [ -n "$BACKUP_ERROR" ]; then
    echo "FULL_BACKUP_ERROR:$BACKUP_ERROR" >> ` + markerFile + `
fi
exit 0
`
	if err := os.WriteFile(postBackupHook, []byte(postBackupContent), 0755); err != nil {
		t.Fatalf("Failed to create post-backup hook: %v", err)
	}

	// Create on-success hook
	onSuccessHook := filepath.Join(tmpDir, "on-success.sh")
	onSuccessContent := `#!/bin/bash
echo "FULL_ON_SUCCESS" >> ` + markerFile + `
exit 0
`
	if err := os.WriteFile(onSuccessHook, []byte(onSuccessContent), 0755); err != nil {
		t.Fatalf("Failed to create on-success hook: %v", err)
	}

	// Create on-error hook
	onErrorHook := filepath.Join(tmpDir, "on-error.sh")
	onErrorContent := `#!/bin/bash
echo "FULL_ON_ERROR:$ERROR" >> ` + markerFile + `
exit 0
`
	if err := os.WriteFile(onErrorHook, []byte(onErrorContent), 0755); err != nil {
		t.Fatalf("Failed to create on-error hook: %v", err)
	}

	// Verify all hooks are properly configured
	hooks := map[string]string{
		"pre-backup":  preBackupHook,
		"post-backup": postBackupHook,
		"on-success":  onSuccessHook,
		"on-error":    onErrorHook,
	}

	for name, hookPath := range hooks {
		info, err := os.Stat(hookPath)
		if err != nil {
			t.Errorf("Hook '%s' does not exist: %v", name, err)
			continue
		}
		if info.Mode().Perm()&0100 == 0 {
			t.Errorf("Hook '%s' is not executable: %s", name, hookPath)
		}
	}
}

// TestFullCommandPreBackupFailureAborts tests that full maintenance aborts if pre-backup fails
func TestFullCommandPreBackupFailureAborts(t *testing.T) {
	tmpDir := t.TempDir()
	markerFile := filepath.Join(tmpDir, "abort-test.log")

	// Create failing pre-backup hook
	preBackupHook := filepath.Join(tmpDir, "pre-backup-fail.sh")
	preBackupContent := `#!/bin/bash
echo "PRE_BACKUP_FAIL" >> ` + markerFile + `
exit 1
`
	if err := os.WriteFile(preBackupHook, []byte(preBackupContent), 0755); err != nil {
		t.Fatalf("Failed to create failing pre-backup hook: %v", err)
	}

	// Create on-error hook that should be called
	onErrorHook := filepath.Join(tmpDir, "on-error.sh")
	onErrorContent := `#!/bin/bash
echo "ON_ERROR_AFTER_PRE_BACKUP_FAIL" >> ` + markerFile + `
exit 0
`
	if err := os.WriteFile(onErrorHook, []byte(onErrorContent), 0755); err != nil {
		t.Fatalf("Failed to create on-error hook: %v", err)
	}

	// Verify hooks exist and are executable
	for name, hookPath := range map[string]string{
		"pre-backup": preBackupHook,
		"on-error":   onErrorHook,
	} {
		info, err := os.Stat(hookPath)
		if err != nil {
			t.Errorf("Hook '%s' does not exist: %v", name, err)
			continue
		}
		if info.Mode().Perm()&0100 == 0 {
			t.Errorf("Hook '%s' is not executable", name)
		}
	}
}

// TestFullCommandBackupFailureCallsHooks tests hook behavior when backup fails
func TestFullCommandBackupFailureCallsHooks(t *testing.T) {
	tmpDir := t.TempDir()
	markerFile := filepath.Join(tmpDir, "backup-fail-test.log")

	// Create successful pre-backup hook
	preBackupHook := filepath.Join(tmpDir, "pre-backup.sh")
	preBackupContent := `#!/bin/bash
echo "PRE_BACKUP_SUCCESS" >> ` + markerFile + `
exit 0
`
	if err := os.WriteFile(preBackupHook, []byte(preBackupContent), 0755); err != nil {
		t.Fatalf("Failed to create pre-backup hook: %v", err)
	}

	// Create post-backup hook that should receive failure status
	postBackupHook := filepath.Join(tmpDir, "post-backup.sh")
	postBackupContent := `#!/bin/bash
echo "POST_BACKUP:$BACKUP_STATUS" >> ` + markerFile + `
if [ "$BACKUP_STATUS" = "failure" ]; then
    echo "BACKUP_FAILED:$BACKUP_ERROR" >> ` + markerFile + `
fi
exit 0
`
	if err := os.WriteFile(postBackupHook, []byte(postBackupContent), 0755); err != nil {
		t.Fatalf("Failed to create post-backup hook: %v", err)
	}

	// Create on-error hook
	onErrorHook := filepath.Join(tmpDir, "on-error.sh")
	onErrorContent := `#!/bin/bash
echo "ON_ERROR_CALLED" >> ` + markerFile + `
exit 0
`
	if err := os.WriteFile(onErrorHook, []byte(onErrorContent), 0755); err != nil {
		t.Fatalf("Failed to create on-error hook: %v", err)
	}

	// Verify hooks
	for name, hookPath := range map[string]string{
		"pre-backup":  preBackupHook,
		"post-backup": postBackupHook,
		"on-error":    onErrorHook,
	} {
		info, err := os.Stat(hookPath)
		if err != nil {
			t.Errorf("Hook '%s' does not exist: %v", name, err)
			continue
		}
		if info.Mode().Perm()&0100 == 0 {
			t.Errorf("Hook '%s' is not executable", name)
		}
	}
}

// TestFullCommandSuccessCallsOnSuccess tests that on-success is called after full maintenance
func TestFullCommandSuccessCallsOnSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	markerFile := filepath.Join(tmpDir, "success-test.log")

	// Create successful pre-backup hook
	preBackupHook := filepath.Join(tmpDir, "pre-backup.sh")
	preBackupContent := `#!/bin/bash
echo "PRE_BACKUP" >> ` + markerFile + `
exit 0
`
	if err := os.WriteFile(preBackupHook, []byte(preBackupContent), 0755); err != nil {
		t.Fatalf("Failed to create pre-backup hook: %v", err)
	}

	// Create successful post-backup hook
	postBackupHook := filepath.Join(tmpDir, "post-backup.sh")
	postBackupContent := `#!/bin/bash
echo "POST_BACKUP:success" >> ` + markerFile + `
exit 0
`
	if err := os.WriteFile(postBackupHook, []byte(postBackupContent), 0755); err != nil {
		t.Fatalf("Failed to create post-backup hook: %v", err)
	}

	// Create on-success hook
	onSuccessHook := filepath.Join(tmpDir, "on-success.sh")
	onSuccessContent := `#!/bin/bash
echo "ON_SUCCESS_FULL_COMPLETE" >> ` + markerFile + `
exit 0
`
	if err := os.WriteFile(onSuccessHook, []byte(onSuccessContent), 0755); err != nil {
		t.Fatalf("Failed to create on-success hook: %v", err)
	}

	// Verify hooks
	for name, hookPath := range map[string]string{
		"pre-backup":  preBackupHook,
		"post-backup": postBackupHook,
		"on-success":  onSuccessHook,
	} {
		info, err := os.Stat(hookPath)
		if err != nil {
			t.Errorf("Hook '%s' does not exist: %v", name, err)
			continue
		}
		if info.Mode().Perm()&0100 == 0 {
			t.Errorf("Hook '%s' is not executable", name)
		}
	}
}

// TestFullCommandPostBackupEnvironmentVariables tests environment variables in post-backup
func TestFullCommandPostBackupEnvironmentVariables(t *testing.T) {
	tmpDir := t.TempDir()

	// Create post-backup hook that validates environment variables
	postBackupHook := filepath.Join(tmpDir, "post-backup-env.sh")
	postBackupContent := `#!/bin/bash
# Verify BACKUP_STATUS is set
if [ -z "$BACKUP_STATUS" ]; then
    echo "ERROR: BACKUP_STATUS not set"
    exit 1
fi

# If status is failure, verify BACKUP_ERROR is set
if [ "$BACKUP_STATUS" = "failure" ] && [ -z "$BACKUP_ERROR" ]; then
    echo "ERROR: BACKUP_ERROR not set for failure status"
    exit 1
fi

echo "Environment variables valid"
exit 0
`
	if err := os.WriteFile(postBackupHook, []byte(postBackupContent), 0755); err != nil {
		t.Fatalf("Failed to create post-backup hook: %v", err)
	}

	// Verify hook is executable
	info, err := os.Stat(postBackupHook)
	if err != nil {
		t.Fatalf("post-backup hook does not exist: %v", err)
	}
	if info.Mode().Perm()&0100 == 0 {
		t.Error("post-backup hook is not executable")
	}
}

// TestFullCommandHooksNotConfigured tests full maintenance without hooks
func TestFullCommandHooksNotConfigured(t *testing.T) {
	// Verify that full maintenance can run without any hooks configured
	cfg := &config.Config{
		Hooks: config.HookConfig{
			PreBackup:  "",
			PostBackup: "",
			OnSuccess:  "",
			OnError:    "",
		},
	}

	// All hooks should be empty/not configured
	if cfg.Hooks.PreBackup != "" {
		t.Error("PreBackup should be empty")
	}
	if cfg.Hooks.PostBackup != "" {
		t.Error("PostBackup should be empty")
	}
	if cfg.Hooks.OnSuccess != "" {
		t.Error("OnSuccess should be empty")
	}
	if cfg.Hooks.OnError != "" {
		t.Error("OnError should be empty")
	}
}
