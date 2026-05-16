package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "0.1.0-dev"
	commit  = "unknown"
	date    = "unknown"
)

// rootCmd is the base command for the DevBoxOS CLI.
var rootCmd = &cobra.Command{
	Use:   "devbox",
	Short: "DevBoxOS — Universal Development Sandbox",
	Long: `DevBoxOS is a universal development sandbox platform that enables
software teams to spin up fully configured, reproducible development
environments with a single command.

One Command. Any Project. Everywhere.`,
}

// Execute runs the CLI.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(resetCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(configCmd)
}
