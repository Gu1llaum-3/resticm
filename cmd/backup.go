package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"resticm/internal/config"
	"resticm/internal/hooks"
	"resticm/internal/restic"
	"resticm/internal/security"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Run backup operation",
	Long: `Execute a restic backup of configured directories.

This command:
  1. Acquires a lock to prevent concurrent runs
  2. Runs pre-backup hook (if configured)
  3. Executes restic backup with configured tags
  4. Runs post-backup hook (if configured)
  5. Sends notifications on error (or success if --notify-success)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBackup(cmd)
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)
	backupCmd.Flags().StringP("tag", "t", "", "Add extra tag to backup")
	backupCmd.Flags().Bool("notify-success", false, "Send notification on success")
	backupCmd.Flags().Bool("no-hooks", false, "Skip all hooks (pre-backup, post-backup, on-error, on-success)")
}

func runBackup(cmd *cobra.Command) error {
	cfg := GetConfig()
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	// Acquire lock
	lock := security.NewLock("")
	if err := lock.Acquire(); err != nil {
		return err
	}
	defer func() { _ = lock.Release() }()

	// Get active backend
	activeBackend, _ := config.GetActiveBackend()

	var repo, password string
	var awsKey, awsSecret string

	if activeBackend == "" || activeBackend == "primary" {
		repo = cfg.Repository
		password = cfg.GetPassword()
		awsKey = cfg.GetAWSAccessKeyID()
		awsSecret = cfg.GetAWSSecretAccessKey()
	} else {
		backend, ok := cfg.Backends[activeBackend]
		if !ok {
			return fmt.Errorf("backend '%s' not found", activeBackend)
		}
		repo = backend.Repository
		password = backend.Password
		awsKey = backend.AWSAccessKeyID
		awsSecret = backend.AWSSecretAccessKey
	}

	// Create executor
	executor := restic.NewExecutor(repo, password)
	executor.SetAWSCredentials(awsKey, awsSecret)
	executor.DryRun = IsDryRun()
	executor.Verbose = IsVerbose()

	// Check restic is installed
	if err := restic.CheckResticInstalled(); err != nil {
		return err
	}

	// Check repository is initialized
	if !executor.IsInitialized() {
		return fmt.Errorf("repository is not initialized. Run 'resticm init' first")
	}

	// Build tags
	extraTag, _ := cmd.Flags().GetString("tag")
	tags := cfg.DefaultTags
	if extraTag != "" {
		tags = append(tags, extraTag)
	}

	// Get hostname
	hostname, _ := os.Hostname()

	// Get notifier
	notifySuccess, _ := cmd.Flags().GetBool("notify-success")
	notifier := GetNotifier(notifySuccess)

	// Check if hooks should be skipped
	noHooks, _ := cmd.Flags().GetBool("no-hooks")
	if noHooks {
		PrintInfo("Skipping all hooks (--no-hooks flag set)")
	}

	// Setup hooks
	var hookRunner *hooks.Runner
	if !noHooks {
		hookRunner = hooks.NewRunner()
		hookRunner.PreBackup = cfg.Hooks.PreBackup
		hookRunner.PostBackup = cfg.Hooks.PostBackup
		hookRunner.OnError = cfg.Hooks.OnError
		hookRunner.OnSuccess = cfg.Hooks.OnSuccess
		hookRunner.DryRun = IsDryRun()
		hookRunner.Verbose = IsVerbose()
		hookRunner.Logger = GetLogger()
	}

	// Run pre-backup hook
	if !noHooks && hookRunner != nil {
		if err := hookRunner.RunPreBackup(); err != nil {
			PrintError("Pre-backup hook failed: %v", err)
			_ = hookRunner.RunOnError(err)
			_ = notifier.NotifyError(
				"❌ Pre-Backup Hook Failed",
				fmt.Sprintf("resticm pre-backup hook failed on %s: %v", hostname, err),
				err,
				map[string]string{
					"host":       hostname,
					"repository": repo,
				},
			)
			return err
		}
	}

	if IsDryRun() {
		PrintInfo("Starting backup (DRY RUN - no changes will be made)...")
	} else {
		PrintInfo("Starting backup...")
	}

	opts := restic.BackupOptions{
		Directories:     cfg.Directories,
		Tags:            tags,
		ExcludePatterns: cfg.ExcludePatterns,
		ExcludeFile:     cfg.ExcludeFile,
		Hostname:        hostname,
	}

	if err := executor.Backup(opts); err != nil {
		PrintError("Backup failed: %v", err)
		if !noHooks && hookRunner != nil {
			_ = hookRunner.RunPostBackup(false, err)
			_ = hookRunner.RunOnError(err)
		}
		_ = notifier.NotifyError(
			"❌ Backup Failed",
			fmt.Sprintf("resticm backup failed on %s: %v", hostname, err),
			err,
			map[string]string{
				"host":       hostname,
				"repository": repo,
			},
		)
		return err
	}

	// Run post-backup hook
	if !noHooks && hookRunner != nil {
		if err := hookRunner.RunPostBackup(true, nil); err != nil {
			PrintError("Post-backup hook failed: %v", err)
			// Don't fail the backup if post-hook fails, but log it
		}
	}

	PrintSuccess("Backup completed successfully")
	if !noHooks && hookRunner != nil {
		_ = hookRunner.RunOnSuccess()
	}
	_ = notifier.NotifySuccess(
		"✅ Backup Completed",
		fmt.Sprintf("resticm backup completed successfully on %s", hostname),
		map[string]string{
			"host":       hostname,
			"repository": repo,
		},
	)
	return nil
}
