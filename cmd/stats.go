package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"resticm/internal/config"
	"resticm/internal/restic"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show repository statistics",
	Long: `Show repository statistics.

By default, shows stats from the active backend only.
Use --all-backends to show stats from all configured backends.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStats(cmd)
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)
	statsCmd.Flags().Bool("all-backends", false, "Show stats from all configured backends")
}

func runStats(cmd *cobra.Command) error {
	cfg := GetConfig()
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	allBackends, _ := cmd.Flags().GetBool("all-backends")

	if allBackends {
		return showStatsAllBackends(cfg)
	}

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

	return showStatsForBackend(backendName, repo, password, awsKey, awsSecret)
}

func showStatsAllBackends(cfg *config.Config) error {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════")
	fmt.Println(" ALL BACKENDS STATISTICS")
	fmt.Println("═══════════════════════════════════════")

	// Primary
	fmt.Println("\n┌─ PRIMARY")
	if err := showStatsForBackend("primary", cfg.Repository, cfg.GetPassword(),
		cfg.GetAWSAccessKeyID(), cfg.GetAWSSecretAccessKey()); err != nil {
		PrintError("Failed to get stats for primary: %v", err)
	}

	// Copy backends
	for _, backendName := range cfg.CopyToBackends {
		backend, ok := cfg.Backends[backendName]
		if !ok {
			continue
		}
		fmt.Printf("\n┌─ BACKEND: %s\n", backendName)
		if err := showStatsForBackend(backendName, backend.Repository, backend.Password,
			backend.AWSAccessKeyID, backend.AWSSecretAccessKey); err != nil {
			PrintError("Failed to get stats for %s: %v", backendName, err)
		}
	}

	return nil
}

func showStatsForBackend(name, repo, password, awsKey, awsSecret string) error {
	executor := restic.NewExecutor(repo, password)
	executor.SetAWSCredentials(awsKey, awsSecret)

	stats, err := executor.GetStats()
	if err != nil {
		return err
	}

	if IsJSONOutput() {
		output, _ := json.MarshalIndent(stats, "", "  ")
		fmt.Println(string(output))
		return nil
	}

	fmt.Printf("│ Total Size:  %s\n", formatBytes(stats.TotalSize))
	fmt.Printf("│ Total Files: %d\n", stats.TotalFileCount)
	fmt.Println("└─────────────────────────")

	return nil
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
