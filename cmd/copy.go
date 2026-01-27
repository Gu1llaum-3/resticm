package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"resticm/internal/restic"
	"resticm/internal/security"
)

var copyCmd = &cobra.Command{
	Use:   "copy",
	Short: "Copy snapshots to secondary backends",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCopy(cmd)
	},
}

func init() {
	rootCmd.AddCommand(copyCmd)
	copyCmd.Flags().Bool("all", false, "Copy snapshots from all hosts")
	copyCmd.Flags().StringSlice("to", nil, "Specific backends to copy to")
}

func runCopy(cmd *cobra.Command) error {
	cfg := GetConfig()
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	if len(cfg.CopyToBackends) == 0 {
		PrintInfo("No secondary backends configured for copy.")
		return nil
	}

	// Acquire lock
	lock := security.NewLock("")
	if err := lock.Acquire(); err != nil {
		return err
	}
	defer func() { _ = lock.Release() }()

	// Determine which backends to copy to
	toBackends, _ := cmd.Flags().GetStringSlice("to")
	if len(toBackends) == 0 {
		toBackends = cfg.CopyToBackends
	}

	// Validate backends exist
	for _, name := range toBackends {
		if _, ok := cfg.Backends[name]; !ok {
			return fmt.Errorf("backend '%s' not found in configuration", name)
		}
	}

	// Determine hostname filter
	hostname := ""
	allHosts, _ := cmd.Flags().GetBool("all")
	if !allHosts {
		hostname, _ = os.Hostname()
	}

	// Source credentials
	fromRepo := cfg.Repository
	fromPass := cfg.GetPassword()
	fromAWSKey := cfg.GetAWSAccessKeyID()
	fromAWSSecret := cfg.GetAWSSecretAccessKey()

	PrintInfo("Copying snapshots to %d backend(s)...", len(toBackends))

	var copyErrors []error

	for _, backendName := range toBackends {
		backend := cfg.Backends[backendName]

		fmt.Printf("\n📦 Copying to backend: %s\n", backendName)

		opts := restic.CopyOptions{
			FromRepository:         fromRepo,
			FromPassword:           fromPass,
			FromAWSAccessKeyID:     fromAWSKey,
			FromAWSSecretAccessKey: fromAWSSecret,
			ToRepository:           backend.Repository,
			ToPassword:             backend.Password,
			ToAWSAccessKeyID:       backend.AWSAccessKeyID,
			ToAWSSecretAccessKey:   backend.AWSSecretAccessKey,
			Hostname:               hostname,
		}

		executor := restic.NewExecutor(backend.Repository, backend.Password)
		executor.SetAWSCredentials(backend.AWSAccessKeyID, backend.AWSSecretAccessKey)
		executor.Verbose = IsVerbose()
		executor.DryRun = IsDryRun()

		if err := executor.Copy(opts); err != nil {
			PrintError("Failed to copy to %s: %v", backendName, err)
			copyErrors = append(copyErrors, fmt.Errorf("%s: %w", backendName, err))
			continue
		}

		PrintSuccess("Copy to %s completed", backendName)
	}

	if len(copyErrors) > 0 {
		return fmt.Errorf("%d copy operation(s) failed", len(copyErrors))
	}

	PrintSuccess("All copy operations completed successfully")
	return nil
}
