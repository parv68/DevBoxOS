package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/devboxos/devboxos/shared/config"
	"github.com/devboxos/devboxos/shared/runtime"
	"github.com/devboxos/devboxos/shared/runtime/docker"
	"github.com/devboxos/devboxos/shared/types"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build [service]",
	Short: "Build service images from Dockerfile",
	Long:  `Build Docker images for services defined in devbox.yml.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runBuild,
}

var (
	buildNoCache bool
	buildPull    bool
)

func init() {
	buildCmd.Flags().BoolVar(&buildNoCache, "no-cache", false, "Do not use cache when building the image")
	buildCmd.Flags().BoolVar(&buildPull, "pull", false, "Always attempt to pull a newer version of the image")
	rootCmd.AddCommand(buildCmd)
}

func runBuild(cmd *cobra.Command, args []string) error {
	projectPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	parser := config.NewParser()
	cfg, err := parser.Parse(projectPath)
	if err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	rt := docker.NewDockerRuntime()
	ctx := context.Background()
	if err := rt.Connect(ctx); err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}
	defer rt.Close()

	statusChan := make(chan string, 64)
	go func() {
		for msg := range statusChan {
			fmt.Printf("ℹ %s\n", msg)
		}
	}()

	if len(args) > 0 {
		// Build specific service
		serviceName := args[0]
		svc, ok := cfg.Services[serviceName]
		if !ok {
			return fmt.Errorf("service %s not found", serviceName)
		}

		if svc.Build == nil || svc.Build.Context == "" {
			fmt.Printf("Service %s uses pre-built image %s, nothing to build\n", serviceName, svc.Image)
			return nil
		}

		if err := buildService(ctx, rt, serviceName, svc, projectPath, statusChan); err != nil {
			return err
		}
	} else {
		// Build all services with build config
		builtCount := 0
		for name, svc := range cfg.Services {
			if svc.Build == nil || svc.Build.Context == "" {
				continue
			}

			if err := buildService(ctx, rt, name, svc, projectPath, statusChan); err != nil {
				return err
			}
			builtCount++
		}

		if builtCount == 0 {
			fmt.Println("No services with build configuration found")
		} else {
			fmt.Printf("✓ Built %d service(s)\n", builtCount)
		}
	}

	close(statusChan)
	return nil
}

func buildService(ctx context.Context, rt runtime.Runtime, name string, svc types.Service, projectPath string, statusChan chan<- string) error {
	buildCfg := svc.Build
	contextDir := buildCfg.Context
	if !filepath.IsAbs(contextDir) {
		contextDir = filepath.Join(projectPath, contextDir)
	}

	imageName := svc.Image
	if imageName == "" {
		imageName = fmt.Sprintf("devbox-%s:latest", name)
	}

	tags := buildCfg.Tags
	if len(tags) == 0 {
		tags = []string{imageName}
	}

	buildOpts := runtime.BuildConfig{
		ContextDir: contextDir,
		Dockerfile: buildCfg.Dockerfile,
		BuildArgs:  buildCfg.Args,
		Target:     buildCfg.Target,
		Tags:       tags,
	}

	_, err := rt.BuildImage(ctx, buildOpts, statusChan)
	if err != nil {
		return fmt.Errorf("build %s: %w", name, err)
	}

	return nil
}
