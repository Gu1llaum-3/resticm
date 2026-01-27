package restic

// Prune removes unreferenced data from the repository
func (e *Executor) Prune() error {
	args := []string{"prune"}

	if e.DryRun {
		args = append(args, "--dry-run")
	}

	return e.Run(args...)
}
