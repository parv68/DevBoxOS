package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/devboxos/devboxos/shared/runtime/docker"
	"github.com/spf13/cobra"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Tear down the entire DevBoxOS environment",
	Long: `Stop all services and remove all containers, volumes, networks,
and images for the current project. This is a full cleanup that cannot
be undone.

Use --force to skip confirmation.`,
	RunE: runDestroy,
	Args: cobra.MaximumNArgs(1),
}

var destroyForce bool

func init() {
	destroyCmd.Flags().BoolVarP(&destroyForce, "force", "f", false, "Skip confirmation prompt")
	rootCmd.AddCommand(destroyCmd)
}

func runDestroy(cmd *cobra.Command, args []string) error {
	projectName := ""
	if len(args) > 0 {
		projectName = args[0]
	}

	rt := docker.NewDockerRuntime()
	ctx := context.Background()
	if err := rt.Connect(ctx); err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}
	defer rt.Close()

	// Build label filter
	labels := map[string]string{
		"devboxos.managed": "true",
	}
	if projectName != "" {
		labels["devboxos.project"] = projectName
	}

	containers, err := rt.ListContainers(ctx, labels)
	if err != nil {
		return fmt.Errorf("list containers: %w", err)
	}

	if len(containers) == 0 {
		fmt.Println("No DevBoxOS containers found")
		return nil
	}

	if !destroyForce {
		fmt.Printf("This will remove %d container(s). Are you sure? [y/N] ", len(containers))
		var response string
		fmt.Scanln(&response)
		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			fmt.Println("Destroy cancelled")
			return nil
		}
	}

	for _, c := range containers {
		fmt.Printf("Stopping %s...\n", c.Name)
		_ = rt.StopContainer(ctx, c.ID, 10)
	}

	for _, c := range containers {
		fmt.Printf("Removing %s...\n", c.Name)
		if err := rt.RemoveContainer(ctx, c.ID, true); err != nil {
			fmt.Printf("Warning: could not remove %s: %v\n", c.Name, err)
		}
	}

	fmt.Printf("✓ Removed %d container(s)\n", len(containers))
	fmt.Println("Tip: Run 'devbox prune' to clean up orphaned volumes and networks")
	return nil
}
