package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"resticm/internal/config"
)

func TestEnvCmd(t *testing.T) {
	// Create temporary config directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	contextDir := filepath.Join(tmpDir, ".config", "resticm")
	if err := os.MkdirAll(contextDir, 0700); err != nil {
		t.Fatalf("Failed to create context dir: %v", err)
	}

	// Override home directory for context
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create test config
	configContent := `
repository: "/tmp/test-repo"
password: "test-password"
aws_access_key_id: "AKIATEST123"
aws_secret_access_key: "secret-key-123"

directories:
  - /tmp/test

backends:
  s3-backup:
    repository: "s3:s3.amazonaws.com/my-bucket/backup"
    password: "s3-password"
    aws_access_key_id: "AKIAS3TEST"
    aws_secret_access_key: "s3-secret-key"
  
  local-backup:
    repository: "/mnt/backup/restic"
    password: "local-password"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	tests := []struct {
		name           string
		backend        string
		format         string
		activeBackend  string
		expectedRepo   string
		expectedPass   string
		expectedAWSKey string
		expectError    bool
	}{
		{
			name:           "Primary backend (no active backend set)",
			backend:        "",
			format:         "bash",
			activeBackend:  "",
			expectedRepo:   "/tmp/test-repo",
			expectedPass:   "test-password",
			expectedAWSKey: "AKIATEST123",
			expectError:    false,
		},
		{
			name:           "Explicit primary backend",
			backend:        "primary",
			format:         "bash",
			activeBackend:  "",
			expectedRepo:   "/tmp/test-repo",
			expectedPass:   "test-password",
			expectedAWSKey: "AKIATEST123",
			expectError:    false,
		},
		{
			name:           "S3 backend via flag",
			backend:        "s3-backup",
			format:         "bash",
			activeBackend:  "",
			expectedRepo:   "s3:s3.amazonaws.com/my-bucket/backup",
			expectedPass:   "s3-password",
			expectedAWSKey: "AKIAS3TEST",
			expectError:    false,
		},
		{
			name:           "Local backend via flag",
			backend:        "local-backup",
			format:         "bash",
			activeBackend:  "",
			expectedRepo:   "/mnt/backup/restic",
			expectedPass:   "local-password",
			expectedAWSKey: "",
			expectError:    false,
		},
		{
			name:           "Active backend from context",
			backend:        "",
			format:         "bash",
			activeBackend:  "s3-backup",
			expectedRepo:   "s3:s3.amazonaws.com/my-bucket/backup",
			expectedPass:   "s3-password",
			expectedAWSKey: "AKIAS3TEST",
			expectError:    false,
		},
		{
			name:          "Non-existent backend",
			backend:       "non-existent",
			format:        "bash",
			activeBackend: "",
			expectError:   true,
		},
		{
			name:           "Fish format",
			backend:        "primary",
			format:         "fish",
			activeBackend:  "",
			expectedRepo:   "/tmp/test-repo",
			expectedPass:   "test-password",
			expectedAWSKey: "AKIATEST123",
			expectError:    false,
		},
		{
			name:           "PowerShell format",
			backend:        "s3-backup",
			format:         "powershell",
			activeBackend:  "",
			expectedRepo:   "s3:s3.amazonaws.com/my-bucket/backup",
			expectedPass:   "s3-password",
			expectedAWSKey: "AKIAS3TEST",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset state
			cfgFile = configPath
			cfg = nil
			envBackend = tt.backend
			envFormat = tt.format

			// Set active backend if specified
			if tt.activeBackend != "" {
				if err := config.SetActiveBackend(tt.activeBackend); err != nil {
					t.Fatalf("Failed to set active backend: %v", err)
				}
			} else {
				// Clear active backend
				if err := config.SetActiveBackend(""); err != nil {
					t.Fatalf("Failed to clear active backend: %v", err)
				}
			}

			// Capture output
			var buf bytes.Buffer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run command
			err := runEnv()

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout
			buf.ReadFrom(r)
			output := buf.String()

			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify output based on format
			switch tt.format {
			case "bash", "sh", "":
				if !strings.Contains(output, "export RESTIC_REPOSITORY=") {
					t.Error("Output missing RESTIC_REPOSITORY export")
				}
				if !strings.Contains(output, "export RESTIC_PASSWORD=") {
					t.Error("Output missing RESTIC_PASSWORD export")
				}
				if !strings.Contains(output, tt.expectedRepo) {
					t.Errorf("Expected repository %q not found in output", tt.expectedRepo)
				}
				if !strings.Contains(output, tt.expectedPass) {
					t.Errorf("Expected password %q not found in output", tt.expectedPass)
				}
				if tt.expectedAWSKey != "" {
					if !strings.Contains(output, "export AWS_ACCESS_KEY_ID=") {
						t.Error("Output missing AWS_ACCESS_KEY_ID export")
					}
					if !strings.Contains(output, tt.expectedAWSKey) {
						t.Errorf("Expected AWS key %q not found in output", tt.expectedAWSKey)
					}
				}

			case "fish":
				if !strings.Contains(output, "set -x RESTIC_REPOSITORY") {
					t.Error("Output missing RESTIC_REPOSITORY set")
				}
				if !strings.Contains(output, "set -x RESTIC_PASSWORD") {
					t.Error("Output missing RESTIC_PASSWORD set")
				}
				if !strings.Contains(output, tt.expectedRepo) {
					t.Errorf("Expected repository %q not found in output", tt.expectedRepo)
				}

			case "powershell", "pwsh":
				if !strings.Contains(output, "$env:RESTIC_REPOSITORY") {
					t.Error("Output missing RESTIC_REPOSITORY env")
				}
				if !strings.Contains(output, "$env:RESTIC_PASSWORD") {
					t.Error("Output missing RESTIC_PASSWORD env")
				}
				if !strings.Contains(output, tt.expectedRepo) {
					t.Errorf("Expected repository %q not found in output", tt.expectedRepo)
				}
			}
		})
	}
}

func TestEnvCmdInvalidFormat(t *testing.T) {
	// Create temporary config directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create minimal valid config
	configContent := `
repository: "/tmp/test-repo"
password: "test-password"
directories:
  - /tmp/test
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Reset state
	cfgFile = configPath
	cfg = nil
	envBackend = ""
	envFormat = "invalid-format"

	err := runEnv()
	if err == nil {
		t.Error("Expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "unsupported format") {
		t.Errorf("Error message %q does not mention unsupported format", err.Error())
	}
}
