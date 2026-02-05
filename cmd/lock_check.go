package cmd

import (
	"fmt"
	"os"

	"resticm/internal/config"
	"resticm/internal/restic"
)

// LockCheckResult contains the result of a lock verification
type LockCheckResult struct {
	HasStaleLocks bool
	Errors        []error
}

// VerifyNoStaleLocks checks all repositories for stale locks from the current host
// This is critical for S3 backends with Object Lock where stale locks cannot be removed
func VerifyNoStaleLocks(cfg *config.Config, checkBackends bool) *LockCheckResult {
	result := &LockCheckResult{
		Errors: make([]error, 0),
	}

	if !cfg.VerifyNoLocks {
		return result
	}

	hostname, err := os.Hostname()
	if err != nil {
		PrintWarning("Could not get hostname for lock verification: %v", err)
		return result
	}

	fmt.Println()
	PrintInfo("🔐 Verifying no stale locks remain...")

	// Check primary repository
	executor := restic.NewExecutor(cfg.Repository, cfg.GetPassword())
	executor.SetAWSCredentials(cfg.GetAWSAccessKeyID(), cfg.GetAWSSecretAccessKey())

	if lockResult, err := executor.VerifyNoStaleLocks(hostname); err != nil {
		PrintWarning("Could not verify locks on primary: %v", err)
	} else {
		if lockResult.HasOwnLocks {
			result.HasStaleLocks = true
			PrintError("⚠️  STALE LOCK DETECTED on primary repository!")
			PrintError("   This host (%s) still has %d lock(s) that should have been released", hostname, len(lockResult.OwnHostLocks))
			for _, lock := range lockResult.OwnHostLocks {
				PrintError("   - Lock from PID %d at %s", lock.PID, lock.Time.Format("2006-01-02 15:04:05"))
			}
			result.Errors = append(result.Errors, fmt.Errorf("stale lock detected on primary repository"))
		} else {
			PrintSuccess("No stale locks from this host on primary")
		}
		if lockResult.HasOtherLocks {
			PrintInfo("Note: %d lock(s) from other hosts (normal in multi-server setup)", len(lockResult.OtherHostLocks))
		}
	}

	// Check copy backends if requested
	if checkBackends && len(cfg.CopyToBackends) > 0 {
		for _, backendName := range cfg.CopyToBackends {
			backend, ok := cfg.Backends[backendName]
			if !ok {
				continue
			}

			backendExecutor := restic.NewExecutor(backend.Repository, backend.Password)
			backendExecutor.SetAWSCredentials(backend.AWSAccessKeyID, backend.AWSSecretAccessKey)

			if lockResult, err := backendExecutor.VerifyNoStaleLocks(hostname); err != nil {
				PrintWarning("Could not verify locks on %s: %v", backendName, err)
			} else {
				if lockResult.HasOwnLocks {
					result.HasStaleLocks = true
					PrintError("⚠️  STALE LOCK DETECTED on backend %s!", backendName)
					result.Errors = append(result.Errors, fmt.Errorf("stale lock detected on backend %s", backendName))
				} else {
					PrintSuccess("No stale locks from this host on %s", backendName)
				}
			}
		}
	}

	if result.HasStaleLocks {
		PrintError("")
		PrintError("🚨 IMPORTANT: Stale locks were detected!")
		PrintError("   For S3 with Object Lock (immutable), these locks cannot be removed")
		PrintError("   and will block repository access until the retention period expires.")
	}

	return result
}

// VerifyNoStaleLocksSingleRepo checks a single repository for stale locks
func VerifyNoStaleLocksSingleRepo(executor *restic.Executor, repoName string) *LockCheckResult {
	result := &LockCheckResult{
		Errors: make([]error, 0),
	}

	hostname, err := os.Hostname()
	if err != nil {
		return result
	}

	if lockResult, err := executor.VerifyNoStaleLocks(hostname); err != nil {
		PrintWarning("Could not verify locks on %s: %v", repoName, err)
	} else {
		if lockResult.HasOwnLocks {
			result.HasStaleLocks = true
			PrintError("⚠️  STALE LOCK: This host still has %d lock(s) on %s", len(lockResult.OwnHostLocks), repoName)
			result.Errors = append(result.Errors, fmt.Errorf("stale lock detected on %s", repoName))
		}
		if lockResult.HasOtherLocks {
			PrintInfo("Note: %d lock(s) from other hosts on %s (normal)", len(lockResult.OtherHostLocks), repoName)
		}
	}

	return result
}
