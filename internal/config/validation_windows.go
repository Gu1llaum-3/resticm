//go:build windows
// +build windows

package config

import (
	"os"
)

// checkFileOwner is a no-op on Windows
func checkFileOwner(info os.FileInfo, path string) error {
	// Windows handles permissions differently
	return nil
}
