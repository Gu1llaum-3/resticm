//go:build !windows
// +build !windows

package config

import (
	"os"
	"os/user"
	"strconv"
	"syscall"
)

// checkFileOwner checks if the file is owned by root or current user (Unix)
func checkFileOwner(info os.FileInfo, path string) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if ok {
		currentUID := uint32(os.Getuid())

		// File must be owned by root (0) or current user
		if stat.Uid != 0 && stat.Uid != currentUID {
			// If running via sudo, also accept the original user's ownership
			if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
				if u, err := user.Lookup(sudoUser); err == nil {
					if sudoUID, err := strconv.ParseUint(u.Uid, 10, 32); err == nil {
						if stat.Uid == uint32(sudoUID) {
							// File is owned by the original sudo user, that's fine
							return nil
						}
					}
				}
			}

			return &OwnerError{
				Path:         path,
				FileOwnerUID: stat.Uid,
				ExpectedUID:  currentUID,
				Message:      "configuration file not owned by root or current user",
			}
		}
	}
	return nil
}
