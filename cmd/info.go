package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"resticm/internal/config"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display configuration summary",
	Long: `Display a summary of the current configuration.

Shows repository settings, backends, retention policy, notifications,
and other important configuration details.

The configuration is loaded following the priority order:
  1. --config flag (if specified)
  2. Active context (if set via 'resticm context use')
  3. Default paths: ./config.yaml, ~/.config/resticm/config.yaml, /etc/resticm/config.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInfo()
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

func runInfo() error {
	// Colors
	bold := color.New(color.Bold)
	cyan := color.New(color.FgCyan)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	gray := color.New(color.FgHiBlack)
	red := color.New(color.FgRed)

	// Load context to check if one is active
	ctx, _ := config.LoadContext()

	// Get config file path from the loaded config
	configPath := config.GetLoadedConfigPath()
	configSource := ""

	if cfgFile != "" {
		configSource = "flag --config"
	} else if ctx != nil && ctx.ConfigFile != "" {
		configSource = "context"
	} else {
		configSource = "default path"
	}

	// Fallback if path is empty
	if configPath == "" {
		configPath = "(not found)"
	}

	fmt.Println()
	bold.Println("╔══════════════════════════════════════════════════════════════════╗")
	bold.Println("║                      RESTICM CONFIGURATION                       ║")
	bold.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Config source section
	bold.Println("📁 Configuration Source")
	fmt.Println("────────────────────────────────────────────────────────────────────")
	if ctx != nil && ctx.ConfigFile != "" {
		_, _ = yellow.Print("  ⚡ Context active: ")
		fmt.Println("yes")
	}
	fmt.Print("  File:   ")
	_, _ = cyan.Println(configPath)
	fmt.Print("  Source: ")
	_, _ = gray.Println(configSource)
	fmt.Println()

	// Load config
	cfg := GetConfig()
	if cfg == nil {
		_, _ = red.Println("  ❌ Configuration not loaded or invalid")
		return fmt.Errorf("configuration not loaded")
	}

	// Primary Repository
	bold.Println("🗄️  Primary Repository")
	fmt.Println("────────────────────────────────────────────────────────────────────")
	fmt.Print("  Repository: ")
	cyan.Println(cfg.Repository)
	fmt.Print("  Password:   ")
	if cfg.Password != "" {
		_, _ = green.Println("configured")
	} else {
		_, _ = yellow.Println("not set (check RESTIC_PASSWORD env)")
	}

	// Check if S3
	if strings.HasPrefix(cfg.Repository, "s3:") {
		if cfg.AWSAccessKeyID != "" {
			fmt.Print("  AWS Keys:   ")
			_, _ = green.Println("configured")
		} else {
			fmt.Print("  AWS Keys:   ")
			_, _ = yellow.Println("not set (check environment)")
		}
	}
	fmt.Println()

	// Directories
	bold.Println("📂 Backup Directories")
	fmt.Println("────────────────────────────────────────────────────────────────────")
	if len(cfg.Directories) == 0 {
		_, _ = yellow.Println("  No directories configured")
	} else {
		for _, dir := range cfg.Directories {
			// Check if directory exists
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				_, _ = red.Printf("  • %s ", dir)
				_, _ = gray.Println("(not found)")
			} else {
				fmt.Printf("  • %s\n", dir)
			}
		}
	}
	fmt.Println()

	// Exclusions
	bold.Println("🚫 Exclusions")
	fmt.Println("────────────────────────────────────────────────────────────────────")
	if cfg.ExcludeFile != "" {
		fmt.Printf("  Exclude file: %s\n", cfg.ExcludeFile)
	}
	if len(cfg.ExcludePatterns) > 0 {
		fmt.Printf("  Patterns: %d configured\n", len(cfg.ExcludePatterns))
		// Show first 5 patterns
		maxShow := 5
		if len(cfg.ExcludePatterns) < maxShow {
			maxShow = len(cfg.ExcludePatterns)
		}
		for i := 0; i < maxShow; i++ {
			fmt.Printf("    • %s\n", cfg.ExcludePatterns[i])
		}
		if len(cfg.ExcludePatterns) > 5 {
			_, _ = gray.Printf("    ... and %d more\n", len(cfg.ExcludePatterns)-5)
		}
	} else if cfg.ExcludeFile == "" {
		_, _ = gray.Println("  No exclusions configured")
	}
	fmt.Println()

	// Retention Policy
	bold.Println("🕐 Retention Policy")
	fmt.Println("────────────────────────────────────────────────────────────────────")
	if cfg.Retention.KeepWithin != "" {
		fmt.Printf("  Keep within:  %s\n", cfg.Retention.KeepWithin)
	}
	if cfg.Retention.KeepHourly > 0 {
		fmt.Printf("  Keep hourly:  %d\n", cfg.Retention.KeepHourly)
	}
	if cfg.Retention.KeepDaily > 0 {
		fmt.Printf("  Keep daily:   %d\n", cfg.Retention.KeepDaily)
	}
	if cfg.Retention.KeepWeekly > 0 {
		fmt.Printf("  Keep weekly:  %d\n", cfg.Retention.KeepWeekly)
	}
	if cfg.Retention.KeepMonthly > 0 {
		fmt.Printf("  Keep monthly: %d\n", cfg.Retention.KeepMonthly)
	}
	if cfg.Retention.KeepYearly > 0 {
		fmt.Printf("  Keep yearly:  %d\n", cfg.Retention.KeepYearly)
	}
	fmt.Println()

	// Secondary Backends
	bold.Println("💾 Secondary Backends")
	fmt.Println("────────────────────────────────────────────────────────────────────")
	if len(cfg.Backends) == 0 {
		gray.Println("  No secondary backends configured")
	} else {
		for name, backend := range cfg.Backends {
			// Check if in copy_to_backends
			inCopyList := false
			for _, copyName := range cfg.CopyToBackends {
				if copyName == name {
					inCopyList = true
					break
				}
			}

			if inCopyList {
				green.Printf("  ✓ %s", name)
				gray.Println(" (auto-copy enabled)")
			} else {
				fmt.Printf("  • %s\n", name)
			}
			gray.Printf("    %s\n", backend.Repository)
		}
	}

	if len(cfg.CopyToBackends) > 0 {
		fmt.Println()
		fmt.Print("  Auto-copy to: ")
		cyan.Println(strings.Join(cfg.CopyToBackends, ", "))
	}
	fmt.Println()

	// Deep Check
	bold.Println("🔍 Deep Check")
	fmt.Println("────────────────────────────────────────────────────────────────────")
	if cfg.DeepCheckIntervalDays > 0 {
		fmt.Printf("  Interval: every %d days\n", cfg.DeepCheckIntervalDays)
	} else {
		gray.Println("  Disabled (interval = 0)")
	}
	fmt.Println()

	// Tags
	if len(cfg.DefaultTags) > 0 {
		bold.Println("🏷️  Default Tags")
		fmt.Println("────────────────────────────────────────────────────────────────────")
		fmt.Printf("  %s\n", strings.Join(cfg.DefaultTags, ", "))
		fmt.Println()
	}

	// Hooks
	bold.Println("🪝 Hooks")
	fmt.Println("────────────────────────────────────────────────────────────────────")
	hasHooks := false
	if cfg.Hooks.PreBackup != "" {
		fmt.Printf("  Pre-backup:  %s\n", cfg.Hooks.PreBackup)
		hasHooks = true
	}
	if cfg.Hooks.PostBackup != "" {
		fmt.Printf("  Post-backup: %s\n", cfg.Hooks.PostBackup)
		hasHooks = true
	}
	if cfg.Hooks.OnError != "" {
		fmt.Printf("  On error:    %s\n", cfg.Hooks.OnError)
		hasHooks = true
	}
	if cfg.Hooks.OnSuccess != "" {
		fmt.Printf("  On success:  %s\n", cfg.Hooks.OnSuccess)
		hasHooks = true
	}
	if !hasHooks {
		gray.Println("  No hooks configured")
	}
	fmt.Println()

	// Notifications
	bold.Println("🔔 Notifications")
	fmt.Println("────────────────────────────────────────────────────────────────────")
	if cfg.Notifications.Enabled {
		green.Println("  Status: enabled")
		fmt.Print("  On success: ")
		if cfg.Notifications.NotifyOnSuccess {
			green.Println("yes")
		} else {
			gray.Println("no")
		}
		fmt.Print("  On error:   ")
		if cfg.Notifications.NotifyOnError {
			green.Println("yes")
		} else {
			gray.Println("no")
		}

		if len(cfg.Notifications.Providers) > 0 {
			fmt.Println("  Providers:")
			for _, p := range cfg.Notifications.Providers {
				fmt.Printf("    • %s\n", p.Type)
			}
		}
	} else {
		gray.Println("  Status: disabled")
	}
	fmt.Println()

	// Logging
	bold.Println("📝 Logging")
	fmt.Println("────────────────────────────────────────────────────────────────────")
	if cfg.Logging.File != "" {
		fmt.Printf("  File:     %s\n", cfg.Logging.File)
		fmt.Printf("  Level:    %s\n", cfg.Logging.Level)
		fmt.Printf("  Max size: %d MB\n", cfg.Logging.MaxSizeMB)
		fmt.Printf("  Max files: %d\n", cfg.Logging.MaxFiles)
		fmt.Print("  Console:  ")
		if cfg.Logging.Console {
			green.Println("yes")
		} else {
			gray.Println("no")
		}
		fmt.Print("  JSON:     ")
		if cfg.Logging.JSON {
			green.Println("yes")
		} else {
			gray.Println("no")
		}
	} else {
		gray.Println("  No log file configured")
	}
	fmt.Println()

	bold.Println("════════════════════════════════════════════════════════════════════")
	fmt.Println()

	return nil
}
