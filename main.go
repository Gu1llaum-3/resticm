// resticm - Restic Manager
// A modern CLI wrapper for restic backup operations
package main

import (
	"os"

	"resticm/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
