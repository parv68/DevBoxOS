package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/devboxos/devboxos/shared/config"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate devbox.yml configuration",
	Long:  "Check your devbox.yml for syntax errors, missing fields, port conflicts, and other issues.",
	RunE:  runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	parser := config.NewParser()
	cfg, err := parser.Parse(dir)
	if err != nil {
		fmt.Printf("✗ Configuration error: %v\n", err)
		return fmt.Errorf("invalid configuration: %w", err)
	}

	validator, err := config.NewValidator()
	if err != nil {
		fmt.Printf("⚠ Schema validation not available: %v\n", err)
	}

	if validator != nil {
		if errs := validator.Validate(cfg); len(errs) > 0 {
			for _, e := range errs {
				fmt.Printf("✗ %s\n", e)
			}
			return fmt.Errorf("configuration has %d validation error(s)", len(errs))
		}
	}

	var issues []string

	if cfg.Name == "" {
		issues = append(issues, "Project name is empty")
	}

	for name, svc := range cfg.Services {
		if svc.Image == "" && (svc.Build == nil || svc.Build.Context == "") {
			issues = append(issues, fmt.Sprintf("Service %q: must specify 'image' or 'build'", name))
		}

		if svc.Port != "" && svc.Ports != nil && len(svc.Ports) > 0 {
			issues = append(issues, fmt.Sprintf("Service %q: specify 'port' or 'ports', not both", name))
		}

		if svc.EnvFile != "" {
			envPath := svc.EnvFile
			if !filepath.IsAbs(envPath) {
				envPath = filepath.Join(dir, envPath)
			}
			if _, err := os.Stat(envPath); os.IsNotExist(err) {
				issues = append(issues, fmt.Sprintf("Service %q: env_file %s not found", name, svc.EnvFile))
			}
		}

		if svc.Build != nil && svc.Build.Context != "" {
			buildDir := svc.Build.Context
			if !filepath.IsAbs(buildDir) {
				buildDir = filepath.Join(dir, buildDir)
			}
			if _, err := os.Stat(buildDir); os.IsNotExist(err) {
				issues = append(issues, fmt.Sprintf("Service %q: build context %s not found", name, svc.Build.Context))
			}
			if svc.Build.Dockerfile != "" {
				dfPath := filepath.Join(buildDir, svc.Build.Dockerfile)
				if _, err := os.Stat(dfPath); os.IsNotExist(err) {
					issues = append(issues, fmt.Sprintf("Service %q: Dockerfile %s not found", name, svc.Build.Dockerfile))
				}
			}
		}

		for _, dep := range svc.DependsOn {
			if _, ok := cfg.Services[dep]; !ok {
				issues = append(issues, fmt.Sprintf("Service %q: depends_on %q does not match any service", name, dep))
			}
		}
	}

	usedPorts := make(map[string]string)
	for name, svc := range cfg.Services {
		ports := svc.Ports
		if svc.Port != "" {
			ports = append(ports, svc.Port)
		}
		for _, p := range ports {
			parts := strings.SplitN(p, ":", 2)
			hostPort := parts[0]
			if existing, ok := usedPorts[hostPort]; ok {
				issues = append(issues, fmt.Sprintf("Port %s conflict: %q and %q both use it", hostPort, existing, name))
			}
			usedPorts[hostPort] = name
		}
	}

	if len(issues) == 0 {
		fmt.Printf("✓ Configuration is valid (%d services)\n", len(cfg.Services))
		return nil
	}

	fmt.Printf("Configuration has %d issue(s):\n", len(issues))
	for _, issue := range issues {
		fmt.Printf("  ✗ %s\n", issue)
	}

	return fmt.Errorf("configuration has %d issue(s)", len(issues))
}
