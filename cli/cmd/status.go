package cmd

import (
	"fmt"
	"os"

	"github.com/devboxos/devboxos/cli/internal/client"
	"github.com/devboxos/devboxos/cli/internal/output"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show running services and environment health",
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

		status, err := conn.Status(dir)
		if err != nil {
			return fmt.Errorf("get status: %w", err)
		}

		output.Status(status)
		return nil
	},
}
