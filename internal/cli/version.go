package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Version information (set by build)
	Version   = "0.1.0-dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print the version, build date, and git commit information for Panka.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Panka Version: %s\n", Version)
		fmt.Printf("Git Commit:    %s\n", GitCommit)
		fmt.Printf("Build Date:    %s\n", BuildDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

