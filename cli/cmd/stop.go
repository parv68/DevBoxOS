package cmd

import (
	"fmt"
	"os"

	"github.com/devboxos/devboxos/cli/internal/client"
	"github.com/devboxos/devboxos/cli/internal/output"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop [service]",
	Short: "Stop all services or a specific service",
	Long: `Stop running services in the DevBoxOS environment.

If no service name is given, all services are stopped.
Services are stopped in reverse-dependency order (dependents first).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}

		service := ""
		if len(args) > 0 {
			service = args[0]
		}

		conn, err := client.New()
		if err != nil {
			return fmt.Errorf("connect to engine: %w", err)
		}
		defer conn.Close()

		if err := conn.Stop(dir, service); err != nil {
			return fmt.Errorf("stop: %w", err)
		}

		output.Success("Environment stopped")
		return nil
	},
}
