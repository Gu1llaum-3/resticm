package restic

import (
	"encoding/json"
	"os"
	"strings"
	"time"
)

// Snapshot represents a restic snapshot
type Snapshot struct {
	ID       string    `json:"id"`
	ShortID  string    `json:"short_id"`
	Time     time.Time `json:"time"`
	Hostname string    `json:"hostname"`
	Username string    `json:"username"`
	Tags     []string  `json:"tags"`
	Paths    []string  `json:"paths"`
}

// Lock represents a restic lock
type Lock struct {
	Time     time.Time `json:"time"`
	Hostname string    `json:"hostname"`
	Username string    `json:"username"`
	PID      int       `json:"pid"`
}

// Stats represents repository statistics
type Stats struct {
	TotalSize      int64 `json:"total_size"`
	TotalFileCount int   `json:"total_file_count"`
}

// ListSnapshots returns all snapshots in the repository
func (e *Executor) ListSnapshots() ([]Snapshot, error) {
	output, err := e.RunWithOutput("snapshots", "--json")
	if err != nil {
		return nil, err
	}

	var snapshots []Snapshot
	if err := json.Unmarshal([]byte(output), &snapshots); err != nil {
		return nil, err
	}

	return snapshots, nil
}

// GetLatestSnapshot returns the most recent snapshot
func (e *Executor) GetLatestSnapshot() (*Snapshot, error) {
	output, err := e.RunWithOutput("snapshots", "--json", "--latest", "1")
	if err != nil {
		return nil, err
	}

	var snapshots []Snapshot
	if err := json.Unmarshal([]byte(output), &snapshots); err != nil {
		return nil, err
	}

	if len(snapshots) == 0 {
		return nil, nil
	}

	return &snapshots[0], nil
}

// ListLocks returns all locks in the repository
func (e *Executor) ListLocks() ([]Lock, error) {
	output, err := e.RunWithOutput("list", "locks", "--json")
	if err != nil {
		return nil, err
	}

	// Handle empty output (no locks) - restic returns empty string, not "[]"
	output = strings.TrimSpace(output)
	if output == "" {
		return []Lock{}, nil
	}

	var locks []Lock
	if err := json.Unmarshal([]byte(output), &locks); err != nil {
		return nil, err
	}

	return locks, nil
}

// HasLocks checks if the repository has any locks
func (e *Executor) HasLocks() (bool, error) {
	locks, err := e.ListLocks()
	if err != nil {
		return false, err
	}
	return len(locks) > 0, nil
}

// UnlockRepository removes all locks from the repository
func (e *Executor) UnlockRepository() error {
	return e.Run("unlock")
}

// GetStats returns repository statistics
func (e *Executor) GetStats() (*Stats, error) {
	output, err := e.RunWithOutput("stats", "--json")
	if err != nil {
		return nil, err
	}

	var stats Stats
	if err := json.Unmarshal([]byte(output), &stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// GetCurrentHostname returns the current hostname
func GetCurrentHostname() (string, error) {
	return os.Hostname()
}

// HasLocksFromHost checks if the repository has any locks from a specific hostname
func (e *Executor) HasLocksFromHost(hostname string) (bool, []Lock, error) {
	locks, err := e.ListLocks()
	if err != nil {
		return false, nil, err
	}

	var hostLocks []Lock
	for _, lock := range locks {
		if lock.Hostname == hostname {
			hostLocks = append(hostLocks, lock)
		}
	}

	return len(hostLocks) > 0, hostLocks, nil
}

// GetLocksSummary returns a summary of all locks in the repository
// grouped by hostname for multi-server awareness
func (e *Executor) GetLocksSummary() (map[string][]Lock, error) {
	locks, err := e.ListLocks()
	if err != nil {
		return nil, err
	}

	summary := make(map[string][]Lock)
	for _, lock := range locks {
		summary[lock.Hostname] = append(summary[lock.Hostname], lock)
	}

	return summary, nil
}

// VerifyNoStaleLocks checks that no locks from the current host remain
// This is important for S3 backends with Object Lock where stale locks
// cannot be removed and would block the repository
func (e *Executor) VerifyNoStaleLocks(currentHostname string) (*LockVerificationResult, error) {
	locks, err := e.ListLocks()
	if err != nil {
		return nil, err
	}

	result := &LockVerificationResult{
		CurrentHostname: currentHostname,
		OtherHostLocks:  make([]Lock, 0),
		OwnHostLocks:    make([]Lock, 0),
	}

	for _, lock := range locks {
		if lock.Hostname == currentHostname {
			result.OwnHostLocks = append(result.OwnHostLocks, lock)
		} else {
			result.OtherHostLocks = append(result.OtherHostLocks, lock)
		}
	}

	result.HasOwnLocks = len(result.OwnHostLocks) > 0
	result.HasOtherLocks = len(result.OtherHostLocks) > 0

	return result, nil
}

// LockVerificationResult contains the result of lock verification
type LockVerificationResult struct {
	CurrentHostname string
	HasOwnLocks     bool
	HasOtherLocks   bool
	OwnHostLocks    []Lock
	OtherHostLocks  []Lock
}
