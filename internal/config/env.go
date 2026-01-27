package config

import (
	"fmt"
	"strings"
)

// EnvExporter handles environment variable export for restic
type EnvExporter struct {
	Repository         string
	Password           string
	AWSAccessKeyID     string
	AWSSecretAccessKey string
}

// ExportFormat represents the shell format for export
type ExportFormat string

const (
	FormatBash       ExportFormat = "bash"
	FormatFish       ExportFormat = "fish"
	FormatPowershell ExportFormat = "powershell"
)

// NewEnvExporter creates an EnvExporter from a config and backend name
func NewEnvExporter(cfg *Config, backendName string) (*EnvExporter, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	exporter := &EnvExporter{}

	if backendName == "" || backendName == "primary" {
		// Use primary backend
		exporter.Repository = cfg.Repository
		exporter.Password = cfg.GetPassword()
		exporter.AWSAccessKeyID = cfg.GetAWSAccessKeyID()
		exporter.AWSSecretAccessKey = cfg.GetAWSSecretAccessKey()
	} else {
		// Use named backend
		backend, exists := cfg.Backends[backendName]
		if !exists {
			return nil, fmt.Errorf("backend '%s' not found in configuration", backendName)
		}
		exporter.Repository = backend.Repository
		exporter.Password = backend.Password
		exporter.AWSAccessKeyID = backend.AWSAccessKeyID
		exporter.AWSSecretAccessKey = backend.AWSSecretAccessKey
	}

	// Validate required fields
	if exporter.Repository == "" {
		return nil, fmt.Errorf("repository not configured for backend '%s'", backendName)
	}
	if exporter.Password == "" {
		return nil, fmt.Errorf("password not configured for backend '%s'", backendName)
	}

	return exporter, nil
}

// NewEnvExporterFromActiveBackend creates an EnvExporter using the active backend
func NewEnvExporterFromActiveBackend(cfg *Config) (*EnvExporter, error) {
	activeBackend, err := GetActiveBackend()
	if err != nil {
		activeBackend = ""
	}

	return NewEnvExporter(cfg, activeBackend)
}

// Export generates the export commands for the specified format
func (e *EnvExporter) Export(format ExportFormat) (string, error) {
	switch format {
	case FormatBash:
		return e.exportBash(), nil
	case FormatFish:
		return e.exportFish(), nil
	case FormatPowershell:
		return e.exportPowershell(), nil
	default:
		return "", fmt.Errorf("unsupported format: %s (supported: bash, fish, powershell)", format)
	}
}

// exportBash exports in bash/sh format
func (e *EnvExporter) exportBash() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("export RESTIC_REPOSITORY='%s'\n", escapeShell(e.Repository)))
	b.WriteString(fmt.Sprintf("export RESTIC_PASSWORD='%s'\n", escapeShell(e.Password)))

	if e.AWSAccessKeyID != "" {
		b.WriteString(fmt.Sprintf("export AWS_ACCESS_KEY_ID='%s'\n", escapeShell(e.AWSAccessKeyID)))
	}
	if e.AWSSecretAccessKey != "" {
		b.WriteString(fmt.Sprintf("export AWS_SECRET_ACCESS_KEY='%s'\n", escapeShell(e.AWSSecretAccessKey)))
	}

	b.WriteString("# Environment variables exported successfully\n")
	b.WriteString("# You can now use restic commands directly\n")

	return b.String()
}

// exportFish exports in fish shell format
func (e *EnvExporter) exportFish() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("set -x RESTIC_REPOSITORY '%s'\n", escapeShell(e.Repository)))
	b.WriteString(fmt.Sprintf("set -x RESTIC_PASSWORD '%s'\n", escapeShell(e.Password)))

	if e.AWSAccessKeyID != "" {
		b.WriteString(fmt.Sprintf("set -x AWS_ACCESS_KEY_ID '%s'\n", escapeShell(e.AWSAccessKeyID)))
	}
	if e.AWSSecretAccessKey != "" {
		b.WriteString(fmt.Sprintf("set -x AWS_SECRET_ACCESS_KEY '%s'\n", escapeShell(e.AWSSecretAccessKey)))
	}

	b.WriteString("# Environment variables exported successfully\n")
	b.WriteString("# You can now use restic commands directly\n")

	return b.String()
}

// exportPowershell exports in PowerShell format
func (e *EnvExporter) exportPowershell() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("$env:RESTIC_REPOSITORY = '%s'\n", escapePowershell(e.Repository)))
	b.WriteString(fmt.Sprintf("$env:RESTIC_PASSWORD = '%s'\n", escapePowershell(e.Password)))

	if e.AWSAccessKeyID != "" {
		b.WriteString(fmt.Sprintf("$env:AWS_ACCESS_KEY_ID = '%s'\n", escapePowershell(e.AWSAccessKeyID)))
	}
	if e.AWSSecretAccessKey != "" {
		b.WriteString(fmt.Sprintf("$env:AWS_SECRET_ACCESS_KEY = '%s'\n", escapePowershell(e.AWSSecretAccessKey)))
	}

	b.WriteString("# Environment variables exported successfully\n")
	b.WriteString("# You can now use restic commands directly\n")

	return b.String()
}

// escapeShell escapes single quotes for bash/fish/sh shells
func escapeShell(s string) string {
	return strings.ReplaceAll(s, "'", "'\\''")
}

// escapePowershell escapes single quotes for PowerShell
func escapePowershell(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
