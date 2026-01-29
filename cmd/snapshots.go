package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"resticm/internal/config"
	"resticm/internal/restic"
)

var snapshotsCmd = &cobra.Command{
	Use:   "snapshots",
	Short: "List repository snapshots",
	Long: `List repository snapshots.

By default, shows snapshots from the active backend only.
Use --all-backends to show snapshots from all configured backends.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSnapshots(cmd)
	},
}

func init() {
	rootCmd.AddCommand(snapshotsCmd)
	snapshotsCmd.Flags().Bool("all", false, "Show snapshots from all hosts")
	snapshotsCmd.Flags().Bool("latest", false, "Show only the latest snapshot")
	snapshotsCmd.Flags().Bool("all-backends", false, "Show snapshots from all configured backends")
}

func runSnapshots(cmd *cobra.Command) (err error) {
	startTime := time.Now()

	cfg := GetConfig()
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	showLatest, _ := cmd.Flags().GetBool("latest")
	showAll, _ := cmd.Flags().GetBool("all")
	allBackends, _ := cmd.Flags().GetBool("all-backends")

	// Build flag map for logging
	flagMap := make(map[string]interface{})
	if showLatest {
		flagMap["latest"] = true
	}
	if showAll {
		flagMap["all"] = true
	}
	if allBackends {
		flagMap["all-backends"] = true
	}

	// Log command start with context
	LogCommandStart(cmd, flagMap)

	// Ensure we log command end
	defer func() {
		LogCommandEnd(cmd, startTime, err)
	}()

	// If --all-backends, show snapshots from all backends
	if allBackends {
		return showSnapshotsAllBackends(cfg, showAll, showLatest)
	}

	// Get active backend
	activeBackend, _ := config.GetActiveBackend()

	var repo, password string
	var awsKey, awsSecret string
	var backendName string

	if activeBackend == "" || activeBackend == "primary" {
		repo = cfg.Repository
		password = cfg.GetPassword()
		awsKey = cfg.GetAWSAccessKeyID()
		awsSecret = cfg.GetAWSSecretAccessKey()
		backendName = "primary"
	} else {
		backend, ok := cfg.Backends[activeBackend]
		if !ok {
			return fmt.Errorf("backend '%s' not found", activeBackend)
		}
		repo = backend.Repository
		password = backend.Password
		awsKey = backend.AWSAccessKeyID
		awsSecret = backend.AWSSecretAccessKey
		backendName = activeBackend
	}

	return showSnapshotsForBackend(backendName, repo, password, awsKey, awsSecret, showAll, showLatest)
}

func showSnapshotsAllBackends(cfg *config.Config, showAll, showLatest bool) error {
	// Primary
	fmt.Println("\n═══ PRIMARY BACKEND ═══")
	if err := showSnapshotsForBackend("primary", cfg.Repository, cfg.GetPassword(),
		cfg.GetAWSAccessKeyID(), cfg.GetAWSSecretAccessKey(), showAll, showLatest); err != nil {
		PrintError("Failed to list snapshots on primary: %v", err)
	}

	// Copy backends
	for _, backendName := range cfg.CopyToBackends {
		backend, ok := cfg.Backends[backendName]
		if !ok {
			continue
		}
		fmt.Printf("\n═══ BACKEND: %s ═══\n", backendName)
		if err := showSnapshotsForBackend(backendName, backend.Repository, backend.Password,
			backend.AWSAccessKeyID, backend.AWSSecretAccessKey, showAll, showLatest); err != nil {
			PrintError("Failed to list snapshots on %s: %v", backendName, err)
		}
	}

	return nil
}

func showSnapshotsForBackend(name, repo, password, awsKey, awsSecret string, showAll, showLatest bool) error {
	executor := restic.NewExecutor(repo, password)
	executor.SetAWSCredentials(awsKey, awsSecret)
	executor.Verbose = IsVerbose()

	if showLatest {
		snapshot, err := executor.GetLatestSnapshot()
		if err != nil {
			return err
		}
		if snapshot == nil {
			PrintInfo("No snapshots found")
			return nil
		}

		if IsJSONOutput() {
			output, _ := json.MarshalIndent(snapshot, "", "  ")
			fmt.Println(string(output))
		} else {
			fmt.Printf("Latest snapshot:\n")
			fmt.Printf("  ID:   %s\n", snapshot.ShortID)
			fmt.Printf("  Time: %s\n", snapshot.Time.Format("2006-01-02 15:04:05"))
			fmt.Printf("  Host: %s\n", snapshot.Hostname)
		}
		return nil
	}

	snapshots, err := executor.ListSnapshots()
	if err != nil {
		return err
	}

	if len(snapshots) == 0 {
		PrintInfo("No snapshots found")
		return nil
	}

	hostname := ""
	if !showAll {
		hostname, _ = restic.GetCurrentHostname()
	}

	if IsJSONOutput() {
		var filtered []restic.Snapshot
		for _, s := range snapshots {
			if hostname == "" || s.Hostname == hostname {
				filtered = append(filtered, s)
			}
		}
		output, _ := json.MarshalIndent(filtered, "", "  ")
		fmt.Println(string(output))
		return nil
	}

	fmt.Println()
	fmt.Printf("%-12s %-20s %-15s %s\n", "ID", "TIME", "HOST", "PATHS")
	fmt.Println("────────────────────────────────────────────────────────────────")

	count := 0
	for _, s := range snapshots {
		if hostname != "" && s.Hostname != hostname {
			continue
		}
		count++
		fmt.Printf("%-12s %-20s %-15s %v\n",
			s.ShortID,
			s.Time.Format("2006-01-02 15:04"),
			s.Hostname,
			s.Paths,
		)
	}

	fmt.Println()
	PrintInfo("Total: %d snapshot(s)", count)
	return nil
}
