package cmd

import (
	"fmt"
	"os"

	"github.com/devboxos/devboxos/cli/internal/client"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs [service]",
	Short: "Stream logs from a service",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}

		service := ""
		if len(args) > 0 {
			service = args[0]
		}

		if service == "" {
			return fmt.Errorf("service name is required. Usage: devbox logs <service>")
		}

		conn, err := client.New()
		if err != nil {
			return fmt.Errorf("connect to engine: %w", err)
		}
		defer conn.Close()

		return conn.Logs(dir, service)
	},
}
