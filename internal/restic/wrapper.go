// Package restic provides a wrapper for restic CLI operations
package restic

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Executor handles restic command execution
type Executor struct {
	Repository string
	Password   string
	Env        map[string]string
	DryRun     bool
	Verbose    bool
	Stdout     io.Writer
	Stderr     io.Writer
	CacheDir   string
}

// NewExecutor creates a new restic executor
func NewExecutor(repository, password string) *Executor {
	return &Executor{
		Repository: repository,
		Password:   password,
		Env:        make(map[string]string),
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
	}
}

// SetAWSCredentials sets AWS credentials for S3 backends
func (e *Executor) SetAWSCredentials(accessKeyID, secretAccessKey string) {
	if accessKeyID != "" {
		e.Env["AWS_ACCESS_KEY_ID"] = accessKeyID
	}
	if secretAccessKey != "" {
		e.Env["AWS_SECRET_ACCESS_KEY"] = secretAccessKey
	}
}

// Run executes a restic command
func (e *Executor) Run(args ...string) error {
	cmd := exec.Command("restic", args...)
	cmd.Env = e.buildEnv()
	cmd.Stdout = e.Stdout
	cmd.Stderr = e.Stderr

	if e.Verbose {
		fmt.Printf("$ restic %s\n", strings.Join(args, " "))
	}

	if err := cmd.Run(); err != nil {
		// Wrap error with restic exit code description
		if exitErr, ok := err.(*exec.ExitError); ok {
			code := exitErr.ExitCode()
			desc := GetExitCodeDescription(code)
			return fmt.Errorf("exit status %d (%s)", code, desc)
		}
		return err
	}
	return nil
}

// GetExitCodeDescription returns a human-readable description for restic exit codes
// See: https://restic.readthedocs.io/en/latest/075_scripting.html
func GetExitCodeDescription(code int) string {
	switch code {
	case 0:
		return "success"
	case 1:
		return "command failed"
	case 2:
		return "Go runtime error"
	case 3:
		return "could not read some source data"
	case 10:
		return "repository does not exist"
	case 11:
		return "failed to lock repository"
	case 12:
		return "wrong password"
	case 130:
		return "interrupted by SIGINT/SIGSTOP"
	default:
		return "unknown error - see https://restic.readthedocs.io/en/latest/075_scripting.html"
	}
}

// RunWithOutput executes a restic command and returns the output
func (e *Executor) RunWithOutput(args ...string) (string, error) {
	cmd := exec.Command("restic", args...)
	cmd.Env = e.buildEnv()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if e.Verbose {
		fmt.Printf("$ restic %s\n", strings.Join(args, " "))
	}

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// RunWithStreaming executes a restic command with live output
func (e *Executor) RunWithStreaming(args ...string) error {
	cmd := exec.Command("restic", args...)
	cmd.Env = e.buildEnv()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// buildEnv builds the environment for restic commands
func (e *Executor) buildEnv() []string {
	env := os.Environ()

	// Add repository
	env = append(env, "RESTIC_REPOSITORY="+e.Repository)

	// Add password
	env = append(env, "RESTIC_PASSWORD="+e.Password)

	// Add cache directory if set
	if e.CacheDir != "" {
		env = append(env, "RESTIC_CACHE_DIR="+e.CacheDir)
	}

	// Add custom environment variables
	for k, v := range e.Env {
		env = append(env, k+"="+v)
	}

	return env
}

// IsInitialized checks if the repository is initialized
func (e *Executor) IsInitialized() bool {
	_, err := e.RunWithOutput("snapshots", "--json", "-q")
	return err == nil
}

// Init initializes the repository
func (e *Executor) Init() error {
	return e.Run("init")
}

// InitOptions contains options for repository initialization
type InitOptions struct {
	FromRepository     string // Source repository to copy chunker params from
	FromPassword       string // Password for source repository
	CopyChunkerParams  bool   // Whether to copy chunker params from source
	FromAWSAccessKeyID string // AWS credentials for source if S3
	FromAWSSecret      string
}

// InitWithOptions initializes the repository with options
func (e *Executor) InitWithOptions(opts InitOptions) error {
	args := []string{"init"}

	if opts.CopyChunkerParams && opts.FromRepository != "" {
		args = append(args, "--from-repo", opts.FromRepository, "--copy-chunker-params")

		// Handle source password via temp file
		if opts.FromPassword != "" {
			tmpFile, err := createTempPasswordFile(opts.FromPassword)
			if err != nil {
				return fmt.Errorf("failed to create temp password file: %w", err)
			}
			defer func() { _ = os.Remove(tmpFile) }()
			args = append(args, "--from-password-file", tmpFile)
		}

		// Set AWS credentials for source if needed
		if opts.FromAWSAccessKeyID != "" {
			e.Env["AWS_ACCESS_KEY_ID"] = opts.FromAWSAccessKeyID
		}
		if opts.FromAWSSecret != "" {
			e.Env["AWS_SECRET_ACCESS_KEY"] = opts.FromAWSSecret
		}
	}

	return e.Run(args...)
}

// CheckResticInstalled verifies restic is available
func CheckResticInstalled() error {
	_, err := exec.LookPath("restic")
	if err != nil {
		return fmt.Errorf("restic not found in PATH. Please install restic first")
	}
	return nil
}

// GetVersion returns the restic version
func GetVersion() (string, error) {
	cmd := exec.Command("restic", "version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// ResticError represents an error from restic
type ResticError struct {
	Command  string
	ExitCode int
	Stderr   string
}

func (e *ResticError) Error() string {
	return fmt.Sprintf("restic %s failed (exit %d): %s", e.Command, e.ExitCode, e.Stderr)
}
