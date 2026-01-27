package restic

import (
	"strings"
	"testing"
)

func TestNewExecutor(t *testing.T) {
	exec := NewExecutor("/tmp/repo", "secret")

	if exec == nil {
		t.Fatal("NewExecutor() returned nil")
	}

	if exec.Repository != "/tmp/repo" {
		t.Errorf("Repository = %q, want %q", exec.Repository, "/tmp/repo")
	}

	if exec.Password != "secret" {
		t.Errorf("Password = %q, want %q", exec.Password, "secret")
	}
}

func TestExecutorBuildEnv(t *testing.T) {
	exec := NewExecutor("/tmp/restic-repo", "testpassword")
	env := exec.buildEnv()

	// Check that RESTIC_REPOSITORY is set
	found := false
	for _, e := range env {
		if strings.HasPrefix(e, "RESTIC_REPOSITORY=") {
			found = true
			if e != "RESTIC_REPOSITORY=/tmp/restic-repo" {
				t.Errorf("RESTIC_REPOSITORY = %q, want %q", e, "RESTIC_REPOSITORY=/tmp/restic-repo")
			}
		}
	}
	if !found {
		t.Error("RESTIC_REPOSITORY not found in environment")
	}

	// Check that RESTIC_PASSWORD is set
	found = false
	for _, e := range env {
		if strings.HasPrefix(e, "RESTIC_PASSWORD=") {
			found = true
			if e != "RESTIC_PASSWORD=testpassword" {
				t.Errorf("RESTIC_PASSWORD = %q, want %q", e, "RESTIC_PASSWORD=testpassword")
			}
		}
	}
	if !found {
		t.Error("RESTIC_PASSWORD not found in environment")
	}
}

func TestExecutorBuildEnvWithAWS(t *testing.T) {
	exec := NewExecutor("s3:s3.amazonaws.com/bucket/restic", "testpassword")
	exec.SetAWSCredentials("AKIATEST", "secretkey")

	env := exec.buildEnv()

	// Check AWS credentials
	foundAccess := false
	foundSecret := false
	for _, e := range env {
		if strings.HasPrefix(e, "AWS_ACCESS_KEY_ID=") {
			foundAccess = true
			if e != "AWS_ACCESS_KEY_ID=AKIATEST" {
				t.Errorf("AWS_ACCESS_KEY_ID = %q, want %q", e, "AWS_ACCESS_KEY_ID=AKIATEST")
			}
		}
		if strings.HasPrefix(e, "AWS_SECRET_ACCESS_KEY=") {
			foundSecret = true
			if e != "AWS_SECRET_ACCESS_KEY=secretkey" {
				t.Errorf("AWS_SECRET_ACCESS_KEY = %q, want %q", e, "AWS_SECRET_ACCESS_KEY=secretkey")
			}
		}
	}
	if !foundAccess {
		t.Error("AWS_ACCESS_KEY_ID not found in environment")
	}
	if !foundSecret {
		t.Error("AWS_SECRET_ACCESS_KEY not found in environment")
	}
}

func TestBackupOptions(t *testing.T) {
	opts := BackupOptions{
		Directories:     []string{"/home", "/etc"},
		ExcludePatterns: []string{"*.tmp", "*.log"},
		Tags:            []string{"daily", "server1"},
		Hostname:        "myhost",
	}

	if len(opts.Directories) != 2 {
		t.Errorf("len(Directories) = %d, want 2", len(opts.Directories))
	}

	if len(opts.Tags) != 2 {
		t.Errorf("len(Tags) = %d, want 2", len(opts.Tags))
	}
}

func TestForgetOptions(t *testing.T) {
	opts := ForgetOptions{
		KeepHourly:  24,
		KeepDaily:   7,
		KeepWeekly:  4,
		KeepMonthly: 12,
		KeepYearly:  5,
		Prune:       true,
	}

	if opts.KeepDaily != 7 {
		t.Errorf("KeepDaily = %d, want 7", opts.KeepDaily)
	}

	if !opts.Prune {
		t.Error("Prune should be true")
	}
}

func TestCheckOptions(t *testing.T) {
	opts := CheckOptions{
		ReadData: true,
	}

	if !opts.ReadData {
		t.Error("ReadData should be true")
	}
}

func TestSetAWSCredentials(t *testing.T) {
	exec := NewExecutor("/tmp/repo", "secret")
	exec.SetAWSCredentials("AKIA123", "secret123")

	if exec.Env["AWS_ACCESS_KEY_ID"] != "AKIA123" {
		t.Errorf("AWS_ACCESS_KEY_ID = %q, want %q", exec.Env["AWS_ACCESS_KEY_ID"], "AKIA123")
	}

	if exec.Env["AWS_SECRET_ACCESS_KEY"] != "secret123" {
		t.Errorf("AWS_SECRET_ACCESS_KEY = %q, want %q", exec.Env["AWS_SECRET_ACCESS_KEY"], "secret123")
	}
}

func TestExecutorDryRun(t *testing.T) {
	exec := NewExecutor("/tmp/repo", "secret")
	exec.DryRun = true

	if !exec.DryRun {
		t.Error("DryRun should be true")
	}
}

func TestExecutorVerbose(t *testing.T) {
	exec := NewExecutor("/tmp/repo", "secret")
	exec.Verbose = true

	if !exec.Verbose {
		t.Error("Verbose should be true")
	}
}
