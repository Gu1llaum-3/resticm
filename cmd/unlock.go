package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"resticm/internal/config"
	"resticm/internal/restic"
	"resticm/internal/security"
)

var unlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Remove stale lock files",
	Long: `Remove stale lock files from resticm and/or restic repositories.

Use this command if a previous resticm process crashed and left
a lock file behind.

By default, unlocks the active backend only.
Use --all-backends to unlock all configured backends.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUnlock(cmd)
	},
}

func init() {
	rootCmd.AddCommand(unlockCmd)
	unlockCmd.Flags().BoolP("force", "f", false, "Force unlock without confirmation")
	unlockCmd.Flags().Bool("all-backends", false, "Unlock all configured backends")
	unlockCmd.Flags().Bool("restic", false, "Also unlock restic repository locks")
}

func runUnlock(cmd *cobra.Command) (err error) {
	startTime := time.Now()

	force, _ := cmd.Flags().GetBool("force")
	allBackends, _ := cmd.Flags().GetBool("all-backends")
	unlockRestic, _ := cmd.Flags().GetBool("restic")

	// Build flag map for logging
	flagMap := make(map[string]interface{})
	if force {
		flagMap["force"] = true
	}
	if allBackends {
		flagMap["all-backends"] = true
	}
	if unlockRestic {
		flagMap["restic"] = true
	}

	// Log command start with context
	LogCommandStart(cmd, flagMap)

	// Ensure we log command end
	defer func() {
		LogCommandEnd(cmd, startTime, err)
	}()

	// Handle resticm lock file
	lock := security.NewLock("")

	if lock.IsLocked() {
		lock.PrintLockInfo()

		if !force {
			fmt.Println()
			fmt.Print("Remove resticm lock file? [y/N] ")
			var response string
			_, _ = fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				PrintInfo("Cancelled")
			} else {
				if err := lock.ForceUnlock(); err != nil {
					return err
				}
				PrintSuccess("Resticm lock file removed")
			}
		} else {
			if err := lock.ForceUnlock(); err != nil {
				return err
			}
			PrintSuccess("Resticm lock file removed")
		}
	} else {
		PrintInfo("No resticm lock file present")
	}

	// If --restic flag, also unlock restic repositories
	if unlockRestic {
		cfg := GetConfig()
		if cfg == nil {
			return fmt.Errorf("configuration not loaded")
		}

		if allBackends {
			return unlockAllResticRepos(cfg, force)
		}

		// Unlock active backend only
		activeBackend, _ := config.GetActiveBackend()
		var repo, password, awsKey, awsSecret string

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

		return unlockResticRepo("active", repo, password, awsKey, awsSecret)
	}

	return nil
}

func unlockAllResticRepos(cfg *config.Config, force bool) error {
	PrintInfo("Unlocking all restic repositories...")

	// Primary
	if err := unlockResticRepo("primary", cfg.Repository, cfg.GetPassword(),
		cfg.GetAWSAccessKeyID(), cfg.GetAWSSecretAccessKey()); err != nil {
		PrintError("Failed to unlock primary: %v", err)
	}

	// Copy backends
	for _, backendName := range cfg.CopyToBackends {
		backend, ok := cfg.Backends[backendName]
		if !ok {
			continue
		}
		if err := unlockResticRepo(backendName, backend.Repository, backend.Password,
			backend.AWSAccessKeyID, backend.AWSSecretAccessKey); err != nil {
			PrintError("Failed to unlock %s: %v", backendName, err)
		}
	}

	return nil
}

func unlockResticRepo(name, repo, password, awsKey, awsSecret string) error {
	executor := restic.NewExecutor(repo, password)
	executor.SetAWSCredentials(awsKey, awsSecret)

	PrintInfo("Unlocking restic repository: %s", name)
	if err := executor.Run("unlock"); err != nil {
		return err
	}
	PrintSuccess("Restic repository %s unlocked", name)
	return nil
}
