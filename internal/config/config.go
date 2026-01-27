// Package config handles configuration loading and validation for resticm
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
	// Primary repository
	Repository string `yaml:"repository"`
	Password   string `yaml:"password"`

	// AWS credentials for S3 backends
	AWSAccessKeyID     string `yaml:"aws_access_key_id"`
	AWSSecretAccessKey string `yaml:"aws_secret_access_key"`

	// Directories to backup
	Directories []string `yaml:"directories"`

	// Exclude patterns
	ExcludePatterns []string `yaml:"exclude_patterns"`
	ExcludeFile     string   `yaml:"exclude_file"`

	// Retention policy
	Retention RetentionConfig `yaml:"retention"`

	// Deep check interval in days
	DeepCheckIntervalDays int `yaml:"deep_check_interval_days"`

	// Default tags for backups
	DefaultTags []string `yaml:"default_tags"`

	// Secondary backends
	Backends map[string]Backend `yaml:"backends"`

	// Backends to copy to after backup
	CopyToBackends []string `yaml:"copy_to_backends"`

	// Hooks configuration
	Hooks HookConfig `yaml:"hooks"`

	// Notifications configuration
	Notifications NotificationConfig `yaml:"notifications"`

	// Logging configuration
	Logging LoggingConfig `yaml:"logging"`
}

// RetentionConfig defines the retention policy
type RetentionConfig struct {
	KeepWithin  string `yaml:"keep_within"`
	KeepHourly  int    `yaml:"keep_hourly"`
	KeepDaily   int    `yaml:"keep_daily"`
	KeepWeekly  int    `yaml:"keep_weekly"`
	KeepMonthly int    `yaml:"keep_monthly"`
	KeepYearly  int    `yaml:"keep_yearly"`
}

// Backend represents a secondary backend configuration
type Backend struct {
	Repository         string `yaml:"repository"`
	Password           string `yaml:"password"`
	AWSAccessKeyID     string `yaml:"aws_access_key_id"`
	AWSSecretAccessKey string `yaml:"aws_secret_access_key"`
}

// HookConfig defines hook scripts
type HookConfig struct {
	PreBackup  string `yaml:"pre_backup"`
	PostBackup string `yaml:"post_backup"`
	OnError    string `yaml:"on_error"`
	OnSuccess  string `yaml:"on_success"`
}

// NotificationConfig defines notification settings
type NotificationConfig struct {
	Enabled         bool             `yaml:"enabled"`
	NotifyOnSuccess bool             `yaml:"notify_on_success"`
	NotifyOnError   bool             `yaml:"notify_on_error"`
	Providers       []ProviderConfig `yaml:"providers"`
}

// ProviderConfig defines a notification provider
type ProviderConfig struct {
	Type    string            `yaml:"type"`
	URL     string            `yaml:"url"`
	Token   string            `yaml:"token"`
	Channel string            `yaml:"channel"`
	Options map[string]string `yaml:"options"`
}

// LoggingConfig defines logging settings
type LoggingConfig struct {
	File      string `yaml:"file"`
	MaxSizeMB int    `yaml:"max_size_mb"`
	MaxFiles  int    `yaml:"max_files"`
	Level     string `yaml:"level"`
	Console   bool   `yaml:"console"`
	JSON      bool   `yaml:"json"`
}

// DefaultConfig returns a config with default values
func DefaultConfig() *Config {
	return &Config{
		Retention: RetentionConfig{
			KeepWithin:  "7d",
			KeepHourly:  24,
			KeepDaily:   7,
			KeepWeekly:  4,
			KeepMonthly: 12,
			KeepYearly:  5,
		},
		DeepCheckIntervalDays: 30,
		Logging: LoggingConfig{
			File:      "/var/log/resticm/resticm.log",
			MaxSizeMB: 10,
			MaxFiles:  5,
			Level:     "info",
			Console:   true,
		},
	}
}

// loadedConfigPath stores the path of the loaded configuration file
var loadedConfigPath string

// GetLoadedConfigPath returns the path of the currently loaded configuration file
func GetLoadedConfigPath() string {
	return loadedConfigPath
}

// Load loads configuration from a file
func Load(path string) (*Config, error) {
	// Resolve config path
	configPath, err := resolveConfigPath(path)
	if err != nil {
		return nil, err
	}

	// Store the resolved path
	loadedConfigPath = configPath

	// Validate file permissions
	if err := ValidateFilePermissions(configPath); err != nil {
		return nil, err
	}

	// Read file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// alternateConfigPaths stores paths of config files that exist but weren't selected
var alternateConfigPaths []string

// GetAlternateConfigPaths returns paths of config files that exist but weren't selected
func GetAlternateConfigPaths() []string {
	return alternateConfigPaths
}

// resolveConfigPath finds the configuration file
func resolveConfigPath(path string) (string, error) {
	// Reset alternate paths
	alternateConfigPaths = nil

	if path != "" {
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("config file not found: %s", path)
		}
		return path, nil
	}

	// Check context for config file
	ctx, err := LoadContext()
	if err == nil && ctx.ConfigFile != "" {
		if _, err := os.Stat(ctx.ConfigFile); err == nil {
			return ctx.ConfigFile, nil
		}
	}

	// Build list of default locations based on whether we're running as root
	var locations []string

	if IsRoot() {
		// Running as root/sudo: prefer system config first
		locations = append(locations, "/etc/resticm/config.yaml")

		// If running via sudo, also check the original user's config as fallback
		if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
			sudoUserHome := filepath.Join("/home", sudoUser)
			locations = append(locations, filepath.Join(sudoUserHome, ".config", "resticm", "config.yaml"))
		}
	} else {
		// Running as normal user: prefer user config first
		if home := os.Getenv("HOME"); home != "" {
			locations = append(locations, filepath.Join(home, ".config", "resticm", "config.yaml"))
		}
		// System-wide config as fallback
		locations = append(locations, "/etc/resticm/config.yaml")
	}

	// Find the first existing config and track alternatives
	var selectedPath string
	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			if selectedPath == "" {
				selectedPath = loc
			} else {
				// This config exists but wasn't selected
				alternateConfigPaths = append(alternateConfigPaths, loc)
			}
		}
	}

	if selectedPath != "" {
		return selectedPath, nil
	}

	return "", fmt.Errorf("no configuration file found. Create one at ~/.config/resticm/config.yaml or /etc/resticm/config.yaml")
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Repository == "" {
		return fmt.Errorf("repository is required")
	}

	if c.Password == "" && os.Getenv("RESTIC_PASSWORD") == "" {
		return fmt.Errorf("password is required (set in config or RESTIC_PASSWORD env)")
	}

	if len(c.Directories) == 0 {
		return fmt.Errorf("at least one directory to backup is required")
	}

	return nil
}

// GetPassword returns the password, checking env var first
func (c *Config) GetPassword() string {
	if env := os.Getenv("RESTIC_PASSWORD"); env != "" {
		return env
	}
	return c.Password
}

// GetAWSAccessKeyID returns the AWS access key, checking env var first
func (c *Config) GetAWSAccessKeyID() string {
	if env := os.Getenv("AWS_ACCESS_KEY_ID"); env != "" {
		return env
	}
	return c.AWSAccessKeyID
}

// GetAWSSecretAccessKey returns the AWS secret key, checking env var first
func (c *Config) GetAWSSecretAccessKey() string {
	if env := os.Getenv("AWS_SECRET_ACCESS_KEY"); env != "" {
		return env
	}
	return c.AWSSecretAccessKey
}

// CreateExampleConfig creates an example configuration file
func CreateExampleConfig(path string) error {
	example := `# Resticm Configuration
repository: "s3:s3.amazonaws.com/my-bucket/restic"
password: "your-secure-password"

directories:
  - /etc
  - /root
  - /home

exclude_patterns:
  - "*.tmp"
  - "*.log"
  - "**/node_modules/**"

retention:
  keep_within: "7d"
  keep_hourly: 24
  keep_daily: 7
  keep_weekly: 4
  keep_monthly: 12
  keep_yearly: 5
`

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write with secure permissions
	if err := os.WriteFile(path, []byte(example), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ExpandPath expands ~ to home directory
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[1:])
	}
	return path
}
