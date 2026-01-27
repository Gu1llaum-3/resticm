package config

import (
	"strings"
	"testing"
)

func TestNewEnvExporter(t *testing.T) {
	cfg := &Config{
		Repository:         "/tmp/test-repo",
		Password:           "test-password",
		AWSAccessKeyID:     "AKIATEST123",
		AWSSecretAccessKey: "secret-key-123",
		Backends: map[string]Backend{
			"s3-backup": {
				Repository:         "s3:s3.amazonaws.com/my-bucket/backup",
				Password:           "s3-password",
				AWSAccessKeyID:     "AKIAS3TEST",
				AWSSecretAccessKey: "s3-secret-key",
			},
			"local-backup": {
				Repository: "/mnt/backup/restic",
				Password:   "local-password",
			},
		},
	}

	tests := []struct {
		name           string
		backend        string
		expectedRepo   string
		expectedPass   string
		expectedAWSKey string
		expectError    bool
	}{
		{
			name:           "Primary backend",
			backend:        "primary",
			expectedRepo:   "/tmp/test-repo",
			expectedPass:   "test-password",
			expectedAWSKey: "AKIATEST123",
			expectError:    false,
		},
		{
			name:           "Empty backend (defaults to primary)",
			backend:        "",
			expectedRepo:   "/tmp/test-repo",
			expectedPass:   "test-password",
			expectedAWSKey: "AKIATEST123",
			expectError:    false,
		},
		{
			name:           "S3 backend",
			backend:        "s3-backup",
			expectedRepo:   "s3:s3.amazonaws.com/my-bucket/backup",
			expectedPass:   "s3-password",
			expectedAWSKey: "AKIAS3TEST",
			expectError:    false,
		},
		{
			name:           "Local backend without AWS",
			backend:        "local-backup",
			expectedRepo:   "/mnt/backup/restic",
			expectedPass:   "local-password",
			expectedAWSKey: "",
			expectError:    false,
		},
		{
			name:        "Non-existent backend",
			backend:     "non-existent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exporter, err := NewEnvExporter(cfg, tt.backend)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if exporter.Repository != tt.expectedRepo {
				t.Errorf("Repository = %q, want %q", exporter.Repository, tt.expectedRepo)
			}
			if exporter.Password != tt.expectedPass {
				t.Errorf("Password = %q, want %q", exporter.Password, tt.expectedPass)
			}
			if exporter.AWSAccessKeyID != tt.expectedAWSKey {
				t.Errorf("AWSAccessKeyID = %q, want %q", exporter.AWSAccessKeyID, tt.expectedAWSKey)
			}
		})
	}
}

func TestNewEnvExporterValidation(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *Config
		backend     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Nil config",
			cfg:         nil,
			backend:     "primary",
			expectError: true,
			errorMsg:    "config is nil",
		},
		{
			name: "Missing repository",
			cfg: &Config{
				Password: "test-password",
			},
			backend:     "primary",
			expectError: true,
			errorMsg:    "repository not configured",
		},
		{
			name: "Missing password",
			cfg: &Config{
				Repository: "/tmp/repo",
			},
			backend:     "primary",
			expectError: true,
			errorMsg:    "password not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEnvExporter(tt.cfg, tt.backend)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Error message %q does not contain %q", err.Error(), tt.errorMsg)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestEnvExporterExportBash(t *testing.T) {
	exporter := &EnvExporter{
		Repository:         "/tmp/test-repo",
		Password:           "test-password",
		AWSAccessKeyID:     "AKIATEST123",
		AWSSecretAccessKey: "secret-key-123",
	}

	output, err := exporter.Export(FormatBash)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Check required exports
	if !strings.Contains(output, "export RESTIC_REPOSITORY=") {
		t.Error("Output missing RESTIC_REPOSITORY export")
	}
	if !strings.Contains(output, "export RESTIC_PASSWORD=") {
		t.Error("Output missing RESTIC_PASSWORD export")
	}
	if !strings.Contains(output, "export AWS_ACCESS_KEY_ID=") {
		t.Error("Output missing AWS_ACCESS_KEY_ID export")
	}
	if !strings.Contains(output, "export AWS_SECRET_ACCESS_KEY=") {
		t.Error("Output missing AWS_SECRET_ACCESS_KEY export")
	}

	// Check values
	if !strings.Contains(output, "/tmp/test-repo") {
		t.Error("Output missing repository path")
	}
	if !strings.Contains(output, "test-password") {
		t.Error("Output missing password")
	}
}

func TestEnvExporterExportBashWithoutAWS(t *testing.T) {
	exporter := &EnvExporter{
		Repository: "/tmp/test-repo",
		Password:   "test-password",
	}

	output, err := exporter.Export(FormatBash)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Should not contain AWS exports
	if strings.Contains(output, "AWS_ACCESS_KEY_ID") {
		t.Error("Output should not contain AWS_ACCESS_KEY_ID")
	}
	if strings.Contains(output, "AWS_SECRET_ACCESS_KEY") {
		t.Error("Output should not contain AWS_SECRET_ACCESS_KEY")
	}
}

func TestEnvExporterExportFish(t *testing.T) {
	exporter := &EnvExporter{
		Repository:         "s3:s3.amazonaws.com/bucket",
		Password:           "fish-password",
		AWSAccessKeyID:     "AKIAFISH123",
		AWSSecretAccessKey: "fish-secret",
	}

	output, err := exporter.Export(FormatFish)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Check fish format
	if !strings.Contains(output, "set -x RESTIC_REPOSITORY") {
		t.Error("Output missing RESTIC_REPOSITORY set")
	}
	if !strings.Contains(output, "set -x RESTIC_PASSWORD") {
		t.Error("Output missing RESTIC_PASSWORD set")
	}
	if !strings.Contains(output, "set -x AWS_ACCESS_KEY_ID") {
		t.Error("Output missing AWS_ACCESS_KEY_ID set")
	}

	// Check values
	if !strings.Contains(output, "s3:s3.amazonaws.com/bucket") {
		t.Error("Output missing repository URL")
	}
}

func TestEnvExporterExportPowershell(t *testing.T) {
	exporter := &EnvExporter{
		Repository:         "C:\\backup\\restic",
		Password:           "pwsh-password",
		AWSAccessKeyID:     "AKIAPWSH123",
		AWSSecretAccessKey: "pwsh-secret",
	}

	output, err := exporter.Export(FormatPowershell)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Check PowerShell format
	if !strings.Contains(output, "$env:RESTIC_REPOSITORY") {
		t.Error("Output missing RESTIC_REPOSITORY env")
	}
	if !strings.Contains(output, "$env:RESTIC_PASSWORD") {
		t.Error("Output missing RESTIC_PASSWORD env")
	}
	if !strings.Contains(output, "$env:AWS_ACCESS_KEY_ID") {
		t.Error("Output missing AWS_ACCESS_KEY_ID env")
	}

	// Check values
	if !strings.Contains(output, "C:\\backup\\restic") {
		t.Error("Output missing repository path")
	}
}

func TestEnvExporterUnsupportedFormat(t *testing.T) {
	exporter := &EnvExporter{
		Repository: "/tmp/repo",
		Password:   "password",
	}

	_, err := exporter.Export(ExportFormat("unsupported"))
	if err == nil {
		t.Error("Expected error for unsupported format")
	}
	if !strings.Contains(err.Error(), "unsupported format") {
		t.Errorf("Error message %q does not mention unsupported format", err.Error())
	}
}

func TestEscapeShell(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple string",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "String with single quote",
			input:    "hello'world",
			expected: "hello'\\''world",
		},
		{
			name:     "String with multiple single quotes",
			input:    "it's can't won't",
			expected: "it'\\''s can'\\''t won'\\''t",
		},
		{
			name:     "Password with special chars",
			input:    "p@ss'w0rd!",
			expected: "p@ss'\\''w0rd!",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Only single quote",
			input:    "'",
			expected: "'\\''",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeShell(tt.input)
			if result != tt.expected {
				t.Errorf("escapeShell(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEscapePowershell(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple string",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "String with single quote",
			input:    "hello'world",
			expected: "hello''world",
		},
		{
			name:     "String with multiple single quotes",
			input:    "it's can't won't",
			expected: "it''s can''t won''t",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Only single quote",
			input:    "'",
			expected: "''",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapePowershell(tt.input)
			if result != tt.expected {
				t.Errorf("escapePowershell(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
