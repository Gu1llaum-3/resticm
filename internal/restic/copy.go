package restic

import (
	"fmt"
	"os"
	"strings"
)

// CopyOptions contains options for the copy operation
type CopyOptions struct {
	FromRepository         string
	FromPassword           string
	FromAWSAccessKeyID     string
	FromAWSSecretAccessKey string
	ToRepository           string
	ToPassword             string
	ToAWSAccessKeyID       string
	ToAWSSecretAccessKey   string
	Hostname               string
	SnapshotIDs            []string
}

// Copy copies snapshots from source to destination repository
func (e *Executor) Copy(opts CopyOptions) error {
	// Check for cross-account S3 copy (both S3 with different credentials)
	if isCrossAccountS3(opts.FromRepository, opts.ToRepository, opts.FromAWSAccessKeyID, opts.ToAWSAccessKeyID) {
		return &CrossAccountS3Error{
			From: opts.FromRepository,
			To:   opts.ToRepository,
		}
	}

	args := []string{"copy"}

	// Add source repository
	args = append(args, "--from-repo", opts.FromRepository)

	// Add hostname filter
	if opts.Hostname != "" {
		args = append(args, "--host", opts.Hostname)
	}

	// Add specific snapshots if specified
	args = append(args, opts.SnapshotIDs...)

	// Create temp file for source password
	if opts.FromPassword != "" {
		tmpFile, err := createTempPasswordFile(opts.FromPassword)
		if err != nil {
			return fmt.Errorf("failed to create temp password file: %w", err)
		}
		defer func() { _ = os.Remove(tmpFile) }()
		args = append(args, "--from-password-file", tmpFile)
	}

	// Handle AWS credentials based on repository types
	fromIsS3 := strings.HasPrefix(opts.FromRepository, "s3:")
	toIsS3 := strings.HasPrefix(opts.ToRepository, "s3:")

	if fromIsS3 && toIsS3 {
		// Both S3 - they must use the same credentials (checked above)
		// Use source credentials
		if opts.FromAWSAccessKeyID != "" {
			e.Env["AWS_ACCESS_KEY_ID"] = opts.FromAWSAccessKeyID
		}
		if opts.FromAWSSecretAccessKey != "" {
			e.Env["AWS_SECRET_ACCESS_KEY"] = opts.FromAWSSecretAccessKey
		}
	} else if fromIsS3 {
		// Only source is S3 - use source credentials
		if opts.FromAWSAccessKeyID != "" {
			e.Env["AWS_ACCESS_KEY_ID"] = opts.FromAWSAccessKeyID
		}
		if opts.FromAWSSecretAccessKey != "" {
			e.Env["AWS_SECRET_ACCESS_KEY"] = opts.FromAWSSecretAccessKey
		}
	}
	// Note: If only destination is S3, executor already has the credentials

	return e.Run(args...)
}

// createTempPasswordFile creates a temporary file with the password
func createTempPasswordFile(password string) (string, error) {
	tmpFile, err := os.CreateTemp("", "resticm-pwd-*")
	if err != nil {
		return "", err
	}

	if _, err := tmpFile.WriteString(password); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return "", err
	}

	_ = tmpFile.Close()
	return tmpFile.Name(), nil
}

// CrossAccountS3Error is returned when trying to copy between S3 buckets with different credentials
type CrossAccountS3Error struct {
	From string
	To   string
}

func (e *CrossAccountS3Error) Error() string {
	return fmt.Sprintf(`cross-account S3 copy detected

Source:      %s
Destination: %s

Restic's copy command cannot handle S3 buckets with different credentials.

💡 Solution: Use rclone as an intermediary:
   1. Configure rclone with both S3 accounts
   2. Use 'rclone:remote:bucket/path' as repository URL
   3. Restic will use rclone for the copy operation

See: https://restic.readthedocs.io/en/latest/030_preparing_a_new_repo.html#other-services-via-rclone
`, e.From, e.To)
}

// isCrossAccountS3 checks if the copy is between S3 buckets with different credentials
func isCrossAccountS3(fromRepo, toRepo, fromKey, toKey string) bool {
	if !isS3Repository(fromRepo) || !isS3Repository(toRepo) {
		return false
	}
	return fromKey != "" && toKey != "" && fromKey != toKey
}

// isS3Repository checks if a repository URL is an S3 repository
func isS3Repository(repo string) bool {
	return strings.HasPrefix(repo, "s3:")
}
