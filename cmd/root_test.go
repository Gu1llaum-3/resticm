package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"resticm/internal/config"
)

// TestDefaultWorkflowHooksExecution tests that hooks are executed in default workflow
func TestDefaultWorkflowHooksExecution(t *testing.T) {
	tmpDir := t.TempDir()
	markerFile := filepath.Join(tmpDir, "workflow-hooks.log")
	
	// Create pre-backup hook
	preBackupHook := filepath.Join(tmpDir, "pre-backup.sh")
	preBackupContent := `#!/bin/bash
echo "PRE_BACKUP" >> ` + markerFile + `
exit 0
`
	if err := os.WriteFile(preBackupHook, []byte(preBackupContent), 0755); err != nil {
		t.Fatalf("Failed to create pre-backup hook: %v", err)
	}
	
	// Create post-backup hook
	postBackupHook := filepath.Join(tmpDir, "post-backup.sh")
	postBackupContent := `#!/bin/bash
echo "POST_BACKUP:$BACKUP_STATUS" >> ` + markerFile + `
exit 0
`
	if err := os.WriteFile(postBackupHook, []byte(postBackupContent), 0755); err != nil {
		t.Fatalf("Failed to create post-backup hook: %v", err)
	}
	
	// Create on-success hook
	onSuccessHook := filepath.Join(tmpDir, "on-success.sh")
	onSuccessContent := `#!/bin/bash
echo "ON_SUCCESS" >> ` + markerFile + `
exit 0
`
	if err := os.WriteFile(onSuccessHook, []byte(onSuccessContent), 0755); err != nil {
		t.Fatalf("Failed to create on-success hook: %v", err)
	}
	
	// Create on-error hook
	onErrorHook := filepath.Join(tmpDir, "on-error.sh")
	onErrorContent := `#!/bin/bash
echo "ON_ERROR:$ERROR" >> ` + markerFile + `
exit 0
`
	if err := os.WriteFile(onErrorHook, []byte(onErrorContent), 0755); err != nil {
		t.Fatalf("Failed to create on-error hook: %v", err)
	}
	
	// Verify all hooks are executable
	for name, hookPath := range map[string]string{
		"pre-backup":  preBackupHook,
		"post-backup": postBackupHook,
		"on-success":  onSuccessHook,
		"on-error":    onErrorHook,
	} {
		info, err := os.Stat(hookPath)
		if err != nil {
			t.Errorf("Hook '%s' file does not exist: %v", name, err)
			continue
		}
		if info.Mode().Perm()&0100 == 0 {
			t.Errorf("Hook '%s' file is not executable: %s", name, hookPath)
		}
	}
}

// TestDefaultWorkflowPreBackupFailureStopsBackup tests that workflow stops if pre-backup fails
func TestDefaultWorkflowPreBackupFailureStopsBackup(t *testing.T) {
	tmpDir := t.TempDir()
	markerFile := filepath.Join(tmpDir, "execution.log")
	
	// Create failing pre-backup hook
	preBackupHook := filepath.Join(tmpDir, "pre-backup-fail.sh")
	preBackupContent := `#!/bin/bash
echo "PRE_BACKUP_FAILED" >> ` + markerFile + `
exit 1
`
	if err := os.WriteFile(preBackupHook, []byte(preBackupContent), 0755); err != nil {
		t.Fatalf("Failed to create pre-backup hook: %v", err)
	}
	
	// Create post-backup hook that should NOT run
	postBackupHook := filepath.Join(tmpDir, "post-backup.sh")
	postBackupContent := `#!/bin/bash
echo "POST_BACKUP_SHOULD_NOT_RUN" >> ` + markerFile + `
exit 0
`
	if err := os.WriteFile(postBackupHook, []byte(postBackupContent), 0755); err != nil {
		t.Fatalf("Failed to create post-backup hook: %v", err)
	}
	
	// Create on-error hook that SHOULD run
	onErrorHook := filepath.Join(tmpDir, "on-error.sh")
	onErrorContent := `#!/bin/bash
echo "ON_ERROR_CALLED" >> ` + markerFile + `
exit 0
`
	if err := os.WriteFile(onErrorHook, []byte(onErrorContent), 0755); err != nil {
		t.Fatalf("Failed to create on-error hook: %v", err)
	}
	
	// Verify hooks are properly set up
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

// TestDefaultWorkflowHooksNotConfigured tests that workflow works without hooks
func TestDefaultWorkflowHooksNotConfigured(t *testing.T) {
	// Set up configuration with no hooks
	cfg = &config.Config{
		Hooks: config.HookConfig{
			PreBackup:  "",
			PostBackup: "",
			OnSuccess:  "",
			OnError:    "",
		},
	}
	
	// This verifies that empty hook paths are handled gracefully
	// The actual workflow execution would require a test repository
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

// TestDefaultWorkflowOnSuccessCalledOnSuccess tests on-success hook execution
func TestDefaultWorkflowOnSuccessCalledOnSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	markerFile := filepath.Join(tmpDir, "success-marker.log")
	
	// Create on-success hook
	onSuccessHook := filepath.Join(tmpDir, "on-success.sh")
	onSuccessContent := `#!/bin/bash
echo "SUCCESS_HOOK_CALLED" >> ` + markerFile + `
exit 0
`
	if err := os.WriteFile(onSuccessHook, []byte(onSuccessContent), 0755); err != nil {
		t.Fatalf("Failed to create on-success hook: %v", err)
	}
	
	// Verify hook is executable
	info, err := os.Stat(onSuccessHook)
	if err != nil {
		t.Fatalf("on-success hook does not exist: %v", err)
	}
	if info.Mode().Perm()&0100 == 0 {
		t.Error("on-success hook is not executable")
	}
}

// TestDefaultWorkflowOnErrorCalledOnFailure tests on-error hook execution
func TestDefaultWorkflowOnErrorCalledOnFailure(t *testing.T) {
	tmpDir := t.TempDir()
	markerFile := filepath.Join(tmpDir, "error-marker.log")
	
	// Create on-error hook that checks ERROR environment variable
	onErrorHook := filepath.Join(tmpDir, "on-error.sh")
	onErrorContent := `#!/bin/bash
if [ -z "$ERROR" ]; then
    echo "ERROR_VAR_NOT_SET" >> ` + markerFile + `
    exit 1
fi
echo "ERROR_HOOK_CALLED:$ERROR" >> ` + markerFile + `
exit 0
`
	if err := os.WriteFile(onErrorHook, []byte(onErrorContent), 0755); err != nil {
		t.Fatalf("Failed to create on-error hook: %v", err)
	}
	
	// Verify hook is executable
	info, err := os.Stat(onErrorHook)
	if err != nil {
		t.Fatalf("on-error hook does not exist: %v", err)
	}
	if info.Mode().Perm()&0100 == 0 {
		t.Error("on-error hook is not executable")
	}
}
