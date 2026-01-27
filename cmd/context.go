package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"resticm/internal/config"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage configuration contexts",
	Long: `Manage which configuration file to use.

Contexts allow you to easily switch between different configurations
(e.g., production vs staging, different servers).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return showContext()
	},
}

var contextUseCmd = &cobra.Command{
	Use:   "use <config-file>",
	Short: "Set the active configuration file",
	Long: `Set the active configuration file.

Example:
  resticm context use ~/.config/resticm/production.yaml
  resticm context use /etc/resticm/server1.yaml`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := args[0]

		// Expand path
		configPath = config.ExpandPath(configPath)

		// Make absolute
		if !filepath.IsAbs(configPath) {
			cwd, _ := os.Getwd()
			configPath = filepath.Join(cwd, configPath)
		}

		// Check file exists
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return fmt.Errorf("configuration file not found: %s", configPath)
		}

		if err := config.SetConfigFile(configPath); err != nil {
			return err
		}

		PrintSuccess("Context set to: %s", configPath)
		return nil
	},
}

var contextListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available configuration files",
	RunE: func(cmd *cobra.Command, args []string) error {
		return listContexts()
	},
}

var contextResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset to default context",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.ResetContext(); err != nil {
			return err
		}
		PrintSuccess("Context reset to default")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(contextCmd)
	contextCmd.AddCommand(contextUseCmd)
	contextCmd.AddCommand(contextListCmd)
	contextCmd.AddCommand(contextResetCmd)
}

func showContext() error {
	ctx, err := config.LoadContext()
	if err != nil {
		return err
	}

	bold := color.New(color.Bold)
	cyan := color.New(color.FgCyan)

	fmt.Println()
	_, _ = bold.Println("🔧 Current Context")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if ctx.ConfigFile != "" {
		fmt.Printf("  Config:  ")
		cyan.Println(ctx.ConfigFile)
	} else {
		fmt.Println("  Config:  (default)")
	}

	if ctx.ActiveBackend != "" {
		fmt.Printf("  Backend: ")
		cyan.Println(ctx.ActiveBackend)
	} else {
		fmt.Println("  Backend: primary")
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	return nil
}

func listContexts() error {
	configs, err := config.ListConfigs()
	if err != nil {
		return err
	}

	ctx, _ := config.LoadContext()

	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)

	fmt.Println()
	bold.Println("📋 Available Configurations")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if len(configs) == 0 {
		fmt.Println("  No configuration files found")
		fmt.Println()
		fmt.Println("  Create one with:")
		fmt.Println("    resticm context use <path-to-config.yaml>")
	} else {
		for _, c := range configs {
			if ctx != nil && ctx.ConfigFile == c {
				green.Printf("  → %s", c)
				fmt.Println(" (active)")
			} else {
				fmt.Printf("    %s\n", c)
			}
		}
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	return nil
}
