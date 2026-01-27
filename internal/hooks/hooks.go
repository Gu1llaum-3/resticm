// Package hooks provides hook execution functionality for resticm
package hooks

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

// Runner executes hook scripts
type Runner struct {
	PreBackup  string
	PostBackup string
	OnError    string
	OnSuccess  string
	DryRun     bool
	Env        []string
	Verbose    bool
}

// NewRunner creates a new hook runner
func NewRunner() *Runner {
	return &Runner{}
}

// Run executes a hook script
func (r *Runner) Run(path string, extraEnv []string) (string, error) {
	// Empty path means no hook configured
	if path == "" {
		return "", nil
	}

	// Check if hook exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Hook not configured, silently skip
		return "", nil
	}

	// In dry-run mode, don't execute
	if r.DryRun {
		return "", nil
	}

	// Prepare command
	cmd := exec.Command(path)
	cmd.Env = append(os.Environ(), r.Env...)
	cmd.Env = append(cmd.Env, extraEnv...)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute
	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		output += stderr.String()
	}

	if err != nil {
		return output, fmt.Errorf("hook %s failed: %w\nOutput: %s", path, err, output)
	}

	return output, nil
}

// RunPreBackup executes the pre-backup hook
func (r *Runner) RunPreBackup() error {
	_, err := r.Run(r.PreBackup, nil)
	return err
}

// RunPostBackup executes the post-backup hook
func (r *Runner) RunPostBackup(success bool, backupErr error) error {
	var env []string
	if success {
		env = append(env, "BACKUP_STATUS=success")
	} else {
		env = append(env, "BACKUP_STATUS=failure")
		if backupErr != nil {
			env = append(env, fmt.Sprintf("BACKUP_ERROR=%s", backupErr.Error()))
		}
	}
	_, err := r.Run(r.PostBackup, env)
	return err
}

// RunOnError executes the on-error hook
func (r *Runner) RunOnError(opErr error) error {
	env := []string{fmt.Sprintf("ERROR=%s", opErr.Error())}
	_, err := r.Run(r.OnError, env)
	return err
}

// RunOnSuccess executes the on-success hook
func (r *Runner) RunOnSuccess() error {
	_, err := r.Run(r.OnSuccess, nil)
	return err
}
