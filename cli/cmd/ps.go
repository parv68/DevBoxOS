package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

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
	rt := docker.NewDockerRuntime()
	ctx := context.Background()
	if err := rt.Connect(ctx); err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}
	defer rt.Close()

	containers, err := rt.ListContainers(ctx, map[string]string{
		"devboxos.managed": "true",
	})
	if err != nil {
		return fmt.Errorf("list containers: %w", err)
	}

	if len(containers) == 0 {
		fmt.Println("No active DevBoxOS projects")
		return nil
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
		projects[project] = append(projects[project], fmt.Sprintf("  %s\t%s", c.Name, status))
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Project\tServices\t\n")
	fmt.Fprintf(w, "-------\t--------\t\n")
	for project, services := range projects {
		fmt.Fprintf(w, "%s\t%s\t\n", project, services[0])
		for _, s := range services[1:] {
			fmt.Fprintf(w, "\t%s\t\n", s)
		}
	}
	w.Flush()
	return nil
}
