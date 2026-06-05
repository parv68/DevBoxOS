package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"

	devboxclient "github.com/devboxos/devboxos/cli/internal/client"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "exec <service> <command> [args...]",
	Short: "Execute a command in a running service container",
	Long: `Run a command inside a running service container.

Example:
  devbox exec web /bin/sh
  devbox exec db psql -U postgres
  devbox exec api npm test`,
	Args:              cobra.MinimumNArgs(2),
	ValidArgsFunction: completeServiceName,
	RunE:              runExec,
}

func init() {
	rootCmd.AddCommand(execCmd)
}

func isShellCommand(cmd string) bool {
	switch cmd {
	case "sh", "/bin/sh", "/bin/bash", "bash", "zsh", "fish", "powershell", "pwsh", "cmd", "/bin/zsh":
		return true
	}
	return false
}

func runExec(cmd *cobra.Command, args []string) error {
	serviceName := args[0]
	commandArgs := args[1:]

	projectPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// For non-interactive commands, try gRPC first (handles both runtimes)
	// Skip for shell commands — unary Exec can't attach TTY
	if !isShellCommand(commandArgs[0]) {
		if cl, err := devboxclient.New(); err == nil {
			defer cl.Close()
			stdout, stderr, exitCode, err := cl.Exec(projectPath, serviceName, commandArgs[0], commandArgs[1:])
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
	}

	// Interactive command or engine unavailable: try local Docker
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

			execResp, err := dockerClient.ContainerExecCreate(ctx, containerID, container.ExecOptions{
				Cmd:          commandArgs,
				AttachStdout: true,
				AttachStderr: true,
				AttachStdin:  true,
				Tty:          true,
			})
			if err == nil {
				attachResp, err := dockerClient.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{
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

					_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, attachResp.Reader)
					if err == nil || strings.Contains(err.Error(), "use of closed file") {
						return nil
					}
				}
			}
		}
	}

	// Docker unavailable or no container found: run as host process
	hostCmd := exec.Command(commandArgs[0], commandArgs[1:]...)
	hostCmd.Stdin = os.Stdin
	hostCmd.Stdout = os.Stdout
	hostCmd.Stderr = os.Stderr

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		if hostCmd.Process != nil {
			hostCmd.Process.Signal(os.Interrupt)
		}
	}()

	if err := hostCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("exec failed: %w", err)
	}

	return nil
}
