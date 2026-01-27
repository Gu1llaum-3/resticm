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

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove unused data from repository",
	Long: `Remove unused data from repository.

By default, this command applies to the primary repository AND all configured
copy backends to keep them synchronized. Use --primary-only to only affect
the primary repository.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPrune(cmd)
	},
}

func init() {
	rootCmd.AddCommand(pruneCmd)
	pruneCmd.Flags().Bool("primary-only", false, "Only apply to primary repository (skip copy backends)")
}

func runPrune(cmd *cobra.Command) error {
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

	primaryOnly, _ := cmd.Flags().GetBool("primary-only")

	// Get notifier for error notifications
	notifier := GetNotifier(false)
	hostname, _ := os.Hostname()

	var pruneErrors []error

	// Get active backend (if user explicitly selected one)
	activeBackend, _ := config.GetActiveBackend()

	// If user explicitly selected a specific backend, only operate on that one
	if activeBackend != "" && activeBackend != "primary" {
		backend, ok := cfg.Backends[activeBackend]
		if !ok {
			return fmt.Errorf("backend '%s' not found", activeBackend)
		}
		err := pruneOnBackend(activeBackend, backend.Repository, backend.Password,
			backend.AWSAccessKeyID, backend.AWSSecretAccessKey)
		if err != nil {
			_ = notifier.NotifyError(
				"❌ Prune Failed",
				fmt.Sprintf("resticm prune failed on %s backend '%s'", hostname, activeBackend),
				err,
				map[string]string{"host": hostname, "backend": activeBackend},
			)
		}
		return err
	}

	// Run prune on primary repository
	PrintInfo("🧹 Running prune on primary repository...")
	if err := pruneOnBackend("primary", cfg.Repository, cfg.GetPassword(),
		cfg.GetAWSAccessKeyID(), cfg.GetAWSSecretAccessKey()); err != nil {
		pruneErrors = append(pruneErrors, fmt.Errorf("primary: %w", err))
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
			PrintInfo("🧹 Running prune on backend: %s", backendName)
			if err := pruneOnBackend(backendName, backend.Repository, backend.Password,
				backend.AWSAccessKeyID, backend.AWSSecretAccessKey); err != nil {
				PrintError("Prune failed on backend '%s': %v", backendName, err)
				pruneErrors = append(pruneErrors, fmt.Errorf("%s: %w", backendName, err))
			}
		}
	}

	// Send notification if any prune failed
	if len(pruneErrors) > 0 {
		var errMsgs []string
		for _, e := range pruneErrors {
			errMsgs = append(errMsgs, e.Error())
		}
		_ = notifier.NotifyError(
			"❌ Prune Failed",
			fmt.Sprintf("resticm prune failed on %s - %d backend(s) affected", hostname, len(pruneErrors)),
			nil,
			map[string]string{"host": hostname, "errors": strings.Join(errMsgs, "; ")},
		)
		return fmt.Errorf("%d prune operation(s) failed", len(pruneErrors))
	}

	PrintSuccess("Prune completed on all backends")
	return nil
}

func pruneOnBackend(name, repo, password, awsKey, awsSecret string) error {
	executor := restic.NewExecutor(repo, password)
	executor.SetAWSCredentials(awsKey, awsSecret)
	executor.DryRun = IsDryRun()
	executor.Verbose = IsVerbose()

	if err := executor.Prune(); err != nil {
		PrintError("Prune failed on %s: %v", name, err)
		return err
	}

	PrintSuccess("Prune completed on %s", name)
	return nil
}
