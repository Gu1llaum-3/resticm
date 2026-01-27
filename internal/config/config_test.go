package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
repository: "/tmp/restic-repo"
password: "testpassword"
directories:
  - /home
  - /etc
exclude_patterns:
  - "*.tmp"
retention:
  keep_daily: 7
  keep_weekly: 4
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Repository != "/tmp/restic-repo" {
		t.Errorf("Repository = %q, want %q", cfg.Repository, "/tmp/restic-repo")
	}

	if cfg.Password != "testpassword" {
		t.Errorf("Password = %q, want %q", cfg.Password, "testpassword")
	}

	if len(cfg.Directories) != 2 {
		t.Errorf("len(Directories) = %d, want 2", len(cfg.Directories))
	}

	if len(cfg.ExcludePatterns) != 1 {
		t.Errorf("len(ExcludePatterns) = %d, want 1", len(cfg.ExcludePatterns))
	}

	if cfg.Retention.KeepDaily != 7 {
		t.Errorf("Retention.KeepDaily = %d, want 7", cfg.Retention.KeepDaily)
	}
}

func TestLoadConfigWithBackends(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
repository: "/tmp/restic-repo"
password: "testpassword"
directories:
  - /home
backends:
  secondary:
    repository: "s3:s3.amazonaws.com/bucket/restic"
    password: "s3password"
  local:
    repository: "/mnt/backup/restic"
    password: "localpassword"
copy_to_backends:
  - secondary
  - local
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(cfg.Backends) != 2 {
		t.Errorf("len(Backends) = %d, want 2", len(cfg.Backends))
	}

	secondary, ok := cfg.Backends["secondary"]
	if !ok {
		t.Fatal("Backend 'secondary' not found")
	}

	if secondary.Repository != "s3:s3.amazonaws.com/bucket/restic" {
		t.Errorf("secondary.Repository = %q, want %q", secondary.Repository, "s3:s3.amazonaws.com/bucket/restic")
	}

	if len(cfg.CopyToBackends) != 2 {
		t.Errorf("len(CopyToBackends) = %d, want 2", len(cfg.CopyToBackends))
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Expected error for missing config file, got nil")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				Repository:  "/tmp/repo",
				Password:    "secret",
				Directories: []string{"/home"},
			},
			wantErr: false,
		},
		{
			name: "missing repository",
			cfg: Config{
				Password:    "secret",
				Directories: []string{"/home"},
			},
			wantErr: true,
		},
		{
			name: "missing password",
			cfg: Config{
				Repository:  "/tmp/repo",
				Directories: []string{"/home"},
			},
			wantErr: true,
		},
		{
			name: "missing directories",
			cfg: Config{
				Repository: "/tmp/repo",
				Password:   "secret",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetPassword(t *testing.T) {
	// Clear any existing env var
	_ = os.Unsetenv("RESTIC_PASSWORD")

	// Test with direct password
	cfg := &Config{
		Password: "direct-password",
	}

	pwd := cfg.GetPassword()
	if pwd != "direct-password" {
		t.Errorf("GetPassword() = %q, want %q", pwd, "direct-password")
	}

	// Test with environment variable
	_ = os.Setenv("RESTIC_PASSWORD", "env-password")
	defer func() { _ = os.Unsetenv("RESTIC_PASSWORD") }()

	pwd = cfg.GetPassword()
	if pwd != "env-password" {
		t.Errorf("GetPassword() with env = %q, want %q", pwd, "env-password")
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input string
		want  string
	}{
		{"~/.config/restic", filepath.Join(home, ".config/restic")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ExpandPath(tt.input)
			if got != tt.want {
				t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Retention.KeepDaily != 7 {
		t.Errorf("DefaultConfig().Retention.KeepDaily = %d, want 7", cfg.Retention.KeepDaily)
	}

	if cfg.Retention.KeepYearly != 5 {
		t.Errorf("DefaultConfig().Retention.KeepYearly = %d, want 5", cfg.Retention.KeepYearly)
	}

	if cfg.DeepCheckIntervalDays != 30 {
		t.Errorf("DefaultConfig().DeepCheckIntervalDays = %d, want 30", cfg.DeepCheckIntervalDays)
	}
}
