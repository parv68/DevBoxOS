package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/devboxos/devboxos/shared/config"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

var (
	pushAll bool
	pushTag string
)

var pushCmd = &cobra.Command{
	Use:               "push [service]",
	Short:             "Push a service image to a container registry",
	Long:              `Tag and push a service image to a container registry.`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeServiceName,
	RunE:              runPush,
}

func init() {
	pushCmd.Flags().BoolVar(&pushAll, "all", false, "Push all services")
	pushCmd.Flags().StringVarP(&pushTag, "tag", "t", "", "Target image tag (required)")
	rootCmd.AddCommand(pushCmd)
}

func runPush(cmd *cobra.Command, args []string) error {
	if pushTag == "" {
		return fmt.Errorf("--tag is required")
	}

	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}

	ctx := context.Background()

	serviceNames := args
	if pushAll || len(serviceNames) == 0 {
		dir, _ := os.Getwd()
		parser := config.NewParser()
		cfg, parseErr := parser.Parse(dir)
		if parseErr != nil {
			return fmt.Errorf("parse devbox config: %w", parseErr)
		}
		for name := range cfg.Services {
			serviceNames = append(serviceNames, name)
		}
	}

	if len(serviceNames) == 0 {
		return fmt.Errorf("specify a service name or use --all")
	}

	for _, name := range serviceNames {
		containers, err := dockerClient.ContainerList(ctx, container.ListOptions{
			Filters: filters.NewArgs(
				filters.Arg("label", "devboxos.service="+name),
			),
		})
		if err != nil {
			return fmt.Errorf("list containers for %s: %w", name, err)
		}
		if len(containers) == 0 {
			fmt.Fprintf(os.Stderr, "  Warning: no container found for service %s\n", name)
			continue
		}

		sourceImage := containers[0].Image

		fmt.Printf("  Tagging %s -> %s...\n", sourceImage, pushTag)
		if err := dockerClient.ImageTag(ctx, sourceImage, pushTag); err != nil {
			return fmt.Errorf("tag image %s: %w", sourceImage, err)
		}

		fmt.Printf("  Pushing %s...\n", pushTag)
		reader, err := dockerClient.ImagePush(ctx, pushTag, image.PushOptions{
			RegistryAuth: "",
		})
		if err != nil {
			return fmt.Errorf("push %s: %w", pushTag, err)
		}

		decoder := json.NewDecoder(reader)
		for {
			var msg struct {
				Status   string `json:"status"`
				Error    string `json:"error"`
				Progress string `json:"progress"`
				ID       string `json:"id"`
			}
			if err := decoder.Decode(&msg); err != nil {
				if err == io.EOF {
					break
				}
				reader.Close()
				return fmt.Errorf("read push response: %w", err)
			}
			if msg.Error != "" {
				reader.Close()
				return fmt.Errorf("push failed: %s", msg.Error)
			}
			if msg.Status != "" {
				if msg.Progress != "" {
					fmt.Printf("\r  %s: %s %s", msg.ID, msg.Status, msg.Progress)
				} else {
					fmt.Printf("  %s: %s\n", msg.ID, msg.Status)
				}
			}
		}
		reader.Close()
		fmt.Println()
	}

	fmt.Println("✓ Push complete")
	return nil
}
