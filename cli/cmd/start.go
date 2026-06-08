package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/devboxos/devboxos/cli/internal/client"
	"github.com/devboxos/devboxos/cli/internal/output"
	"github.com/devboxos/devboxos/shared/config"
	"github.com/spf13/cobra"
)

var (
	startCmd     = &cobra.Command{
		Use:   "start",
		Short: "Start all services defined in devbox.yml",
		Long:  "Start all services defined in devbox.yml in dependency order.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if watchEnabled {
				return runStartWithWatch(cmd, args)
			}
			return runStart(cmd, args)
		},
	}
	watchEnabled bool
)

func init() {
	startCmd.Flags().BoolVarP(&watchEnabled, "watch", "w", false, "Watch files and auto-restart services on changes")
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	conn, err := client.New()
	if err != nil {
		return fmt.Errorf("connect to engine: %w", err)
	}
	defer conn.Close()

	err = conn.Start(dir, func(status, msg string) {
		switch status {
		case "info":
			output.Info("%s", msg)
		case "error":
			output.Error("%s", msg)
		case "warning":
			output.Warning("%s", msg)
		default:
			fmt.Println(msg)
		}
	})
	if err != nil {
		return fmt.Errorf("start: %w", err)
	}

	output.Success("Environment started successfully")

	printServiceURLs(dir)
	return nil
}

func printServiceURLs(dir string) {
	parser := config.NewParser()
	cfg, err := parser.Parse(dir)
	if err != nil {
		return
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

		if !hasURLs {
			fmt.Println()
			output.Title("Access URLs")
			hasURLs = true
		}

		fmt.Printf("  %-15s → %s://localhost:%s\n", name, protocol, hostPort)
	}
}
