package cmd

import (
	"fmt"
	"os"

	"github.com/devboxos/devboxos/cli/internal/client"
	"github.com/spf13/cobra"
)

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Tear down and rebuild environment from config",
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

		return conn.Reset(dir)
	},
}
