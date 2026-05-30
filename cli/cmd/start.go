package cmd

import (
	"fmt"
	"os"

	"github.com/devboxos/devboxos/cli/internal/client"
	"github.com/devboxos/devboxos/cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	startCmd     = &cobra.Command{
		Use:   "start",
		Short: "Start all services defined in devbox.yml",
		Long:  "Start all services defined in devbox.yml in dependency order.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if watchEnabled {
				return runStartWithWatch(cmd, args)
			}
			return runStart(cmd, args)
		},
	}
	watchEnabled bool
)

func init() {
	startCmd.Flags().BoolVarP(&watchEnabled, "watch", "w", false, "Watch files and auto-restart services on changes")
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	conn, err := client.New()
	if err != nil {
		return fmt.Errorf("connect to engine: %w", err)
	}
	defer conn.Close()

	err = conn.Start(dir, func(status, msg string) {
		switch status {
		case "info":
			output.Info("%s", msg)
		case "error":
			output.Error("%s", msg)
		case "warning":
			output.Warning("%s", msg)
		default:
			fmt.Println(msg)
		}
	})
	if err != nil {
		return fmt.Errorf("start: %w", err)
	}

	output.Success("Environment started successfully")
	return nil
}
