package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"resticm/internal/config"
	"resticm/internal/hooks"
	"resticm/internal/logging"
	"resticm/internal/notify"
	"resticm/internal/restic"
	"resticm/internal/security"
)

// Version information (set by ldflags)
var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

// Global flags
var (
	cfgFile    string
	verbose    bool
	dryRun     bool
	jsonOutput bool
)

// Global config instance
var cfg *config.Config

// Global logger instance
var logger *logging.Logger

// Color outputs
var (
	colorError   = color.New(color.FgRed, color.Bold)
	colorSuccess = color.New(color.FgGreen, color.Bold)
	colorWarning = color.New(color.FgYellow)
	colorInfo    = color.New(color.FgCyan)
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "resticm",
	Short: "Restic Manager - Automated backup solution",
	Long: `Resticm is a modern CLI wrapper for restic that provides:
  - Multi-environment support (contexts)
  - Multi-backend replication
  - Automated backup workflows
  - Webhook notifications
  - Pre/post backup hooks

Default behavior (no subcommand): runs backup + forget + copy`,
	SilenceUsage:  true, // Don't show usage on error
	SilenceErrors: true, // We handle errors ourselves
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for commands that don't need it or handle it themselves
		if cmd.Name() == "version" || cmd.Name() == "help" || cmd.Name() == "completion" || cmd.Name() == "run" {
			return nil
		}
		// Skip for context commands that manage config themselves
		if cmd.Parent() != nil && cmd.Parent().Name() == "context" {
			return nil
		}

		// Load configuration
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Warn if running backup commands without root privileges
		if needsRootForFullAccess(cmd) && !config.IsRoot() {
			colorWarning.Fprintln(os.Stderr, "⚠️  Running without root privileges.")
			colorWarning.Fprintln(os.Stderr, "   Some files may not be accessible for backup (e.g., /etc, other users' files).")
			colorWarning.Fprintln(os.Stderr, "   For a complete system backup, run with: sudo resticm")
			fmt.Fprintln(os.Stderr)
		}

		// Warn if multiple config files exist
		if altPaths := config.GetAlternateConfigPaths(); len(altPaths) > 0 {
			colorWarning.Fprintf(os.Stderr, "⚠️  Multiple config files found. Using: %s\n", config.GetLoadedConfigPath())
			for _, alt := range altPaths {
				colorWarning.Fprintf(os.Stderr, "   Also found: %s\n", alt)
			}
			colorWarning.Fprintln(os.Stderr, "   Use --config to specify explicitly.")
			fmt.Fprintln(os.Stderr)
		}

		// Initialize logger (adapt path if not root)
		logFile := cfg.Logging.File
		if !config.IsRoot() && strings.HasPrefix(logFile, "/var/log/") {
			// Use user's home directory for logs if not root
			home, _ := os.UserHomeDir()
			logFile = filepath.Join(home, ".local", "share", "resticm", "resticm.log")
		}

		logger, err = logging.Configure(logging.Config{
			File:      logFile,
			MaxSizeMB: cfg.Logging.MaxSizeMB,
			MaxFiles:  cfg.Logging.MaxFiles,
			Level:     cfg.Logging.Level,
			Console:   cfg.Logging.Console,
			JSON:      cfg.Logging.JSON,
		})
		if err != nil {
			// Log to stderr but don't fail
			fmt.Fprintf(os.Stderr, "Warning: failed to configure logging: %v\n", err)
			logger = logging.NewLogger(logging.INFO)
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default mode: backup + forget + copy
		return runDefaultWorkflow(cmd)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		// Print error in red
		colorError.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return nil
}

func init() {
	// Disable usage display on all errors
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		cmd.SilenceUsage = true
		return err
	})

	// Global persistent flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default: ~/.config/resticm/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "n", false, "perform a trial run with no changes made")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output in JSON format")

	// Root command flags (for default mode)
	rootCmd.Flags().BoolP("prune", "p", false, "also run prune after forget")
	rootCmd.Flags().Bool("check", false, "also run repository check")
	rootCmd.Flags().Bool("deep", false, "run deep check (implies --check)")
	rootCmd.Flags().Bool("no-backup", false, "skip backup operation")
	rootCmd.Flags().Bool("no-forget", false, "skip forget operation")
	rootCmd.Flags().Bool("no-prune", false, "skip prune operation")
	rootCmd.Flags().Bool("no-check", false, "skip check operation")
	rootCmd.Flags().Bool("no-copy", false, "skip copy to secondary backends")
	rootCmd.Flags().StringP("tag", "t", "", "add extra tag to backup")
	rootCmd.Flags().Bool("notify-success", false, "send notification on success")
	rootCmd.Flags().Bool("copy-all", false, "copy snapshots from all hosts")
}

// GetConfig returns the global configuration
func GetConfig() *config.Config {
	return cfg
}

// IsVerbose returns true if verbose mode is enabled
func IsVerbose() bool {
	return verbose
}

// IsDryRun returns true if dry-run mode is enabled
func IsDryRun() bool {
	return dryRun
}

// IsJSONOutput returns true if JSON output mode is enabled
func IsJSONOutput() bool {
	return jsonOutput
}

// GetLogger returns the global logger instance
func GetLogger() *logging.Logger {
	return logger
}

// PrintError prints an error message in red and logs it
func PrintError(format string, a ...interface{}) {
	colorError.Fprintf(os.Stderr, "❌ "+format+"\n", a...)
	if logger != nil {
		logger.Error(format, a...)
	}
}

// PrintSuccess prints a success message in green and logs it
func PrintSuccess(format string, a ...interface{}) {
	colorSuccess.Printf("✅ "+format+"\n", a...)
	if logger != nil {
		logger.Info(format, a...)
	}
}

// PrintWarning prints a warning message in yellow and logs it
func PrintWarning(format string, a ...interface{}) {
	colorWarning.Printf("⚠️  "+format+"\n", a...)
	if logger != nil {
		logger.Warn(format, a...)
	}
}

// PrintInfo prints an info message in cyan and logs it
func PrintInfo(format string, a ...interface{}) {
	colorInfo.Printf("ℹ️  "+format+"\n", a...)
	if logger != nil {
		logger.Info(format, a...)
	}
}

// PrintVerbose prints a message only if verbose mode is enabled
func PrintVerbose(format string, a ...interface{}) {
	if verbose {
		fmt.Printf(format+"\n", a...)
	}
}

// runDefaultWorkflow executes the default workflow: backup + forget + copy
func runDefaultWorkflow(cmd *cobra.Command) error {
	cfg := GetConfig()
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	// Log start
	if logger != nil {
		logger.Info("Starting default workflow")
	}

	// Parse flags
	noBackup, _ := cmd.Flags().GetBool("no-backup")
	noForget, _ := cmd.Flags().GetBool("no-forget")
	noCopy, _ := cmd.Flags().GetBool("no-copy")
	doPrune, _ := cmd.Flags().GetBool("prune")
	noPrune, _ := cmd.Flags().GetBool("no-prune")
	doCheck, _ := cmd.Flags().GetBool("check")
	noCheck, _ := cmd.Flags().GetBool("no-check")
	deep, _ := cmd.Flags().GetBool("deep")
	extraTag, _ := cmd.Flags().GetString("tag")
	copyAll, _ := cmd.Flags().GetBool("copy-all")
	notifySuccess, _ := cmd.Flags().GetBool("notify-success")

	// Setup notifier
	notifier := notify.NewNotifier(notify.Config{
		Enabled:         cfg.Notifications.Enabled,
		NotifyOnSuccess: cfg.Notifications.NotifyOnSuccess || notifySuccess,
		NotifyOnError:   cfg.Notifications.NotifyOnError,
		Providers:       convertProviders(cfg.Notifications.Providers),
	})

	// Acquire lock
	lock := security.NewLock("")
	if err := lock.Acquire(); err != nil {
		return err
	}
	defer func() { _ = lock.Release() }()

	// Setup executor for primary repository
	repo := cfg.Repository
	password := cfg.GetPassword()
	awsKey := cfg.GetAWSAccessKeyID()
	awsSecret := cfg.GetAWSSecretAccessKey()

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

	hostname, _ := os.Hostname()
	var errors []error
	separator := strings.Repeat("━", 50)

	// Setup hooks
	hookRunner := hooks.NewRunner()
	hookRunner.PreBackup = cfg.Hooks.PreBackup
	hookRunner.PostBackup = cfg.Hooks.PostBackup
	hookRunner.OnError = cfg.Hooks.OnError
	hookRunner.OnSuccess = cfg.Hooks.OnSuccess
	hookRunner.DryRun = IsDryRun()
	hookRunner.Verbose = IsVerbose()

	// Banner
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println(" RESTICM DEFAULT WORKFLOW")
	fmt.Println("═══════════════════════════════════════════════════")

	// 1. BACKUP
	if !noBackup {
		fmt.Println("\n" + separator)
		fmt.Println("📦 BACKUP")
		fmt.Println(separator)

		// Run pre-backup hook
		if err := hookRunner.RunPreBackup(); err != nil {
			PrintError("Pre-backup hook failed: %v", err)
			errors = append(errors, err)
			_ = hookRunner.RunOnError(err)
		} else {
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
				if logger != nil {
					logger.Error("Backup failed: %v", err)
				}
				errors = append(errors, err)
				_ = hookRunner.RunPostBackup(false, err)
				_ = hookRunner.RunOnError(err)
			} else {
				PrintSuccess("Backup completed")
				if logger != nil {
					logger.Info("Backup completed successfully")
				}
				_ = hookRunner.RunPostBackup(true, nil)
			}
		}
	}

	// 2. FORGET
	if !noForget {
		fmt.Println("\n" + separator)
		fmt.Println("🗑️  FORGET")
		fmt.Println(separator)

		forgetOpts := restic.ForgetOptions{
			KeepWithin:  cfg.Retention.KeepWithin,
			KeepHourly:  cfg.Retention.KeepHourly,
			KeepDaily:   cfg.Retention.KeepDaily,
			KeepWeekly:  cfg.Retention.KeepWeekly,
			KeepMonthly: cfg.Retention.KeepMonthly,
			KeepYearly:  cfg.Retention.KeepYearly,
			Hostname:    hostname,
		}

		if err := executor.Forget(forgetOpts); err != nil {
			PrintError("Forget failed: %v", err)
			if logger != nil {
				logger.Error("Forget failed: %v", err)
			}
			errors = append(errors, err)
		} else {
			PrintSuccess("Forget completed")
			if logger != nil {
				logger.Info("Forget completed successfully")
			}
		}
	}

	// 3. PRUNE (optional)
	if doPrune && !noPrune {
		fmt.Println("\n" + separator)
		fmt.Println("🧹 PRUNE")
		fmt.Println(separator)

		if err := executor.Prune(); err != nil {
			PrintError("Prune failed: %v", err)
			if logger != nil {
				logger.Error("Prune failed: %v", err)
			}
			errors = append(errors, err)
		} else {
			PrintSuccess("Prune completed")
			if logger != nil {
				logger.Info("Prune completed successfully")
			}
		}
	}

	// 4. CHECK (optional)
	if (doCheck || deep) && !noCheck {
		fmt.Println("\n" + separator)
		fmt.Println("🔍 CHECK")
		fmt.Println(separator)

		checkOpts := restic.CheckOptions{ReadData: deep}
		if err := executor.Check(checkOpts); err != nil {
			PrintError("Check failed: %v", err)
			if logger != nil {
				logger.Error("Check failed: %v", err)
			}
			errors = append(errors, err)
		} else {
			PrintSuccess("Check passed")
			if logger != nil {
				logger.Info("Check completed successfully")
			}
		}
	}

	// 5. COPY & SYNC
	if !noCopy && len(cfg.CopyToBackends) > 0 {
		fmt.Println("\n" + separator)
		fmt.Println("📤 COPY & SYNC BACKENDS")
		fmt.Println(separator)

		copyHostname := hostname
		if copyAll {
			copyHostname = ""
		}

		for _, backendName := range cfg.CopyToBackends {
			backend, ok := cfg.Backends[backendName]
			if !ok {
				PrintWarning("Backend '%s' not found, skipping", backendName)
				if logger != nil {
					logger.Warn("Backend '%s' not found, skipping", backendName)
				}
				continue
			}

			fmt.Printf("\n  ┌─ Backend: %s\n", backendName)

			// 5a. COPY
			fmt.Println("  │ 📦 Copying snapshots...")
			if logger != nil {
				logger.Info("Copying to backend: %s", backendName)
			}

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
				if logger != nil {
					logger.Error("Copy to %s failed: %v", backendName, err)
				}
				errors = append(errors, err)
				fmt.Println("  └─ ❌ Skipping maintenance due to copy failure")
				continue
			}
			PrintSuccess("Copy to %s completed", backendName)

			// 5b. FORGET on this backend (same retention policy)
			fmt.Println("  │ 🗑️  Applying retention policy...")
			forgetOpts := restic.ForgetOptions{
				KeepWithin:  cfg.Retention.KeepWithin,
				KeepHourly:  cfg.Retention.KeepHourly,
				KeepDaily:   cfg.Retention.KeepDaily,
				KeepWeekly:  cfg.Retention.KeepWeekly,
				KeepMonthly: cfg.Retention.KeepMonthly,
				KeepYearly:  cfg.Retention.KeepYearly,
				Hostname:    hostname,
			}
			if err := destExecutor.Forget(forgetOpts); err != nil {
				PrintError("Forget on %s failed: %v", backendName, err)
				if logger != nil {
					logger.Error("Forget on %s failed: %v", backendName, err)
				}
				errors = append(errors, err)
			} else {
				PrintSuccess("Forget on %s completed", backendName)
			}

			// 5c. PRUNE on this backend (if requested)
			if doPrune && !noPrune {
				fmt.Println("  │ 🧹 Pruning unused data...")
				if err := destExecutor.Prune(); err != nil {
					PrintError("Prune on %s failed: %v", backendName, err)
					if logger != nil {
						logger.Error("Prune on %s failed: %v", backendName, err)
					}
					errors = append(errors, err)
				} else {
					PrintSuccess("Prune on %s completed", backendName)
				}
			}

			// 5d. CHECK on this backend (if requested)
			if (doCheck || deep) && !noCheck {
				fmt.Println("  │ 🔍 Checking integrity...")
				checkOpts := restic.CheckOptions{ReadData: deep}
				if err := destExecutor.Check(checkOpts); err != nil {
					PrintError("Check on %s failed: %v", backendName, err)
					if logger != nil {
						logger.Error("Check on %s failed: %v", backendName, err)
					}
					errors = append(errors, err)
				} else {
					PrintSuccess("Check on %s passed", backendName)
				}
			}

			fmt.Println("  └─ ✅ Backend synchronized")
			if logger != nil {
				logger.Info("Copy to %s completed successfully", backendName)
			}
		}
	}

	// Summary
	fmt.Println("\n" + strings.Repeat("═", 50))
	if len(errors) == 0 {
		PrintSuccess("All operations completed successfully!")
		if logger != nil {
			logger.Info("Workflow completed successfully")
		}
		// Send success notification
		notifier.NotifySuccess(
			"✅ Backup Successful",
			fmt.Sprintf("Resticm backup completed successfully on %s", hostname),
			map[string]string{
				"host":       hostname,
				"repository": cfg.Repository,
			},
		)
	} else {
		PrintError("%d operation(s) failed", len(errors))
		if logger != nil {
			logger.Error("Workflow completed with %d error(s)", len(errors))
		}
		// Send error notification
		var errMsgs []string
		for _, e := range errors {
			errMsgs = append(errMsgs, e.Error())
		}
		_ = notifier.NotifyError(
			"❌ Backup Failed",
			fmt.Sprintf("Resticm backup failed on %s with %d error(s)", hostname, len(errors)),
			nil,
			map[string]string{
				"host":       hostname,
				"repository": cfg.Repository,
				"errors":     strings.Join(errMsgs, "; "),
			},
		)
		return fmt.Errorf("%d operation(s) failed", len(errors))
	}

	return nil
}

// convertProviders converts config providers to notify providers
func convertProviders(cfgProviders []config.ProviderConfig) []notify.ProviderConfig {
	var providers []notify.ProviderConfig
	for _, p := range cfgProviders {
		providers = append(providers, notify.ProviderConfig{
			Type:    p.Type,
			URL:     p.URL,
			Options: p.Options,
		})
	}
	return providers
}

// GetNotifier creates a notifier from the current configuration
// This is used by all commands to send notifications
func GetNotifier(notifySuccess bool) *notify.Notifier {
	if cfg == nil {
		return notify.NewNotifier(notify.Config{Enabled: false})
	}
	return notify.NewNotifier(notify.Config{
		Enabled:         cfg.Notifications.Enabled,
		NotifyOnSuccess: cfg.Notifications.NotifyOnSuccess || notifySuccess,
		NotifyOnError:   cfg.Notifications.NotifyOnError,
		Providers:       convertProviders(cfg.Notifications.Providers),
	})
}

// needsRootForFullAccess returns true if the command benefits from root privileges
// for full system access (backup all files, write to /var/log, etc.)
func needsRootForFullAccess(cmd *cobra.Command) bool {
	// Commands that benefit from root access for full functionality
	rootBenefitCommands := map[string]bool{
		"backup":  true,
		"full":    true,
		"resticm": true, // Root command (default workflow)
	}

	// Check the command name
	if rootBenefitCommands[cmd.Name()] {
		return true
	}

	return false
}
