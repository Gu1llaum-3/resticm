package restic

import "strconv"

// ForgetOptions contains options for the forget operation
type ForgetOptions struct {
	KeepWithin  string
	KeepHourly  int
	KeepDaily   int
	KeepWeekly  int
	KeepMonthly int
	KeepYearly  int
	Hostname    string
	Prune       bool
	GroupBy     string
}

// Forget removes old snapshots according to retention policy
func (e *Executor) Forget(opts ForgetOptions) error {
	args := []string{"forget"}

	// Add retention options
	if opts.KeepWithin != "" {
		args = append(args, "--keep-within", opts.KeepWithin)
	}
	if opts.KeepHourly > 0 {
		args = append(args, "--keep-hourly", itoa(opts.KeepHourly))
	}
	if opts.KeepDaily > 0 {
		args = append(args, "--keep-daily", itoa(opts.KeepDaily))
	}
	if opts.KeepWeekly > 0 {
		args = append(args, "--keep-weekly", itoa(opts.KeepWeekly))
	}
	if opts.KeepMonthly > 0 {
		args = append(args, "--keep-monthly", itoa(opts.KeepMonthly))
	}
	if opts.KeepYearly > 0 {
		args = append(args, "--keep-yearly", itoa(opts.KeepYearly))
	}

	// Filter by hostname
	if opts.Hostname != "" {
		args = append(args, "--host", opts.Hostname)
	}

	// Group by
	if opts.GroupBy != "" {
		args = append(args, "--group-by", opts.GroupBy)
	}

	// Add prune if requested
	if opts.Prune {
		args = append(args, "--prune")
	}

	// Add dry-run if enabled
	if e.DryRun {
		args = append(args, "--dry-run")
	}

	return e.Run(args...)
}

// itoa converts int to string
func itoa(i int) string {
	return strconv.Itoa(i)
}
