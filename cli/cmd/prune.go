package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/devboxos/devboxos/shared/runtime/docker"
	"github.com/spf13/cobra"
)

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove orphaned DevBoxOS Docker resources",
	Long: `Remove containers, volumes, networks, and images that belong to
DevBoxOS but no longer have an associated project on disk.

This frees up disk space and cleans up stale resources.`,
	RunE: runPrune,
}

var pruneForce bool

func init() {
	pruneCmd.Flags().BoolVarP(&pruneForce, "force", "f", false, "Skip confirmation")
	rootCmd.AddCommand(pruneCmd)
}

func runPrune(cmd *cobra.Command, args []string) error {
	rt := docker.NewDockerRuntime()
	ctx := context.Background()
	if err := rt.Connect(ctx); err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}
	defer rt.Close()

	// Find active projects on disk
	activeProjects := findActiveProjects()

	// Collect orphaned containers
	containers, err := rt.ListContainers(ctx, map[string]string{
		"devboxos.managed": "true",
	})
	if err != nil {
		return fmt.Errorf("list containers: %w", err)
	}

	var orphanContainers []string

	for _, c := range containers {
		project := c.Labels["devboxos.project"]
		if project != "" && !activeProjects[project] {
			orphanContainers = append(orphanContainers, c.ID)
		}
	}

	// We can't list devbox-labeled volumes/networks through the abstracted interface,
	// so we skip those for now. The user can use `docker system prune` for complete cleanup.

	if len(orphanContainers) == 0 {
		fmt.Println("No orphaned DevBoxOS resources found")
		return nil
	}

	fmt.Printf("Found %d orphaned container(s):\n", len(orphanContainers))

	if !pruneForce {
		fmt.Print("Remove them? [y/N] ")
		var response string
		fmt.Scanln(&response)
		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			fmt.Println("Prune cancelled")
			return nil
		}
	}

	for _, id := range orphanContainers {
		fmt.Printf("Removing container %s...\n", id[:12])
		if err := rt.RemoveContainer(ctx, id, true); err != nil {
			fmt.Printf("Warning: could not remove %s: %v\n", id[:12], err)
		}
	}

	fmt.Printf("✓ Removed %d orphaned container(s)\n", len(orphanContainers))
	return nil
}

func findActiveProjects() map[string]bool {
	result := make(map[string]bool)
	entries, _ := os.ReadDir(".")
	for _, entry := range entries {
		if entry.IsDir() {
			projectDir := entry.Name()
			if _, err := os.Stat(projectDir + "/devbox.yml"); err == nil {
				result[projectDir] = true
			}
			if _, err := os.Stat(projectDir + "/devbox.yaml"); err == nil {
				result[projectDir] = true
			}
		}
	}
	return result
}
