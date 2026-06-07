package cmd

import (
	"fmt"
	"os"

	"github.com/devboxos/devboxos/cli/internal/client"
	"github.com/devboxos/devboxos/cli/internal/output"
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Stop and restart all services",
	Long:  `Stop all running services and start them again fresh from devbox.yml.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}

		conn, err := client.New()
		if err != nil {
			return fmt.Errorf("connect to engine: %w", err)
		}
		defer conn.Close()

		output.Info("Stopping services...")
		if err := conn.Stop(dir, ""); err != nil {
			output.Warning("Stop failed: %v", err)
		}

		output.Info("Starting services...")
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
			return fmt.Errorf("restart: %w", err)
		}
		output.Success("Environment restarted successfully")

		printServiceURLs(dir)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(restartCmd)
}
