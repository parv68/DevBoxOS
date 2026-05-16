package cmd

import (
	"fmt"

	"github.com/devboxos/devboxos/cli/internal/client"
	"github.com/devboxos/devboxos/cli/internal/output"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config [key] [value]",
	Short: "Manage DevBoxOS configuration",
	Long:  "Get or set DevBoxOS configuration values.",
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := client.New()
		if err != nil {
			return fmt.Errorf("connect to engine: %w", err)
		}
		defer conn.Close()

		if len(args) == 0 {
			// Show all config
			cfg, err := conn.GetConfig()
			if err != nil {
				return fmt.Errorf("get config: %w", err)
			}
			output.Config(cfg)
			return nil
		}

		if len(args) == 1 {
			// Get specific key
			val, err := conn.GetConfigKey(args[0])
			if err != nil {
				return fmt.Errorf("get config key: %w", err)
			}
			fmt.Printf("%s = %s\n", args[0], val)
			return nil
		}

		// Set key=value
		if err := conn.SetConfigKey(args[0], args[1]); err != nil {
			return fmt.Errorf("set config key: %w", err)
		}
		output.Success("Set %s = %s", args[0], args[1])
		return nil
	},
}
