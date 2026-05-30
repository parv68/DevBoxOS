package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/devboxos/devboxos/cli/internal/client"
	"github.com/devboxos/devboxos/shared/config"
	"github.com/devboxos/devboxos/shared/logging"
	"github.com/spf13/cobra"
)

var (
	logsSearch  string
	logsSince   string
	logsTail    int
	logsExport  string
	logsFollow  bool
)

var logsCmd = &cobra.Command{
	Use:               "logs [service]",
	Short:             "View logs from a service",
	Long:              `View, search, and export logs from a service. Supports both live streaming and historical logs.`,
	ValidArgsFunction: completeServiceName,
	RunE:              runLogs,
}

func init() {
	logsCmd.Flags().StringVar(&logsSearch, "search", "", "Search logs for pattern")
	logsCmd.Flags().StringVar(&logsSince, "since", "", "Show logs since (e.g. 1h, 24h, 2d)")
	logsCmd.Flags().IntVar(&logsTail, "tail", 100, "Number of lines to show")
	logsCmd.Flags().StringVar(&logsExport, "export", "", "Export logs to file")
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output")
}

func runLogs(cmd *cobra.Command, args []string) error {
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

	// Parse config to get project name
	parser := config.NewParser()
	cfg, err := parser.Parse(dir)
	if err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	logStore := logging.NewStore(dir)

	if logsExport != "" {
		return exportLogs(logStore, cfg.Name, service, logsExport)
	}

	if logsSearch != "" || logsSince != "" {
		return searchHistoricalLogs(logStore, cfg.Name, service, logsSearch, logsSince)
	}

	if logsFollow {
		conn, err := client.New()
		if err != nil {
			return fmt.Errorf("connect to engine: %w", err)
		}
		defer conn.Close()

		return conn.Logs(dir, service)
	}

	return streamRecentLogs(logStore, cfg.Name, service, logsTail)
}

func streamRecentLogs(store *logging.Store, projectName, service string, tail int) error {
	var since time.Time
	if logsSince != "" {
		d, err := time.ParseDuration(logsSince)
		if err != nil {
			return fmt.Errorf("invalid --since duration: %w", err)
		}
		since = time.Now().Add(-d)
	}

	lines, err := store.Read(projectName, service, since, tail)
	if err != nil {
		return fmt.Errorf("read logs: %w", err)
	}

	if len(lines) == 0 {
		fmt.Println("No logs found")
		return nil
	}

	for _, line := range lines {
		fmt.Println(line)
	}

	return nil
}

func searchHistoricalLogs(store *logging.Store, projectName, service, pattern, sinceStr string) error {
	var since time.Time
	if sinceStr != "" {
		d, err := time.ParseDuration(sinceStr)
		if err != nil {
			return fmt.Errorf("invalid --since duration: %w", err)
		}
		since = time.Now().Add(-d)
	}

	if pattern == "" {
		lines, err := store.Read(projectName, service, since, 0)
		if err != nil {
			return fmt.Errorf("read logs: %w", err)
		}

		if len(lines) == 0 {
			fmt.Println("No logs found")
			return nil
		}

		for _, line := range lines {
			fmt.Println(line)
		}
		return nil
	}

	matches, err := store.Search(projectName, service, pattern, since)
	if err != nil {
		return fmt.Errorf("search logs: %w", err)
	}

	if len(matches) == 0 {
		fmt.Printf("No matches for pattern: %s\n", pattern)
		return nil
	}

	for _, line := range matches {
		fmt.Println(line)
	}

	fmt.Printf("\n✓ Found %d matches\n", len(matches))
	return nil
}

func exportLogs(store *logging.Store, projectName, service, outputPath string) error {
	if !filepath.IsAbs(outputPath) {
		cwd, _ := os.Getwd()
		outputPath = filepath.Join(cwd, outputPath)
	}

	if err := store.Export(projectName, service, outputPath); err != nil {
		return fmt.Errorf("export logs: %w", err)
	}

	fmt.Printf("✓ Logs exported to %s\n", outputPath)
	return nil
}
