package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Display the version, commit hash, and build date of resticm.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("resticm %s\n", version)
		fmt.Printf("  Commit:     %s\n", commit)
		fmt.Printf("  Built:      %s\n", buildDate)
		fmt.Printf("  Go version: %s\n", runtime.Version())
		fmt.Printf("  OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
