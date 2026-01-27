package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"resticm/internal/config"
)

var (
	envBackend string
	envFormat  string
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Export restic environment variables",
	Long: `Export restic environment variables for use with the restic CLI directly.

By default, uses the active backend (set via 'resticm backend use').
Use --backend to temporarily use a different backend.

Examples:
  # Use active backend
  eval $(resticm env)
  
  # Use specific backend
  eval $(resticm env --backend s3-backup)
  
  # Export to file
  resticm env > ~/.resticm.env
  source ~/.resticm.env
  
  # Use with fish shell
  resticm env --format fish | source`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEnv()
	},
}

func init() {
	rootCmd.AddCommand(envCmd)
	envCmd.Flags().StringVarP(&envBackend, "backend", "b", "", "Backend to export (default: active backend)")
	envCmd.Flags().StringVar(&envFormat, "format", "bash", "Output format: bash, fish, powershell")
}

func runEnv() error {
	cfg := GetConfig()
	if cfg == nil {
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Create exporter
	var exporter *config.EnvExporter
	var err error

	if envBackend != "" {
		// Use specified backend
		exporter, err = config.NewEnvExporter(cfg, envBackend)
	} else {
		// Use active backend
		exporter, err = config.NewEnvExporterFromActiveBackend(cfg)
	}

	if err != nil {
		return err
	}

	// Convert format string to ExportFormat
	var format config.ExportFormat
	switch envFormat {
	case "bash", "sh":
		format = config.FormatBash
	case "fish":
		format = config.FormatFish
	case "powershell", "pwsh":
		format = config.FormatPowershell
	default:
		return fmt.Errorf("unsupported format: %s (supported: bash, fish, powershell)", envFormat)
	}

	// Export and print
	output, err := exporter.Export(format)
	if err != nil {
		return err
	}

	fmt.Print(output)
	return nil
}
