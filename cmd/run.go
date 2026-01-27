package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"resticm/internal/config"
	"resticm/internal/restic"
)

var runCmd = &cobra.Command{
	Use:   "run [restic-command] [args...]",
	Short: "Run any restic command with current context",
	Long: `Run any restic command with the credentials from the current context.

Examples:
  resticm run snapshots
  resticm run list locks
  resticm run stats
  resticm run check
  resticm -c config.yaml run snapshots`,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Parse out global flags manually since DisableFlagParsing is true
		var configFile string
		var resticArgs []string

		for i := 0; i < len(args); i++ {
			arg := args[i]
			if arg == "-c" || arg == "--config" {
				if i+1 < len(args) {
					configFile = args[i+1]
					i++ // skip next arg
					continue
				}
			} else if strings.HasPrefix(arg, "-c=") {
				configFile = strings.TrimPrefix(arg, "-c=")
				continue
			} else if strings.HasPrefix(arg, "--config=") {
				configFile = strings.TrimPrefix(arg, "--config=")
				continue
			} else if arg == "-v" || arg == "--verbose" {
				verbose = true
				continue
			} else if arg == "-n" || arg == "--dry-run" {
				dryRun = true
				continue
			} else if arg == "-h" || arg == "--help" {
				_ = cmd.Help()
				return nil
			}
			resticArgs = append(resticArgs, arg)
		}

		// Use global cfgFile if not specified in args
		if configFile == "" {
			configFile = cfgFile
		}

		if len(resticArgs) == 0 {
			return fmt.Errorf("no restic command specified")
		}

		return runResticCommand(configFile, resticArgs)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func runResticCommand(configFile string, args []string) error {
	// Load config
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Get active backend
	activeBackend, _ := config.GetActiveBackend()

	var repo, password string
	var awsKey, awsSecret string

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

	// Create executor
	executor := restic.NewExecutor(repo, password)
	executor.SetAWSCredentials(awsKey, awsSecret)

	return executor.RunWithStreaming(args...)
}
