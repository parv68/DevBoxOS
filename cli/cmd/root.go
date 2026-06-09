package cmd

import (
	"os"
	"strings"
	"time"

	"github.com/devboxos/devboxos/cli/internal/telemetry"
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

// Execute runs the CLI with anonymous telemetry.
func Execute() {
	// Respect environment opt-out before initializing
	if os.Getenv("DEVBOX_TELEMETRY_DISABLED") == "1" ||
		strings.EqualFold(os.Getenv("DEVBOX_TELEMETRY_DISABLED"), "true") {
		telemetry.Disable()
	}

	telemetry.Init(version)
	defer telemetry.Close()

	start := time.Now()
	cmdName := guessCommandName()
	telemetry.Record("command_start", cmdName, 0, true)

	err := rootCmd.Execute()
	duration := time.Since(start).Milliseconds()
	success := err == nil
	telemetry.Record("command_end", cmdName, duration, success)

	if err != nil {
		os.Exit(1)
	}
}

// guessCommandName extracts the subcommand from os.Args for telemetry.
func guessCommandName() string {
	if len(os.Args) < 2 {
		return "root"
	}
	return os.Args[1]
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
