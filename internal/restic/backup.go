package restic

import (
	"os"
)

// BackupOptions contains options for the backup operation
type BackupOptions struct {
	Directories     []string
	Tags            []string
	ExcludePatterns []string
	ExcludeFile     string
	ExtraArgs       []string
	Hostname        string
}

// Backup performs a backup operation
func (e *Executor) Backup(opts BackupOptions) error {
	args := []string{"backup"}

	// Add directories
	args = append(args, opts.Directories...)

	// Add tags
	for _, tag := range opts.Tags {
		args = append(args, "--tag", tag)
	}

	// Add exclude patterns
	for _, pattern := range opts.ExcludePatterns {
		args = append(args, "--exclude", pattern)
	}

	// Add exclude file
	if opts.ExcludeFile != "" {
		if _, err := os.Stat(opts.ExcludeFile); err == nil {
			args = append(args, "--exclude-file", opts.ExcludeFile)
		}
	}

	// Add hostname
	if opts.Hostname != "" {
		args = append(args, "--host", opts.Hostname)
	}

	// Add extra arguments
	args = append(args, opts.ExtraArgs...)

	// Add dry-run if enabled
	if e.DryRun {
		args = append(args, "--dry-run")
	}

	return e.Run(args...)
}
