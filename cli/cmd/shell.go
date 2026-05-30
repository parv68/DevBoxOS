package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/spf13/cobra"
)

var shellCmd = &cobra.Command{
	Use:   "shell <service>",
	Short: "Open an interactive shell in a running service container",
	Long: `Open an interactive shell inside a running service container.

Uses bash if available, falls back to sh.

Example:
  devbox shell web
  devbox shell db`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeServiceName,
	RunE:              runShell,
}

func init() {
	rootCmd.AddCommand(shellCmd)
}

func runShell(cmd *cobra.Command, args []string) error {
	serviceName := args[0]

	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}

	ctx := context.Background()

	containers, err := dockerClient.ContainerList(ctx, container.ListOptions{
		Filters: filters.NewArgs(
			filters.Arg("label", "devboxos.service="+serviceName),
		),
	})
	if err != nil {
		return fmt.Errorf("list containers: %w", err)
	}

	if len(containers) == 0 {
		return fmt.Errorf("no running container found for service: %s", serviceName)
	}

	containerID := containers[0].ID

	shells := []string{"/bin/bash", "/bin/sh"}
	var execID string
	var shellCmdUsed string

	for _, shell := range shells {
		execResp, err := dockerClient.ContainerExecCreate(ctx, containerID, container.ExecOptions{
			Cmd:          []string{shell},
			AttachStdout: true,
			AttachStderr: true,
			AttachStdin:  true,
			Tty:          true,
		})
		if err != nil {
			continue
		}
		execID = execResp.ID
		shellCmdUsed = shell
		break
	}

	if execID == "" {
		return fmt.Errorf("could not start shell in container %s (tried bash, sh)", serviceName)
	}

	attachResp, err := dockerClient.ContainerExecAttach(ctx, execID, container.ExecStartOptions{
		Tty: true,
	})
	if err != nil {
		return fmt.Errorf("attach to shell: %w", err)
	}
	defer attachResp.Close()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		attachResp.Close()
	}()

	fmt.Fprintf(os.Stderr, "Opening shell (%s) in service: %s\n", shellCmdUsed, serviceName)
	_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, attachResp.Reader)
	if err != nil {
		return fmt.Errorf("shell session: %w", err)
	}

	return nil
}
