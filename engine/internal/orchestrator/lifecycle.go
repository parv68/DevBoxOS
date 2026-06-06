package orchestrator

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/devboxos/devboxos/shared/runtime"
	"github.com/devboxos/devboxos/shared/secrets"
	"github.com/devboxos/devboxos/shared/types"
)

// Lifecycle manages the start/stop/restart lifecycle of services.
type Lifecycle struct {
	resolver *secrets.Resolver
}

// NewLifecycle creates a new service lifecycle manager.
func NewLifecycle(resolver *secrets.Resolver) *Lifecycle {
	return &Lifecycle{resolver: resolver}
}

// StartService starts a single service using the given runtime.
func (l *Lifecycle) StartService(ctx context.Context, rt runtime.Runtime, name string, svc types.Service, networkName string, projectPath string, statusChan chan<- string) (string, error) {
	// Build container config
	cfg := runtime.ContainerConfig{
		Name:       fmt.Sprintf("devbox-%s", name),
		Image:      svc.Image,
		Command:    parseCommand(svc.Command),
		WorkingDir: svc.WorkingDir,
		Env:        svc.Env,
		Network:    networkName,
		Labels: map[string]string{
			"devboxos.project":    name,
			"devboxos.service":    name,
			"devboxos.managed":    "true",
		},
	}

	// Set ports
	if svc.Port != "" {
		cfg.Ports = make(map[string]string)
		// Parse "host:container" or just "container"
		hostPort := svc.Port
		containerPort := svc.Port
		for i := len(svc.Port) - 1; i >= 0; i-- {
			if svc.Port[i] == ':' {
				hostPort = svc.Port[:i]
				containerPort = svc.Port[i+1:]
				break
			}
		}
		cfg.Ports[hostPort] = containerPort
	}

	// Set volumes
	if svc.Data != "" {
		cfg.Volumes = make(map[string]string)
		cfg.Volumes[svc.Data] = "/data"
	}
	for _, vol := range svc.Volumes {
		if cfg.Volumes == nil {
			cfg.Volumes = make(map[string]string)
		}
		// Parse "host:container" format
		hostPath := vol
		containerPath := vol
		for i := len(vol) - 1; i >= 0; i-- {
			if vol[i] == ':' {
				hostPath = vol[:i]
				containerPath = vol[i+1:]
				break
			}
		}
		cfg.Volumes[hostPath] = containerPath
	}

	// Set resource limits
	if svc.Resources != nil {
		cfg.Memory = svc.Resources.Memory
		cfg.CPU = svc.Resources.CPU
	}

	// Set security
	if svc.Security != nil {
		cfg.ReadOnly = svc.Security.ReadOnly
	}

	// Resolve secrets and inject as environment variables
	if l.resolver != nil && len(svc.Secrets) > 0 {
		if cfg.Env == nil {
			cfg.Env = make(map[string]string)
		}
		for _, secretRef := range svc.Secrets {
			value, err := l.resolver.Resolve(secretRef)
			if err != nil {
				return "", fmt.Errorf("resolve secret %s for %s: %w", secretRef.Name, name, err)
			}
			cfg.Env[secretRef.Name] = value
		}
	}

	// Build image if build config is defined
	if svc.Build != nil && svc.Build.Context != "" {
		contextDir := svc.Build.Context
		if !filepath.IsAbs(contextDir) {
			contextDir = filepath.Join(projectPath, contextDir)
		}

		buildCfg := runtime.BuildConfig{
			ContextDir: contextDir,
			Dockerfile: svc.Build.Dockerfile,
			BuildArgs:  svc.Build.Args,
			Target:     svc.Build.Target,
		}

		builtImage, err := rt.BuildImage(ctx, buildCfg, statusChan)
		if err != nil {
			return "", fmt.Errorf("build image for %s: %w", name, err)
		}
		cfg.Image = builtImage
	} else if svc.Image != "" {
		// Pull image if no build config
		if err := rt.PullImage(ctx, svc.Image); err != nil {
			return "", fmt.Errorf("pull image %s: %w", svc.Image, err)
		}
	}

	// Remove existing container if present
	existingContainers, _ := rt.ListContainers(ctx, map[string]string{
		"devboxos.service": name,
	})
	for _, existing := range existingContainers {
		_ = rt.RemoveContainer(ctx, existing.ID, true)
	}

	// Create container
	containerID, err := rt.CreateContainer(ctx, cfg)
	if err != nil {
		return "", fmt.Errorf("create container for %s: %w", name, err)
	}

	// Start container
	if err := rt.StartContainer(ctx, containerID); err != nil {
		return "", fmt.Errorf("start container for %s: %w", name, err)
	}

	return containerID, nil
}

// StopService stops a single service.
func (l *Lifecycle) StopService(ctx context.Context, rt runtime.Runtime, containerID string, gracePeriod int) error {
	if err := rt.StopContainer(ctx, containerID, gracePeriod); err != nil {
		return fmt.Errorf("stop container %s: %w", containerID, err)
	}
	return nil
}

// RemoveService removes a service container.
func (l *Lifecycle) RemoveService(ctx context.Context, rt runtime.Runtime, containerID string) error {
	return rt.RemoveContainer(ctx, containerID, true)
}

// WaitForHealthy waits for a service to become healthy.
func (l *Lifecycle) WaitForHealthy(ctx context.Context, rt runtime.Runtime, containerID string, svc types.Service) error {
	if svc.Healthcheck == nil {
		// No health check defined, assume healthy after brief delay
		time.Sleep(2 * time.Second)
		return nil
	}

	// Parse start period
	startPeriod := 30 * time.Second
	if svc.Healthcheck.StartPeriod != "" {
		d, err := time.ParseDuration(svc.Healthcheck.StartPeriod)
		if err == nil {
			startPeriod = d
		}
	}

	deadline := time.Now().Add(startPeriod)
	interval := 5 * time.Second
	if svc.Healthcheck.Interval != "" {
		d, err := time.ParseDuration(svc.Healthcheck.Interval)
		if err == nil {
			interval = d
		}
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Until(deadline)):
			return fmt.Errorf("service did not become healthy within %s", startPeriod)
		case <-ticker.C:
			info, err := rt.GetContainerInfo(ctx, containerID)
			if err != nil {
				continue
			}
			if info.Health == "healthy" {
				return nil
			}
			if info.Status == "exited" || info.Status == "dead" {
				return fmt.Errorf("container exited with status %s", info.Status)
			}
		}
	}
}

// parseCommand splits a command string into args.
func parseCommand(cmd string) []string {
	if cmd == "" {
		return nil
	}
	// Simple split — a real implementation would handle quotes
	return []string{"sh", "-c", cmd}
}
