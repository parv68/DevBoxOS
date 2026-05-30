package cmd

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/devboxos/devboxos/shared/config"
	"github.com/spf13/cobra"
)

var urlCmd = &cobra.Command{
	Use:   "url",
	Short: "Show accessible URLs for all services",
	Long: `Display all accessible URLs for services with port mappings.

Example:
  devbox url
    web     → http://localhost:8080
    api     → http://localhost:3000`,
	RunE: runURL,
}

func init() {
	rootCmd.AddCommand(urlCmd)
}

func runURL(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	parser := config.NewParser()
	cfg, err := parser.Parse(dir)
	if err != nil {
		return fmt.Errorf("parse devbox config: %w", err)
	}

	names := make([]string, 0, len(cfg.Services))
	for name := range cfg.Services {
		names = append(names, name)
	}
	sort.Strings(names)

	hasURLs := false
	for _, name := range names {
		svc := cfg.Services[name]
		portStr := svc.Port
		if portStr == "" && len(svc.Ports) > 0 {
			portStr = svc.Ports[0]
		}
		if portStr == "" {
			continue
		}

		hostPort := extractHostPort(portStr)
		if hostPort == "" {
			continue
		}

		protocol := svc.Protocol
		if protocol == "" {
			protocol = "http"
		}

		fmt.Printf("  %-15s → %s://localhost:%s\n", name, protocol, hostPort)
		hasURLs = true
	}

	if !hasURLs {
		fmt.Println("No services with port mappings found")
	}

	return nil
}

func extractHostPort(portStr string) string {
	parts := strings.SplitN(portStr, ":", 2)
	if len(parts) == 2 {
		portStr = parts[0]
	}
	_, err := strconv.Atoi(portStr)
	if err != nil {
		return ""
	}
	return portStr
}
