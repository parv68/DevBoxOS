package cmd

import (
	"fmt"
	"os"

	"github.com/devboxos/devboxos/shared/config"
	"github.com/spf13/cobra"
)

// completeServiceName provides dynamic tab completion for service names.
func completeServiceName(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	parser := config.NewParser()
	cfg, err := parser.Parse(dir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	names := make([]string, 0, len(cfg.Services))
	for name := range cfg.Services {
		names = append(names, name)
	}

	return names, cobra.ShellCompDirectiveNoFileComp
}

// completeServicePath provides tab completion for "service:path" format (for cp).
func completeServicePath(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	parser := config.NewParser()
	cfg, err := parser.Parse(dir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	names := make([]string, 0, len(cfg.Services))
	for name := range cfg.Services {
		names = append(names, fmt.Sprintf("%s:", name))
	}

	return names, cobra.ShellCompDirectiveNoFileComp
}
