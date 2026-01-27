package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"resticm/internal/config"
)

var backendCmd = &cobra.Command{
	Use:   "backend",
	Short: "Manage backend selection",
	Long: `Manage which backend to use for restic operations.

The primary backend uses the main repository configuration.
Secondary backends are defined in the 'backends' section of the config file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return showBackend()
	},
}

var backendUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Set the active backend",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		backendName := args[0]

		// Validate backend exists if not "primary"
		if backendName != "primary" {
			cfg := GetConfig()
			if cfg == nil {
				var err error
				cfg, err = config.Load(cfgFile)
				if err != nil {
					return fmt.Errorf("failed to load config: %w", err)
				}
			}
			if _, exists := cfg.Backends[backendName]; !exists {
				return fmt.Errorf("backend '%s' not found in configuration", backendName)
			}
		}

		// Set to empty string for primary
		if backendName == "primary" {
			backendName = ""
		}

		if err := config.SetActiveBackend(backendName); err != nil {
			return err
		}

		if backendName == "" {
			PrintSuccess("Switched to primary backend")
		} else {
			PrintSuccess("Switched to backend: %s", backendName)
		}

		return nil
	},
}

var backendListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available backends",
	RunE: func(cmd *cobra.Command, args []string) error {
		return listBackends()
	},
}

func init() {
	rootCmd.AddCommand(backendCmd)
	backendCmd.AddCommand(backendUseCmd)
	backendCmd.AddCommand(backendListCmd)
}

func showBackend() error {
	activeBackend, err := config.GetActiveBackend()
	if err != nil {
		activeBackend = ""
	}
	if activeBackend == "" {
		activeBackend = "primary"
	}

	bold := color.New(color.Bold)
	cyan := color.New(color.FgCyan)

	fmt.Println()
	_, _ = bold.Println("🗄️  Current Backend")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  Active:  ")
	_, _ = cyan.Println(activeBackend)

	cfg := GetConfig()
	if cfg != nil {
		if activeBackend == "primary" {
			fmt.Printf("  Repository:  ")
			_, _ = cyan.Println(cfg.Repository)
		} else if backend, ok := cfg.Backends[activeBackend]; ok {
			fmt.Printf("  Repository:  ")
			_, _ = cyan.Println(backend.Repository)
		}
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	return nil
}

func listBackends() error {
	cfg := GetConfig()
	if cfg == nil {
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	activeBackend, _ := config.GetActiveBackend()
	if activeBackend == "" {
		activeBackend = "primary"
	}

	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)
	cyan := color.New(color.FgCyan)

	fmt.Println()
	_, _ = bold.Println("🗄️  Configured Backends")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Primary backend
	if activeBackend == "primary" {
		_, _ = green.Printf("→ primary")
		fmt.Println(" (active)")
	} else {
		fmt.Println("  primary")
	}
	fmt.Printf("     Repository: ")
	cyan.Println(cfg.Repository)
	fmt.Println()

	// Secondary backends
	if len(cfg.Backends) == 0 {
		fmt.Println("  No secondary backends configured")
	} else {
		for name, backend := range cfg.Backends {
			if activeBackend == name {
				_, _ = green.Printf("→ %s", name)
				fmt.Println(" (active)")
			} else {
				fmt.Printf("  %s\n", name)
			}
			fmt.Printf("     Repository: ")
			_, _ = cyan.Println(backend.Repository)
			fmt.Println()
		}
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	return nil
}
