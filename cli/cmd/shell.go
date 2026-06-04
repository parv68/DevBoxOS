package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"

	devboxclient "github.com/devboxos/devboxos/cli/internal/client"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
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

	// Try gRPC Exec first (handles both Docker and host runtimes)
	if cl, err := devboxclient.New(); err == nil {
		defer cl.Close()
		stdout, stderr, exitCode, err := cl.Exec(".", serviceName, "/bin/sh", nil)
		if err == nil {
			if stdout != "" {
				fmt.Print(stdout)
			}
			if stderr != "" {
				fmt.Fprint(os.Stderr, stderr)
			}
			if exitCode != 0 {
				os.Exit(exitCode)
			}
			return nil
		}
	}

	// Engine unavailable: try local Docker
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err == nil {
		ctx := context.Background()

		containers, err := dockerClient.ContainerList(ctx, container.ListOptions{
			Filters: filters.NewArgs(
				filters.Arg("label", "devboxos.service="+serviceName),
			),
		})
		if err == nil && len(containers) > 0 {
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

			if execID != "" {
				attachResp, err := dockerClient.ContainerExecAttach(ctx, execID, container.ExecStartOptions{
					Tty: true,
				})
				if err == nil {
					defer attachResp.Close()

					sigCh := make(chan os.Signal, 1)
					signal.Notify(sigCh, os.Interrupt)
					go func() {
						<-sigCh
						attachResp.Close()
					}()

					fmt.Fprintf(os.Stderr, "Opening shell (%s) in service: %s\n", shellCmdUsed, serviceName)
					_, err = io.Copy(os.Stdout, attachResp.Reader)
					if err == nil {
						return nil
					}
				}
			}
		}
	}

	// Docker unavailable or no container found: open local shell
	localShell := findShell()
	if localShell == "" {
		return fmt.Errorf("could not start shell in service %s (no Docker and no local shell found)", serviceName)
	}

	fmt.Fprintf(os.Stderr, "Opening local shell for service: %s\n", serviceName)
	shCmd := exec.Command(localShell)
	shCmd.Stdin = os.Stdin
	shCmd.Stdout = os.Stdout
	shCmd.Stderr = os.Stderr

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		if shCmd.Process != nil {
			shCmd.Process.Signal(os.Interrupt)
		}
	}()

	if err := shCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("shell failed: %w", err)
	}

	return nil
}

func findShell() string {
	for _, s := range []string{"/bin/bash", "/bin/sh", "bash", "sh", "powershell", "pwsh", "cmd"} {
		if _, err := exec.LookPath(s); err == nil {
			return s
		}
	}
	return ""
}
