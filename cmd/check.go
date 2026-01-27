package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"resticm/internal/config"
	"resticm/internal/restic"
	"resticm/internal/security"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Verify repository integrity",
	Long: `Verify repository integrity.

By default, this command applies to the primary repository AND all configured
copy backends to keep them synchronized. Use --primary-only to only affect
the primary repository.

Modes:
  - Default: metadata check only (fast)
  - --deep: read all data blobs (slow but thorough)
  - --auto: automatically run deep check if interval has elapsed
  - --subset: read a subset of data (e.g., '1/5' for 20%)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCheck(cmd)
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
	checkCmd.Flags().Bool("deep", false, "Read all data (slower but more thorough)")
	checkCmd.Flags().Bool("auto", false, "Automatically run deep check if interval has elapsed")
	checkCmd.Flags().String("subset", "", "Read a subset of data (e.g., '1/5' for 20%)")
	checkCmd.Flags().Bool("primary-only", false, "Only apply to primary repository (skip copy backends)")
}

func runCheck(cmd *cobra.Command) error {
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

	deep, _ := cmd.Flags().GetBool("deep")
	auto, _ := cmd.Flags().GetBool("auto")
	subset, _ := cmd.Flags().GetString("subset")
	primaryOnly, _ := cmd.Flags().GetBool("primary-only")

	// Get notifier for error notifications (check failures are CRITICAL)
	notifier := GetNotifier(false)
	hostname, _ := os.Hostname()

	var checkErrors []error

	// Get active backend (if user explicitly selected one)
	activeBackend, _ := config.GetActiveBackend()

	// If user explicitly selected a specific backend, only operate on that one
	if activeBackend != "" && activeBackend != "primary" {
		backend, ok := cfg.Backends[activeBackend]
		if !ok {
			return fmt.Errorf("backend '%s' not found", activeBackend)
		}
		err := checkOnBackend(activeBackend, backend.Repository, backend.Password,
			backend.AWSAccessKeyID, backend.AWSSecretAccessKey, deep, auto, subset, cfg.DeepCheckIntervalDays)
		if err != nil {
			_ = notifier.NotifyError(
				"🚨 Repository Check FAILED",
				fmt.Sprintf("CRITICAL: Repository integrity check failed on %s backend '%s'", hostname, activeBackend),
				err,
				map[string]string{
					"host":       hostname,
					"backend":    activeBackend,
					"repository": backend.Repository,
				},
			)
		}
		return err
	}

	// Run check on primary repository
	PrintInfo("🔍 Running check on primary repository...")
	if err := checkOnBackend("primary", cfg.Repository, cfg.GetPassword(),
		cfg.GetAWSAccessKeyID(), cfg.GetAWSSecretAccessKey(), deep, auto, subset, cfg.DeepCheckIntervalDays); err != nil {
		checkErrors = append(checkErrors, fmt.Errorf("primary: %w", err))
	}

	// Apply to copy backends unless --primary-only is set
	if !primaryOnly && len(cfg.CopyToBackends) > 0 {
		for _, backendName := range cfg.CopyToBackends {
			backend, ok := cfg.Backends[backendName]
			if !ok {
				PrintWarning("Backend '%s' not found in configuration, skipping", backendName)
				continue
			}

			fmt.Println()
			PrintInfo("🔍 Running check on backend: %s", backendName)
			if err := checkOnBackend(backendName, backend.Repository, backend.Password,
				backend.AWSAccessKeyID, backend.AWSSecretAccessKey, deep, auto, subset, cfg.DeepCheckIntervalDays); err != nil {
				PrintError("Check failed on backend '%s': %v", backendName, err)
				checkErrors = append(checkErrors, fmt.Errorf("%s: %w", backendName, err))
			}
		}
	}

	// Send notification if any check failed
	if len(checkErrors) > 0 {
		var errMsgs []string
		for _, e := range checkErrors {
			errMsgs = append(errMsgs, e.Error())
		}
		_ = notifier.NotifyError(
			"🚨 Repository Check FAILED",
			fmt.Sprintf("CRITICAL: Repository integrity check failed on %s - %d backend(s) affected", hostname, len(checkErrors)),
			nil,
			map[string]string{
				"host":   hostname,
				"errors": strings.Join(errMsgs, "; "),
			},
		)
		return fmt.Errorf("%d check(s) failed", len(checkErrors))
	}

	PrintSuccess("Check completed on all backends")
	return nil
}

func checkOnBackend(name, repo, password, awsKey, awsSecret string, deep, auto bool, subset string, deepCheckIntervalDays int) error {
	executor := restic.NewExecutor(repo, password)
	executor.SetAWSCredentials(awsKey, awsSecret)
	executor.Verbose = IsVerbose()

	// Determine if we should do a deep check
	doDeepCheck := deep

	// If --auto is set, check if deep check interval has elapsed
	if auto && !deep {
		tracker, err := restic.NewDeepCheckTracker(repo)
		if err == nil && tracker.ShouldRunDeepCheck(deepCheckIntervalDays) {
			PrintInfo("Deep check interval elapsed for %s, running deep check...", name)
			doDeepCheck = true
		}
	}

	opts := restic.CheckOptions{
		ReadData:       doDeepCheck,
		ReadDataSubset: subset,
	}

	if doDeepCheck {
		PrintInfo("Running deep check on %s (reading all data)...", name)
	} else {
		PrintInfo("Running metadata check on %s...", name)
	}

	if err := executor.Check(opts); err != nil {
		PrintError("Check failed on %s: %v", name, err)
		return err
	}

	// Record deep check if performed
	if doDeepCheck {
		if tracker, err := restic.NewDeepCheckTracker(repo); err == nil {
			tracker.RecordCheck()
		}
	}

	PrintSuccess("Check passed on %s", name)
	return nil
}
