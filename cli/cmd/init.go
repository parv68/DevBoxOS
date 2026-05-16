package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/devboxos/devboxos/cli/internal/autodetect"
	"github.com/devboxos/devboxos/cli/internal/output"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new DevBoxOS project",
	Long:  "Generate a devbox.yml configuration file by scanning the current project.",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}

		// Check if devbox.yml already exists
		configPath := filepath.Join(dir, "devbox.yml")
		if _, err := os.Stat(configPath); err == nil {
			output.Warning("devbox.yml already exists in %s", dir)
			return nil
		}

		output.Info("Scanning project...")

		// Auto-detect project configuration
		cfg, err := autodetect.AutoDetect(dir)
		if err != nil {
			return fmt.Errorf("auto-detect: %w", err)
		}

		// Write config file
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("marshal config: %w", err)
		}

		if err := os.WriteFile(configPath, data, 0644); err != nil {
			return fmt.Errorf("write config file: %w", err)
		}

		output.Success("Created devbox.yml in %s", dir)
		output.Info("Detected %d service(s):", len(cfg.Services))
		for name := range cfg.Services {
			output.Info("  - %s", name)
		}
		output.Info("Run 'devbox start' to launch your environment")
		return nil
	},
}
