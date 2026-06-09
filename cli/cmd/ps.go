package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/devboxos/devboxos/cli/internal/client"
	"github.com/devboxos/devboxos/shared/runtime/docker"
	"github.com/spf13/cobra"
)

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "List running DevBoxOS projects",
	Long:  "Show all active DevBoxOS projects and their service status.",
	RunE:  runPs,
}

func init() {
	rootCmd.AddCommand(psCmd)
}

func runPs(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	found := false

	// Try engine first — shows host-runtime projects
	conn, err := client.New()
	if err == nil {
		dir, err := os.Getwd()
		if err == nil {
			status, err := conn.Status(dir)
			if err == nil && status.Status == "running" && len(status.Services) > 0 {
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintf(w, "Project\tServices\tStatus\t\n")
				fmt.Fprintf(w, "-------\t--------\t------\t\n")
				for _, svc := range status.Services {
					fmt.Fprintf(w, "%s\t%s\t%s\t\n", "default", svc.Name, svc.Status)
				}
				w.Flush()
				found = true
			}
		}
		conn.Close()
	}

	// Also check Docker containers
	rt := docker.NewDockerRuntime()
	if err := rt.Connect(ctx); err == nil {
		defer rt.Close()
		containers, err := rt.ListContainers(ctx, map[string]string{
			"devboxos.managed": "true",
		})
		if err == nil && len(containers) > 0 {
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			if !found {
				fmt.Fprintf(w, "Project\tServices\tStatus\t\n")
				fmt.Fprintf(w, "-------\t--------\t------\t\n")
			}
			projects := make(map[string][]string)
			for _, c := range containers {
				project := c.Labels["devboxos.project"]
				if project == "" {
					project = c.Labels["devboxos.service"]
				}
				status := c.Status
				if c.Health != "" {
					status = fmt.Sprintf("%s (%s)", status, c.Health)
				}
				projects[project] = append(projects[project], fmt.Sprintf("  %s\t%s\t", c.Name, status))
			}
			for project, services := range projects {
				fmt.Fprintf(w, "%s\t%s\n", project, services[0])
				for _, s := range services[1:] {
					fmt.Fprintf(w, "\t%s\n", s)
				}
			}
			w.Flush()
			found = true
		}
	}

	if !found {
		fmt.Println("No active DevBoxOS projects")
	}
	return nil
}
