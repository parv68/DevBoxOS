package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show DevBoxOS version",
	Long:  "Display the installed DevBoxOS version, build commit, and build date.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("DevBoxOS %s (commit: %s, built: %s)\n", version, commit, date)
	},
}
