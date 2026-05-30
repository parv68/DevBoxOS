package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/devboxos/devboxos/shared/config"
	"github.com/devboxos/devboxos/shared/secrets"
	"github.com/devboxos/devboxos/shared/types"
	"github.com/spf13/cobra"
)

var (
	envReveal bool
	envSvc    string
)

var envCmd = &cobra.Command{
	Use:   "env [service]",
	Short: "Show environment variables for services",
	Long: `Display environment variables (including resolved secrets) for services.

By default, secret values are masked. Use --reveal to show them.

Example:
  devbox env
  devbox env web
  devbox env web --reveal`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeServiceName,
	RunE:              runEnv,
}

func init() {
	envCmd.Flags().BoolVar(&envReveal, "reveal", false, "Show secret values (masked by default)")
	rootCmd.AddCommand(envCmd)
}

func runEnv(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	parser := config.NewParser()
	cfg, err := parser.Parse(dir)
	if err != nil {
		return fmt.Errorf("parse devbox config: %w", err)
	}

	var resolver *secrets.Resolver
	keyPath := filepath.Join(dir, ".devbox", "secrets.key")
	storePath := filepath.Join(dir, ".devbox", "secrets.enc")
	if _, err := os.Stat(keyPath); err == nil {
		r, err := secrets.NewResolver(dir, keyPath, storePath)
		if err == nil {
			resolver = r
		}
	}

	svcName := ""
	if len(args) > 0 {
		svcName = args[0]
	}

	names := make([]string, 0, len(cfg.Services))
	for name := range cfg.Services {
		names = append(names, name)
	}
	sort.Strings(names)

	hasOutput := false
	for _, name := range names {
		if svcName != "" && name != svcName {
			continue
		}
		svc := cfg.Services[name]

		fmt.Printf("\n  %s:\n", name)

		envVars := resolveServiceEnv(svc, resolver)
		if len(envVars) == 0 {
			fmt.Println("    (no environment variables defined)")
		} else {
			for k, v := range envVars {
				if !envReveal {
					v = maskValue(v)
				}
				fmt.Printf("    %s=%s\n", k, v)
			}
		}
		hasOutput = true
	}

	if !hasOutput {
		if svcName != "" {
			return fmt.Errorf("service %q not found in devbox.yml", svcName)
		}
		fmt.Println("No services defined")
	}

	fmt.Println()
	return nil
}

func resolveServiceEnv(svc types.Service, resolver *secrets.Resolver) map[string]string {
	result := make(map[string]string)

	for k, v := range svc.Env {
		result[k] = v
	}

	if resolver != nil {
		for _, ref := range svc.Secrets {
			val, err := resolver.Resolve(ref)
			if err == nil {
				result[ref.Name] = val
			}
		}
	}

	return result
}

func maskValue(v string) string {
	if len(v) <= 4 {
		return "****"
	}
	return v[:2] + "****" + v[len(v)-2:]
}
