package cmd

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/spf13/cobra"

	"resticm/internal/config"
	"resticm/internal/restic"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize restic repositories",
	Long: `Initialize restic repositories.

When initializing backends configured in copy_to_backends, the chunker
parameters are automatically copied from the primary repository to ensure
optimal deduplication between repositories.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInit(cmd)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().String("backend", "", "Initialize a specific backend")
	initCmd.Flags().Bool("all", false, "Initialize all configured backends")
}

func runInit(cmd *cobra.Command) error {
	cfg := GetConfig()
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	backendName, _ := cmd.Flags().GetString("backend")
	initAll, _ := cmd.Flags().GetBool("all")

	if backendName != "" {
		return initBackend(cfg, backendName)
	}

	if initAll {
		return initAllBackends(cfg)
	}

	return initPrimary(cfg)
}

func initPrimary(cfg *config.Config) error {
	PrintInfo("Initializing primary repository...")
	fmt.Printf("  Repository: %s\n", cfg.Repository)

	executor := restic.NewExecutor(cfg.Repository, cfg.GetPassword())
	executor.SetAWSCredentials(cfg.GetAWSAccessKeyID(), cfg.GetAWSSecretAccessKey())
	executor.Verbose = IsVerbose()

	if executor.IsInitialized() {
		PrintSuccess("Repository already initialized")
		return nil
	}

	if err := executor.Init(); err != nil {
		PrintError("Failed to initialize repository: %v", err)
		return err
	}

	PrintSuccess("Primary repository initialized successfully")
	return nil
}

func initBackend(cfg *config.Config, name string) error {
	backend, ok := cfg.Backends[name]
	if !ok {
		return fmt.Errorf("backend '%s' not found in configuration", name)
	}

	PrintInfo("Initializing backend: %s", name)
	fmt.Printf("  Repository: %s\n", backend.Repository)

	password := backend.Password
	if password == "" {
		password = generatePassword()
		PrintWarning("No password configured for backend '%s'", name)
		fmt.Printf("\n  Generated password: %s\n\n", password)
	}

	executor := restic.NewExecutor(backend.Repository, password)
	executor.SetAWSCredentials(backend.AWSAccessKeyID, backend.AWSSecretAccessKey)
	executor.Verbose = IsVerbose()

	if executor.IsInitialized() {
		PrintSuccess("Backend '%s' already initialized", name)
		return nil
	}

	// Check if this backend is a copy target - if so, copy chunker params from primary
	isCopyTarget := false
	for _, copyBackend := range cfg.CopyToBackends {
		if copyBackend == name {
			isCopyTarget = true
			break
		}
	}

	if isCopyTarget && cfg.Repository != "" {
		PrintInfo("Copying chunker parameters from primary repository for optimal deduplication...")
		opts := restic.InitOptions{
			FromRepository:     cfg.Repository,
			FromPassword:       cfg.GetPassword(),
			CopyChunkerParams:  true,
			FromAWSAccessKeyID: cfg.GetAWSAccessKeyID(),
			FromAWSSecret:      cfg.GetAWSSecretAccessKey(),
		}
		if err := executor.InitWithOptions(opts); err != nil {
			PrintError("Failed to initialize backend '%s': %v", name, err)
			return err
		}
	} else {
		if err := executor.Init(); err != nil {
			PrintError("Failed to initialize backend '%s': %v", name, err)
			return err
		}
	}

	PrintSuccess("Backend '%s' initialized successfully", name)
	return nil
}

func initAllBackends(cfg *config.Config) error {
	if err := initPrimary(cfg); err != nil {
		return err
	}

	for name := range cfg.Backends {
		fmt.Println()
		if err := initBackend(cfg, name); err != nil {
			PrintWarning("Failed to initialize backend '%s': %v", name, err)
		}
	}

	return nil
}

func generatePassword() string {
	bytes := make([]byte, 32)
	_, _ = rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)[:32]
}
