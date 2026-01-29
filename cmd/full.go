package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"resticm/internal/hooks"
	"resticm/internal/restic"
	"resticm/internal/security"
)

var fullCmd = &cobra.Command{
	Use:   "full",
	Short: "Run full maintenance: backup + forget + prune + check + copy",
	Long: `Run full maintenance workflow on primary AND all copy backends.

This command ensures all repositories stay synchronized by running:
  1. Backup on primary
  2. Forget on primary
  3. Prune on primary
  4. Check on primary (with auto deep-check based on interval)
  5. Copy to secondary backends
  6. Forget on each copy backend (same retention policy)
  7. Prune on each copy backend
  8. Check on each copy backend`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFull(cmd)
	},
}

func init() {
	rootCmd.AddCommand(fullCmd)
	fullCmd.Flags().StringP("tag", "t", "", "Add extra tag to backup")
	fullCmd.Flags().Bool("deep", false, "Force deep check on all backends")
	fullCmd.Flags().Bool("all-hosts", false, "Process snapshots from all hosts")
	fullCmd.Flags().Bool("no-hooks", false, "Skip all hooks (pre-backup, post-backup, on-error, on-success)")
}

func runFull(cmd *cobra.Command) error {
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

	extraTag, _ := cmd.Flags().GetString("tag")
	deep, _ := cmd.Flags().GetBool("deep")
	allHosts, _ := cmd.Flags().GetBool("all-hosts")
	noHooks, _ := cmd.Flags().GetBool("no-hooks")

	if noHooks {
		PrintInfo("Skipping all hooks (--no-hooks flag set)")
	}

	repo := cfg.Repository
	password := cfg.GetPassword()
	awsKey := cfg.GetAWSAccessKeyID()
	awsSecret := cfg.GetAWSSecretAccessKey()

	executor := restic.NewExecutor(repo, password)
	executor.SetAWSCredentials(awsKey, awsSecret)
	executor.DryRun = IsDryRun()
	executor.Verbose = IsVerbose()

	hostname, _ := os.Hostname()

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

	var errors []error
	separator := strings.Repeat("━", 50)

	// Banner
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════")
	if IsDryRun() {
		fmt.Println(" RESTICM FULL MAINTENANCE (DRY RUN)")
	} else {
		fmt.Println(" RESTICM FULL MAINTENANCE")
	}
	fmt.Println("═══════════════════════════════════════════════════")

	// 1. BACKUP
	fmt.Println("\n" + separator)
	fmt.Println("📦 STEP 1/5: BACKUP")
	fmt.Println(separator)

	// Run pre-backup hook
	if !noHooks && hookRunner != nil {
		if err := hookRunner.RunPreBackup(); err != nil {
			PrintError("Pre-backup hook failed: %v", err)
			errors = append(errors, err)
			_ = hookRunner.RunOnError(err)
			_ = GetNotifier(false).NotifyError(
				"❌ Pre-Backup Hook Failed",
				fmt.Sprintf("resticm pre-backup hook failed on %s: %v", hostname, err),
				err,
				map[string]string{
					"host":       hostname,
					"repository": cfg.Repository,
				},
			)
			return err
		}
	}

	tags := cfg.DefaultTags
	if extraTag != "" {
		tags = append(tags, extraTag)
	}

	backupOpts := restic.BackupOptions{
		Directories:     cfg.Directories,
		Tags:            tags,
		ExcludePatterns: cfg.ExcludePatterns,
		ExcludeFile:     cfg.ExcludeFile,
		Hostname:        hostname,
	}

	if err := executor.Backup(backupOpts); err != nil {
		PrintError("Backup failed: %v", err)
		errors = append(errors, err)
		if !noHooks && hookRunner != nil {
			_ = hookRunner.RunPostBackup(false, err)
			_ = hookRunner.RunOnError(err)
		}
	} else {
		PrintSuccess("Backup completed")
		if !noHooks && hookRunner != nil {
			_ = hookRunner.RunPostBackup(true, nil)
		}
	}

	// 2. FORGET
	fmt.Println("\n" + separator)
	fmt.Println("🗑️  STEP 2/5: FORGET")
	fmt.Println(separator)

	forgetHostname := hostname
	if allHosts {
		forgetHostname = ""
	}

	forgetOpts := restic.ForgetOptions{
		KeepWithin:  cfg.Retention.KeepWithin,
		KeepHourly:  cfg.Retention.KeepHourly,
		KeepDaily:   cfg.Retention.KeepDaily,
		KeepWeekly:  cfg.Retention.KeepWeekly,
		KeepMonthly: cfg.Retention.KeepMonthly,
		KeepYearly:  cfg.Retention.KeepYearly,
		Hostname:    forgetHostname,
	}

	if err := executor.Forget(forgetOpts); err != nil {
		PrintError("Forget failed: %v", err)
		errors = append(errors, err)
	} else {
		PrintSuccess("Forget completed")
	}

	// 3. PRUNE
	fmt.Println("\n" + separator)
	fmt.Println("🧹 STEP 3/5: PRUNE")
	fmt.Println(separator)

	if err := executor.Prune(); err != nil {
		PrintError("Prune failed: %v", err)
		errors = append(errors, err)
	} else {
		PrintSuccess("Prune completed")
	}

	// 4. CHECK
	fmt.Println("\n" + separator)
	fmt.Println("🔍 STEP 4/5: CHECK")
	fmt.Println(separator)

	shouldDeep := deep
	if !shouldDeep && cfg.DeepCheckIntervalDays > 0 {
		if tracker, err := restic.NewDeepCheckTracker(repo); err == nil {
			shouldDeep = tracker.ShouldRunDeepCheck(cfg.DeepCheckIntervalDays)
		}
	}

	checkOpts := restic.CheckOptions{ReadData: shouldDeep}

	if err := executor.Check(checkOpts); err != nil {
		PrintError("Check failed: %v", err)
		errors = append(errors, err)
	} else {
		PrintSuccess("Check passed")
		if shouldDeep {
			if tracker, err := restic.NewDeepCheckTracker(repo); err == nil {
				_ = tracker.RecordCheck()
			}
		}
	}

	// 5. COPY + MAINTENANCE ON COPY BACKENDS
	fmt.Println("\n" + separator)
	fmt.Println("📤 STEP 5/5: COPY & SYNC BACKENDS")
	fmt.Println(separator)

	if len(cfg.CopyToBackends) == 0 {
		PrintInfo("No secondary backends configured, skipping copy")
	} else {
		copyHostname := hostname
		if allHosts {
			copyHostname = ""
		}

		for _, backendName := range cfg.CopyToBackends {
			backend := cfg.Backends[backendName]

			fmt.Printf("\n  ┌─ Backend: %s\n", backendName)

			// 5a. COPY to this backend
			fmt.Println("  │ 📦 Copying snapshots...")
			copyOpts := restic.CopyOptions{
				FromRepository:         repo,
				FromPassword:           password,
				FromAWSAccessKeyID:     awsKey,
				FromAWSSecretAccessKey: awsSecret,
				ToRepository:           backend.Repository,
				ToPassword:             backend.Password,
				ToAWSAccessKeyID:       backend.AWSAccessKeyID,
				ToAWSSecretAccessKey:   backend.AWSSecretAccessKey,
				Hostname:               copyHostname,
			}

			destExecutor := restic.NewExecutor(backend.Repository, backend.Password)
			destExecutor.SetAWSCredentials(backend.AWSAccessKeyID, backend.AWSSecretAccessKey)
			destExecutor.Verbose = IsVerbose()
			destExecutor.DryRun = IsDryRun()

			if err := destExecutor.Copy(copyOpts); err != nil {
				PrintError("Copy to %s failed: %v", backendName, err)
				errors = append(errors, err)
				fmt.Println("  └─ ❌ Skipping maintenance due to copy failure")
				continue
			}
			PrintSuccess("Copy to %s completed", backendName)

			// 5b. FORGET on this backend
			fmt.Println("  │ 🗑️  Applying retention policy...")
			if err := destExecutor.Forget(forgetOpts); err != nil {
				PrintError("Forget on %s failed: %v", backendName, err)
				errors = append(errors, err)
			} else {
				PrintSuccess("Forget on %s completed", backendName)
			}

			// 5c. PRUNE on this backend
			fmt.Println("  │ 🧹 Pruning unused data...")
			if err := destExecutor.Prune(); err != nil {
				PrintError("Prune on %s failed: %v", backendName, err)
				errors = append(errors, err)
			} else {
				PrintSuccess("Prune on %s completed", backendName)
			}

			// 5d. CHECK on this backend
			fmt.Println("  │ 🔍 Checking integrity...")
			backendShouldDeep := deep
			if !backendShouldDeep && cfg.DeepCheckIntervalDays > 0 {
				if tracker, err := restic.NewDeepCheckTracker(backend.Repository); err == nil {
					backendShouldDeep = tracker.ShouldRunDeepCheck(cfg.DeepCheckIntervalDays)
				}
			}
			backendCheckOpts := restic.CheckOptions{ReadData: backendShouldDeep}
			if err := destExecutor.Check(backendCheckOpts); err != nil {
				PrintError("Check on %s failed: %v", backendName, err)
				errors = append(errors, err)
			} else {
				PrintSuccess("Check on %s passed", backendName)
				if backendShouldDeep {
					if tracker, err := restic.NewDeepCheckTracker(backend.Repository); err == nil {
						_ = tracker.RecordCheck()
					}
				}
			}
			fmt.Println("  └─ ✅ Backend synchronized")
		}
	}

	// Summary
	fmt.Println("\n" + strings.Repeat("═", 50))
	fmt.Println("📊 SUMMARY")
	fmt.Println(strings.Repeat("═", 50))

	// Get notifier for notifications
	notifier := GetNotifier(false)

	if len(errors) == 0 {
		PrintSuccess("All operations completed successfully!")
		if !noHooks && hookRunner != nil {
			_ = hookRunner.RunOnSuccess()
		}
		_ = notifier.NotifySuccess(
			"✅ Full Maintenance Completed",
			fmt.Sprintf("resticm full completed successfully on %s", hostname),
			map[string]string{
				"host":       hostname,
				"repository": cfg.Repository,
				"backends":   fmt.Sprintf("%d", len(cfg.CopyToBackends)+1),
			},
		)
	} else {
		PrintError("%d operation(s) failed", len(errors))
		var errMsgs []string
		for _, e := range errors {
			errMsgs = append(errMsgs, e.Error())
		}
		finalErr := fmt.Errorf("%d operation(s) failed", len(errors))
		if !noHooks && hookRunner != nil {
			_ = hookRunner.RunOnError(finalErr)
		}
		_ = notifier.NotifyError(
			"❌ Full Maintenance Failed",
			fmt.Sprintf("resticm full failed on %s with %d error(s)", hostname, len(errors)),
			nil,
			map[string]string{
				"host":       hostname,
				"repository": cfg.Repository,
				"errors":     strings.Join(errMsgs, "; "),
			},
		)
		return finalErr
	}

	return nil
}
