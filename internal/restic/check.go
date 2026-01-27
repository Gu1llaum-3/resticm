package restic

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// CheckOptions contains options for the check operation
type CheckOptions struct {
	ReadData       bool
	ReadDataSubset string
}

// Check verifies repository integrity
func (e *Executor) Check(opts CheckOptions) error {
	args := []string{"check"}

	if opts.ReadData {
		args = append(args, "--read-data")
	} else if opts.ReadDataSubset != "" {
		args = append(args, "--read-data-subset", opts.ReadDataSubset)
	}

	return e.Run(args...)
}

// DeepCheckTracker tracks when deep checks were performed
type DeepCheckTracker struct {
	path       string
	repository string
}

// DeepCheckState stores deep check state
type DeepCheckState struct {
	Repository string    `yaml:"repository"`
	LastCheck  time.Time `yaml:"last_check"`
}

// NewDeepCheckTracker creates a new deep check tracker for a specific repository
func NewDeepCheckTracker(repositoryURL string) (*DeepCheckTracker, error) {
	// Calculate a short hash of the repository URL for uniqueness
	hash := sha256.Sum256([]byte(repositoryURL))
	shortHash := fmt.Sprintf("%x", hash[:4]) // Use first 8 hex chars (4 bytes)

	// Determine base directory based on privileges
	var baseDir string
	if os.Geteuid() == 0 {
		// Root: use system directory
		baseDir = "/var/lib/resticm"
	} else {
		// Non-root: use user config directory
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		baseDir = filepath.Join(home, ".config", "resticm")
	}

	path := filepath.Join(baseDir, fmt.Sprintf("deep_check_%s.yaml", shortHash))
	return &DeepCheckTracker{
		path:       path,
		repository: repositoryURL,
	}, nil
}

// LastCheck returns the time of the last deep check
func (t *DeepCheckTracker) LastCheck() (time.Time, error) {
	data, err := os.ReadFile(t.path)
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}

	var state DeepCheckState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return time.Time{}, err
	}

	// Validate that the file is for the correct repository
	if state.Repository != t.repository {
		// File exists but is for a different repository (hash collision)
		// Treat as if no check was done
		return time.Time{}, nil
	}

	return state.LastCheck, nil
}

// RecordCheck records a deep check
func (t *DeepCheckTracker) RecordCheck() error {
	state := DeepCheckState{
		Repository: t.repository,
		LastCheck:  time.Now(),
	}

	data, err := yaml.Marshal(&state)
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(t.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	return os.WriteFile(t.path, data, 0600)
}

// ShouldRunDeepCheck returns true if a deep check should be run
func (t *DeepCheckTracker) ShouldRunDeepCheck(intervalDays int) bool {
	if intervalDays <= 0 {
		return false
	}

	last, err := t.LastCheck()
	if err != nil || last.IsZero() {
		return true
	}

	deadline := last.Add(time.Duration(intervalDays) * 24 * time.Hour)
	return time.Now().After(deadline)
}

// FormatDuration formats a duration for display
func FormatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	if days > 0 {
		return fmt.Sprintf("%d days", days)
	}
	return d.Round(time.Minute).String()
}
