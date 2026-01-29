package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"resticm/internal/config"
	"resticm/internal/restic"
	"resticm/internal/security"
)

var forgetCmd = &cobra.Command{
	Use:   "forget",
	Short: "Remove old snapshots according to retention policy",
	Long: `Remove old snapshots according to retention policy.

By default, this command applies to the primary repository AND all configured
copy backends to keep them synchronized. Use --primary-only to only affect
the primary repository.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runForget(cmd)
	},
}

func init() {
	rootCmd.AddCommand(forgetCmd)
	forgetCmd.Flags().Bool("all-hosts", false, "Process snapshots from all hosts")
	forgetCmd.Flags().BoolP("prune", "p", false, "Also run prune after forget")
	forgetCmd.Flags().Bool("primary-only", false, "Only apply to primary repository (skip copy backends)")
}

func runForget(cmd *cobra.Command) (err error) {
	startTime := time.Now()

	cfg := GetConfig()
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	allHosts, _ := cmd.Flags().GetBool("all-hosts")
	prune, _ := cmd.Flags().GetBool("prune")
	primaryOnly, _ := cmd.Flags().GetBool("primary-only")

	// Build flag map for logging
	flagMap := make(map[string]interface{})
	if allHosts {
		flagMap["all-hosts"] = true
	}
	if prune {
		flagMap["prune"] = true
	}
	if primaryOnly {
		flagMap["primary-only"] = true
	}

	// Log command start with context
	LogCommandStart(cmd, flagMap)

	// Ensure we log command end
	defer func() {
		LogCommandEnd(cmd, startTime, err)
	}()

	// Acquire lock
	lock := security.NewLock("")
	if err = lock.Acquire(); err != nil {
		return err
	}
	defer func() { _ = lock.Release() }()

	// Determine hostname filter
	hostname := ""
	if !allHosts {
		hostname, _ = os.Hostname()
	}

	// Get notifier for error notifications
	notifier := GetNotifier(false)
	currentHost, _ := os.Hostname()

	var forgetErrors []error

	opts := restic.ForgetOptions{
		KeepWithin:  cfg.Retention.KeepWithin,
		KeepHourly:  cfg.Retention.KeepHourly,
		KeepDaily:   cfg.Retention.KeepDaily,
		KeepWeekly:  cfg.Retention.KeepWeekly,
		KeepMonthly: cfg.Retention.KeepMonthly,
		KeepYearly:  cfg.Retention.KeepYearly,
		Hostname:    hostname,
		Prune:       prune,
	}

	// Get active backend (if user explicitly selected one)
	activeBackend, _ := config.GetActiveBackend()

	// If user explicitly selected a specific backend, only operate on that one
	if activeBackend != "" && activeBackend != "primary" {
		backend, ok := cfg.Backends[activeBackend]
		if !ok {
			return fmt.Errorf("backend '%s' not found", activeBackend)
		}
		err := forgetOnBackend(activeBackend, backend.Repository, backend.Password,
			backend.AWSAccessKeyID, backend.AWSSecretAccessKey, opts)
		if err != nil {
			_ = notifier.NotifyError(
				"❌ Forget Failed",
				fmt.Sprintf("resticm forget failed on %s backend '%s'", currentHost, activeBackend),
				err,
				map[string]string{"host": currentHost, "backend": activeBackend},
			)
		}
		return err
	}

	// Run forget on primary repository
	PrintInfo("🗑️  Running forget on primary repository...")
	if err := forgetOnBackend("primary", cfg.Repository, cfg.GetPassword(),
		cfg.GetAWSAccessKeyID(), cfg.GetAWSSecretAccessKey(), opts); err != nil {
		forgetErrors = append(forgetErrors, fmt.Errorf("primary: %w", err))
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
			PrintInfo("🗑️  Running forget on backend: %s", backendName)
			if err := forgetOnBackend(backendName, backend.Repository, backend.Password,
				backend.AWSAccessKeyID, backend.AWSSecretAccessKey, opts); err != nil {
				PrintError("Forget failed on backend '%s': %v", backendName, err)
				forgetErrors = append(forgetErrors, fmt.Errorf("%s: %w", backendName, err))
			}
		}
	}

	// Send notification if any forget failed
	if len(forgetErrors) > 0 {
		var errMsgs []string
		for _, e := range forgetErrors {
			errMsgs = append(errMsgs, e.Error())
		}
		_ = notifier.NotifyError(
			"❌ Forget Failed",
			fmt.Sprintf("resticm forget failed on %s - %d backend(s) affected", currentHost, len(forgetErrors)),
			nil,
			map[string]string{"host": currentHost, "errors": strings.Join(errMsgs, "; ")},
		)
		return fmt.Errorf("%d forget operation(s) failed", len(forgetErrors))
	}

	PrintSuccess("Forget completed on all backends")
	return nil
}

func forgetOnBackend(name, repo, password, awsKey, awsSecret string, opts restic.ForgetOptions) error {
	executor := restic.NewExecutor(repo, password)
	executor.SetAWSCredentials(awsKey, awsSecret)
	executor.DryRun = IsDryRun()
	executor.Verbose = IsVerbose()

	if err := executor.Forget(opts); err != nil {
		PrintError("Forget failed on %s: %v", name, err)
		return err
	}

	PrintSuccess("Forget completed on %s", name)
	return nil
}
