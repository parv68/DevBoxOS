package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/devboxos/devboxos/engine/internal/runtime"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// DockerRuntime implements the runtime.Runtime interface using Docker.
type DockerRuntime struct {
	cli *client.Client
}

// NewDockerRuntime creates a new Docker runtime client.
func NewDockerRuntime() *DockerRuntime {
	return &DockerRuntime{}
}

// Connect establishes a connection to the Docker daemon.
func (d *DockerRuntime) Connect(ctx context.Context) error {
	var opts []client.Opt

	// On Windows, try TCP first (Docker Desktop exposes TCP on localhost)
	if os.PathSeparator == '\\' {
		// Try TCP connection first for Docker Desktop on Windows
		opts = append(opts, client.WithHost("tcp://127.0.0.1:2375"))
		opts = append(opts, client.WithAPIVersionNegotiation())

		cli, err := client.NewClientWithOpts(opts...)
		if err == nil {
			_, pingErr := cli.Ping(ctx)
			if pingErr == nil {
				d.cli = cli
				return nil
			}
		}

		// Fall back to named pipe
		opts = []client.Opt{
			client.WithHost("npipe:////./pipe/docker_engine"),
			client.WithAPIVersionNegotiation(),
		}
	} else {
		opts = append(opts, client.FromEnv)
		opts = append(opts, client.WithAPIVersionNegotiation())
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return fmt.Errorf("create docker client: %w", err)
	}

	_, err = cli.Ping(ctx)
	if err != nil {
		cli.Close()
		return fmt.Errorf("ping docker daemon: %w\n\nMake sure Docker is running.", err)
	}

	d.cli = cli
	return nil
}

// Close closes the Docker client connection.
func (d *DockerRuntime) Close() error {
	if d.cli != nil {
		return d.cli.Close()
	}
	return nil
}

// Check verifies Docker is accessible.
func (d *DockerRuntime) Check(ctx context.Context) error {
	if d.cli == nil {
		return fmt.Errorf("docker client not connected")
	}
	_, err := d.cli.Ping(ctx)
	return err
}

// PullImage pulls a container image.
func (d *DockerRuntime) PullImage(ctx context.Context, imageName string) error {
	reader, err := d.cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pull image %s: %w", imageName, err)
	}
	defer reader.Close()

	_, err = io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("read pull response for %s: %w", imageName, err)
	}

	return nil
}

// CreateContainer creates a new container.
func (d *DockerRuntime) CreateContainer(ctx context.Context, cfg runtime.ContainerConfig) (string, error) {
	portBindings := make(nat.PortMap)
	exposedPorts := make(nat.PortSet)

	for hostPort, containerPort := range cfg.Ports {
		port := nat.Port(containerPort + "/tcp")
		exposedPorts[port] = struct{}{}
		portBindings[port] = []nat.PortBinding{{
			HostIP:   "0.0.0.0",
			HostPort: hostPort,
		}}
	}

	env := make([]string, 0, len(cfg.Env))
	for k, v := range cfg.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	binds := make([]string, 0, len(cfg.Volumes))
	for hostPath, containerPath := range cfg.Volumes {
		binds = append(binds, fmt.Sprintf("%s:%s", hostPath, containerPath))
	}

	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		Binds:        binds,
	}

	if cfg.Memory != "" {
		memBytes, err := parseMemory(cfg.Memory)
		if err != nil {
			return "", fmt.Errorf("parse memory limit: %w", err)
		}
		hostConfig.Resources.Memory = memBytes
	}
	if cfg.CPU != "" {
		nanoCPU, err := parseCPU(cfg.CPU)
		if err != nil {
			return "", fmt.Errorf("parse CPU limit: %w", err)
		}
		hostConfig.Resources.NanoCPUs = nanoCPU
	}

	if cfg.ReadOnly {
		hostConfig.ReadonlyRootfs = true
	}

	containerConfig := &container.Config{
		Image:        cfg.Image,
		Cmd:          cfg.Command,
		WorkingDir:   cfg.WorkingDir,
		Env:          env,
		ExposedPorts: exposedPorts,
		Labels:       cfg.Labels,
	}

	resp, err := d.cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, cfg.Name)
	if err != nil {
		return "", fmt.Errorf("create container %s: %w", cfg.Name, err)
	}

	if cfg.Network != "" {
		err = d.cli.NetworkConnect(ctx, cfg.Network, resp.ID, &network.EndpointSettings{})
		if err != nil {
			return "", fmt.Errorf("connect to network %s: %w", cfg.Network, err)
		}
	}

	return resp.ID, nil
}

// StartContainer starts a container.
func (d *DockerRuntime) StartContainer(ctx context.Context, id string) error {
	return d.cli.ContainerStart(ctx, id, container.StartOptions{})
}

// StopContainer stops a container.
func (d *DockerRuntime) StopContainer(ctx context.Context, id string, timeoutSeconds int) error {
	timeout := timeoutSeconds
	return d.cli.ContainerStop(ctx, id, container.StopOptions{
		Timeout: &timeout,
	})
}

// RemoveContainer removes a container.
func (d *DockerRuntime) RemoveContainer(ctx context.Context, id string, force bool) error {
	return d.cli.ContainerRemove(ctx, id, container.RemoveOptions{
		Force: force,
	})
}

// GetContainerInfo returns information about a container.
func (d *DockerRuntime) GetContainerInfo(ctx context.Context, id string) (runtime.ContainerInfo, error) {
	info, err := d.cli.ContainerInspect(ctx, id)
	if err != nil {
		return runtime.ContainerInfo{}, fmt.Errorf("inspect container %s: %w", id, err)
	}

	var ports []runtime.PortMapping
	for port, bindings := range info.NetworkSettings.Ports {
		for _, binding := range bindings {
			ports = append(ports, runtime.PortMapping{
				HostIP:        binding.HostIP,
				HostPort:      binding.HostPort,
				ContainerPort: port.Port(),
				Protocol:      port.Proto(),
			})
		}
	}

	var networks []string
	for name := range info.NetworkSettings.Networks {
		networks = append(networks, name)
	}

	health := "none"
	if info.State.Health != nil {
		health = info.State.Health.Status
	}

	return runtime.ContainerInfo{
		ID:        info.ID,
		Name:      info.Name,
		Image:     info.Config.Image,
		Status:    info.State.Status,
		Ports:     ports,
		Networks:  networks,
		StartedAt: info.State.StartedAt,
		Health:    health,
	}, nil
}

// ListContainers returns containers matching the given labels.
func (d *DockerRuntime) ListContainers(ctx context.Context, labels map[string]string) ([]runtime.ContainerInfo, error) {
	filterArgs := filters.NewArgs()
	for k, v := range labels {
		filterArgs.Add("label", fmt.Sprintf("%s=%s", k, v))
	}

	containers, err := d.cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	var result []runtime.ContainerInfo
	for _, c := range containers {
		var ports []runtime.PortMapping
		for _, p := range c.Ports {
			ports = append(ports, runtime.PortMapping{
				HostPort:      strconv.FormatInt(int64(p.PublicPort), 10),
				ContainerPort: strconv.FormatInt(int64(p.PrivatePort), 10),
				Protocol:      p.Type,
			})
		}

		result = append(result, runtime.ContainerInfo{
			ID:     c.ID,
			Name:   c.Names[0],
			Image:  c.Image,
			Status: c.State,
			Ports:  ports,
		})
	}

	return result, nil
}

// StreamLogs streams logs from a container.
func (d *DockerRuntime) StreamLogs(ctx context.Context, id string, opts runtime.LogOptions) (io.ReadCloser, error) {
	tail := "all"
	if opts.Tail > 0 {
		tail = strconv.Itoa(opts.Tail)
	}

	reader, err := d.cli.ContainerLogs(ctx, id, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     opts.Follow,
		Tail:       tail,
		Since:      opts.Since,
		Timestamps: true,
	})
	if err != nil {
		return nil, fmt.Errorf("stream logs for %s: %w", id, err)
	}
	return reader, nil
}

// CreateNetwork creates a new Docker network.
func (d *DockerRuntime) CreateNetwork(ctx context.Context, name string) error {
	_, err := d.cli.NetworkCreate(ctx, name, network.CreateOptions{
		Labels: map[string]string{
			"devboxos.managed": "true",
		},
	})
	if err != nil {
		return fmt.Errorf("create network %s: %w", name, err)
	}
	return nil
}

// RemoveNetwork removes a Docker network.
func (d *DockerRuntime) RemoveNetwork(ctx context.Context, name string) error {
	return d.cli.NetworkRemove(ctx, name)
}

// NetworkExists checks if a network exists.
func (d *DockerRuntime) NetworkExists(ctx context.Context, name string) (bool, error) {
	networks, err := d.cli.NetworkList(ctx, network.ListOptions{
		Filters: filters.NewArgs(filters.KeyValuePair{
			Key:   "name",
			Value: name,
		}),
	})
	if err != nil {
		return false, fmt.Errorf("list networks: %w", err)
	}
	return len(networks) > 0, nil
}

// CreateVolume creates a named Docker volume.
func (d *DockerRuntime) CreateVolume(ctx context.Context, name string) error {
	_, err := d.cli.VolumeCreate(ctx, volume.CreateOptions{
		Name: name,
		Labels: map[string]string{
			"devboxos.managed": "true",
		},
	})
	if err != nil {
		return fmt.Errorf("create volume %s: %w", name, err)
	}
	return nil
}

// RemoveVolume removes a named Docker volume.
func (d *DockerRuntime) RemoveVolume(ctx context.Context, name string) error {
	return d.cli.VolumeRemove(ctx, name, false)
}

// VolumeExists checks if a volume exists.
func (d *DockerRuntime) VolumeExists(ctx context.Context, name string) (bool, error) {
	_, err := d.cli.VolumeInspect(ctx, name)
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("inspect volume %s: %w", name, err)
	}
	return true, nil
}

// parseMemory parses a memory string like "512m" or "1g" into bytes.
func parseMemory(s string) (int64, error) {
	var multiplier int64 = 1
	numStr := s
	switch {
	case len(s) > 1 && (s[len(s)-1] == 'g' || s[len(s)-1] == 'G'):
		multiplier = 1024 * 1024 * 1024
		numStr = s[:len(s)-1]
	case len(s) > 1 && (s[len(s)-1] == 'm' || s[len(s)-1] == 'M'):
		multiplier = 1024 * 1024
		numStr = s[:len(s)-1]
	case len(s) > 1 && (s[len(s)-1] == 'k' || s[len(s)-1] == 'K'):
		multiplier = 1024
		numStr = s[:len(s)-1]
	case len(s) > 1 && (s[len(s)-1] == 'b' || s[len(s)-1] == 'B'):
		numStr = s[:len(s)-1]
	}

	var val int64
	_, err := fmt.Sscanf(numStr, "%d", &val)
	if err != nil {
		return 0, fmt.Errorf("parse memory value %q: %w", s, err)
	}
	return val * multiplier, nil
}

// parseCPU parses a CPU string like "0.5" into nano CPUs.
func parseCPU(s string) (int64, error) {
	var val float64
	_, err := fmt.Sscanf(s, "%f", &val)
	if err != nil {
		return 0, fmt.Errorf("parse CPU value %q: %w", s, err)
	}
	return int64(val * 1e9), nil
}
