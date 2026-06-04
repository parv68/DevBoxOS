package cmd

import (
	"context"
	"fmt"
	"time"

	devboxclient "github.com/devboxos/devboxos/cli/internal/client"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

var waitTimeout int

var waitCmd = &cobra.Command{
	Use:   "wait <service> [service...]",
	Short: "Wait for services to become healthy",
	Long: `Block until specified services report healthy, with configurable timeout.

Example:
  devbox wait db --timeout 60
  devbox wait web db redis`,
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: completeServiceName,
	RunE:              runWait,
}

func init() {
	waitCmd.Flags().IntVarP(&waitTimeout, "timeout", "t", 120, "Maximum time to wait in seconds")
	rootCmd.AddCommand(waitCmd)
}

func runWait(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(waitTimeout)*time.Second)
	defer cancel()

	serviceNames := args

	// Try gRPC Status endpoint first (handles both Docker and host runtimes)
	if cl, err := devboxclient.New(); err == nil {
		defer cl.Close()
		for _, name := range serviceNames {
			fmt.Printf("  Waiting for %s...\n", name)
			if err := waitForServiceViaEngine(ctx, cl, name); err != nil {
				return fmt.Errorf("wait for %s: %w", name, err)
			}
			fmt.Printf("  ✔ %s is healthy\n", name)
		}
		return nil
	}

	// Fall back to local Docker
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}

	for _, name := range serviceNames {
		fmt.Printf("  Waiting for %s...\n", name)
		if err := waitForService(ctx, dockerClient, name); err != nil {
			return fmt.Errorf("wait for %s: %w", name, err)
		}
		fmt.Printf("  ✔ %s is healthy\n", name)
	}

	return nil
}

func waitForServiceViaEngine(ctx context.Context, cl *devboxclient.Client, serviceName string) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		resp, err := cl.Status(".")
		if err != nil {
			return err
		}

		for _, svc := range resp.Services {
			if svc.Name == serviceName {
				if svc.Health == "healthy" || (svc.Health == "" && svc.Status == "running") {
					return nil
				}
			}
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for service %s to become healthy", serviceName)
		case <-ticker.C:
		}
	}
}

func waitForService(ctx context.Context, dockerClient *client.Client, serviceName string) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		containers, err := dockerClient.ContainerList(ctx, container.ListOptions{
			Filters: filters.NewArgs(
				filters.Arg("label", "devboxos.service="+serviceName),
			),
		})
		if err != nil {
			return err
		}

		if len(containers) == 0 {
			select {
			case <-ctx.Done():
				return fmt.Errorf("timeout waiting for service %s to start", serviceName)
			case <-ticker.C:
				continue
			}
		}

		info, err := dockerClient.ContainerInspect(ctx, containers[0].ID)
		if err != nil {
			return err
		}

		if info.State.Health != nil {
			if info.State.Health.Status == "healthy" {
				return nil
			}
		} else if info.State.Running {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for service %s to become healthy", serviceName)
		case <-ticker.C:
		}
	}
}
