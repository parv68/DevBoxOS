package cmd

import (
	"fmt"
	"sort"

	"github.com/devboxos/devboxos/cli/internal/client"
	"github.com/devboxos/devboxos/cli/internal/output"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config [key] [value]",
	Short: "Manage DevBoxOS configuration",
	Long:  "Get or set DevBoxOS configuration values from the local config file.",
	RunE:  runConfig,
}

func runConfig(cmd *cobra.Command, args []string) error {
	cfg, err := client.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if len(args) == 0 {
		output.Title("DevBoxOS Configuration")
		keys := make([]string, 0, len(cfg))
		for k := range cfg {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if k == "test_key" {
				continue
			}
			fmt.Printf("  %-20s = %s\n", k, cfg[k])
		}
		return nil
	}

	if len(args) == 1 {
		val, ok := cfg[args[0]]
		if !ok {
			return fmt.Errorf("unknown config key: %s", args[0])
		}
		fmt.Printf("%s = %s\n", args[0], val)
		return nil
	}

	cfg[args[0]] = args[1]
	if err := client.SaveConfig(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	output.Success("Set %s = %s", args[0], args[1])
	return nil
}

func init() {
	rootCmd.AddCommand(configCmd)
}
