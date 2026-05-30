package cmd

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

var cpCmd = &cobra.Command{
	Use:   "cp <service>:<path> <local-path>",
	Short: "Copy files between service containers and local filesystem",
	Long: `Copy files between running service containers and the local filesystem.

Examples:
  devbox cp web:/app/logs/error.log ./error.log
  devbox cp ./config.json api:/app/config/production.json`,
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: completeServicePath,
	RunE:              runCP,
}

func init() {
	rootCmd.AddCommand(cpCmd)
}

func runCP(cmd *cobra.Command, args []string) error {
	src := args[0]
	dst := args[1]

	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}

	ctx := context.Background()

	srcIsRemote := strings.Contains(src, ":")
	dstIsRemote := strings.Contains(dst, ":")

	if srcIsRemote && !dstIsRemote {
		return copyFromContainer(ctx, dockerClient, src, dst)
	} else if !srcIsRemote && dstIsRemote {
		return copyToContainer(ctx, dockerClient, src, dst)
	}

	return fmt.Errorf("usage: devbox cp <service>:<path> <local-path> or devbox cp <local-path> <service>:<path>")
}

func parseRemoteRef(ref string) (serviceName, path string, err error) {
	parts := strings.SplitN(ref, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid remote reference %q (expected service:path)", ref)
	}
	return parts[0], parts[1], nil
}

func findContainerID(ctx context.Context, dockerClient *client.Client, serviceName string) (string, error) {
	containers, err := dockerClient.ContainerList(ctx, container.ListOptions{
		Filters: filters.NewArgs(
			filters.Arg("label", "devboxos.service="+serviceName),
		),
	})
	if err != nil {
		return "", fmt.Errorf("list containers for service %s: %w", serviceName, err)
	}
	if len(containers) == 0 {
		return "", fmt.Errorf("no running container found for service: %s", serviceName)
	}
	return containers[0].ID, nil
}

func copyFromContainer(ctx context.Context, dockerClient *client.Client, src, dst string) error {
	svcName, srcPath, err := parseRemoteRef(src)
	if err != nil {
		return err
	}

	containerID, err := findContainerID(ctx, dockerClient, svcName)
	if err != nil {
		return err
	}

	reader, _, err := dockerClient.CopyFromContainer(ctx, containerID, srcPath)
	if err != nil {
		return fmt.Errorf("copy from container %s:%s: %w", svcName, srcPath, err)
	}
	defer reader.Close()

	dstInfo, err := os.Stat(dst)
	if err == nil && dstInfo.IsDir() {
		dst = filepath.Join(dst, filepath.Base(srcPath))
	}

	if err := extractTarToFile(reader, dst); err != nil {
		return fmt.Errorf("write to %s: %w", dst, err)
	}

	fmt.Printf("✓ Copied %s:%s → %s\n", svcName, srcPath, dst)
	return nil
}

func copyToContainer(ctx context.Context, dockerClient *client.Client, src, dst string) error {
	svcName, dstPath, err := parseRemoteRef(dst)
	if err != nil {
		return err
	}

	containerID, err := findContainerID(ctx, dockerClient, svcName)
	if err != nil {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source file %s: %w", src, err)
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("stat source file: %w", err)
	}

	pr, pw := io.Pipe()
	tarWriter := tar.NewWriter(pw)

	go func() {
		defer pw.Close()

		header, err := tar.FileInfoHeader(srcInfo, "")
		if err != nil {
			pw.CloseWithError(fmt.Errorf("create tar header: %w", err))
			return
		}
		header.Name = filepath.Base(dstPath)

		if err := tarWriter.WriteHeader(header); err != nil {
			pw.CloseWithError(fmt.Errorf("write tar header: %w", err))
			return
		}

		if _, err := io.Copy(tarWriter, srcFile); err != nil {
			pw.CloseWithError(fmt.Errorf("copy file to tar: %w", err))
			return
		}

		tarWriter.Close()
	}()

	dstDir := filepath.Dir(dstPath)
	if err := dockerClient.CopyToContainer(ctx, containerID, dstDir, pr, container.CopyToContainerOptions{}); err != nil {
		return fmt.Errorf("copy to container %s:%s: %w", svcName, dstPath, err)
	}

	fmt.Printf("✓ Copied %s → %s:%s\n", src, svcName, dstPath)
	return nil
}

func extractTarToFile(reader io.ReadCloser, dst string) error {
	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar archive: %w", err)
		}

		if header.Name == "." || header.Name == "/" {
			continue
		}

		if header.Typeflag == tar.TypeDir {
			continue
		}

		outFile, err := os.Create(dst)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer outFile.Close()

		if _, err := io.Copy(outFile, tarReader); err != nil {
			return fmt.Errorf("write output file: %w", err)
		}
		break
	}

	return nil
}
