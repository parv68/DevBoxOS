package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/devboxos/devboxos/shared/platform"
	"github.com/devboxos/devboxos/shared/runtime"
	"github.com/docker/docker/api/types/build"
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

	switch platform.Detect() {
	case platform.OSWindows:
		// Try TCP first (Docker Desktop on Windows)
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
			client.WithHost(platform.DockerSocketPath()),
			client.WithAPIVersionNegotiation(),
		}

	case platform.OSDarwin, platform.OSLinux:
		// Unix socket (default)
		opts = append(opts, client.WithHost(platform.DockerSocketPath()))
		opts = append(opts, client.WithAPIVersionNegotiation())

	default:
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

// VerifyImage verifies a container image signature using external Cosign binary.
// Returns nil if cosign is not installed (verification skipped gracefully).
// Returns an error if cosign is installed and verification fails.
func (d *DockerRuntime) VerifyImage(ctx context.Context, image string) error {
	cosignPath, err := exec.LookPath("cosign")
	if err != nil {
		// cosign not installed — skip verification silently
		return nil
	}

	cmd := exec.CommandContext(ctx, cosignPath, "verify", "--insecure-ignore-tlog", image)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if cosign failed because the image isn't signed at all
		if bytes.Contains(bytes.ToLower(output), []byte("no signatures found")) ||
			bytes.Contains(bytes.ToLower(output), []byte("no matching signatures")) {
			return nil
		}
		return fmt.Errorf("image verification failed for %s: %w\n%s", image, err, output)
	}
	return nil
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

// BuildImage builds a container image from a Dockerfile.
func (d *DockerRuntime) BuildImage(ctx context.Context, cfg runtime.BuildConfig, statusChan chan<- string) (string, error) {
	contextDir := cfg.ContextDir
	if contextDir == "" {
		return "", fmt.Errorf("build context directory is required")
	}

	contextDir = filepath.Clean(contextDir)
	if _, err := os.Stat(contextDir); err != nil {
		return "", fmt.Errorf("build context %s: %w", contextDir, err)
	}

	dockerfile := cfg.Dockerfile
	if dockerfile == "" {
		dockerfile = "Dockerfile"
	}

	dockerfilePath := filepath.Join(contextDir, dockerfile)
	if _, err := os.Stat(dockerfilePath); err != nil {
		return "", fmt.Errorf("dockerfile %s: %w", dockerfilePath, err)
	}

	tags := cfg.Tags
	if len(tags) == 0 {
		tags = []string{fmt.Sprintf("devbox-%s:latest", filepath.Base(contextDir))}
	}

	buildArgs := make(map[string]*string)
	for k, v := range cfg.BuildArgs {
		val := v
		buildArgs[k] = &val
	}

	statusChan <- fmt.Sprintf("Building image from %s...", contextDir)

	tarContext, err := createBuildTar(contextDir, dockerfile)
	if err != nil {
		return "", fmt.Errorf("create build context tar: %w", err)
	}
	defer tarContext.Close()

	resp, err := d.cli.ImageBuild(ctx, tarContext, build.ImageBuildOptions{
		Dockerfile: dockerfile,
		Tags:       tags,
		BuildArgs:  buildArgs,
		Target:     cfg.Target,
		Remove:     true,
		NoCache:    cfg.NoCache,
		PullParent: cfg.Pull,
	})
	if err != nil {
		return "", fmt.Errorf("start build: %w", err)
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var imageID string

	for {
		var msg struct {
			Stream      string `json:"stream"`
			Error       string `json:"error"`
			ErrorDetail struct {
				Message string `json:"message"`
			} `json:"errorDetail"`
			Aux struct {
				ID string `json:"ID"`
			} `json:"aux"`
		}

		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("read build output: %w", err)
		}

		if msg.Error != "" {
			return "", fmt.Errorf("build failed: %s", msg.Error)
		}

		if msg.Stream != "" {
			statusChan <- msg.Stream
		}

		if msg.Aux.ID != "" {
			imageID = msg.Aux.ID
		}
	}

	if imageID == "" {
		return "", fmt.Errorf("build completed but no image ID returned")
	}

	statusChan <- fmt.Sprintf("Build complete: %s", tags[0])
	return tags[0], nil
}

// createBuildTar creates a tar archive of the build context.
func createBuildTar(contextDir, dockerfile string) (io.ReadCloser, error) {
	pipeReader, pipeWriter := io.Pipe()
	tarWriter := tar.NewWriter(pipeWriter)

	go func() {
		defer pipeWriter.Close()

		err := filepath.Walk(contextDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(contextDir, path)
			if err != nil {
				return err
			}

			if relPath == "." {
				return nil
			}

			if shouldIgnore(relPath, dockerfile) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}

			header.Name = filepath.ToSlash(relPath)

			if info.IsDir() {
				header.Name += "/"
				header.Size = 0
			}

			if err := tarWriter.WriteHeader(header); err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(tarWriter, file)
			return err
		})

		if err != nil {
			pipeWriter.CloseWithError(err)
		}
	}()

	return pipeReader, nil
}

// shouldIgnore checks if a path should be excluded from the build context.
func shouldIgnore(path, dockerfile string) bool {
	ignorePatterns := []string{
		".git",
		".gitignore",
		".devbox",
		"node_modules",
		"__pycache__",
		".pytest_cache",
		".tox",
		".venv",
		"venv",
		"dist",
		"build",
		"*.exe",
		"*.dll",
		"*.so",
		"*.dylib",
	}

	for _, pattern := range ignorePatterns {
		matched, _ := filepath.Match(pattern, filepath.Base(path))
		if matched {
			return true
		}
	}

	return false
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

	// Security hardening: drop all capabilities by default, add back only those requested
	hostConfig.CapDrop = []string{"ALL"}
	if len(cfg.Capabilities) > 0 {
		hostConfig.CapAdd = cfg.Capabilities
	}

	// no-new-privileges blocks privilege escalation via suid binaries
	if cfg.NoNewPrivileges {
		hostConfig.SecurityOpt = append(hostConfig.SecurityOpt, "no-new-privileges:true")
	}

	// Seccomp profile: "" = Docker default, "unconfined" = no seccomp, path = custom
	if cfg.SeccompProfile != "" && cfg.SeccompProfile != "unconfined" {
		hostConfig.SecurityOpt = append(hostConfig.SecurityOpt, "seccomp="+cfg.SeccompProfile)
	} else if cfg.SeccompProfile == "unconfined" {
		hostConfig.SecurityOpt = append(hostConfig.SecurityOpt, "seccomp=unconfined")
	}

	// AppArmor profile
	if cfg.AppArmorProfile != "" {
		hostConfig.SecurityOpt = append(hostConfig.SecurityOpt, "apparmor="+cfg.AppArmorProfile)
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
		Labels:    info.Config.Labels,
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

		name := ""
		if len(c.Names) > 0 {
			name = c.Names[0]
		}
		result = append(result, runtime.ContainerInfo{
			ID:     c.ID,
			Name:   name,
			Image:  c.Image,
			Status: c.State,
			Ports:  ports,
			Labels: c.Labels,
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

// VolumePath returns the host path for a Docker volume.
// Docker volumes are stored in Docker's internal storage, so they are not
// directly accessible from the host filesystem.
func (d *DockerRuntime) VolumePath(ctx context.Context, name string) (string, error) {
	return "", fmt.Errorf("%w: Docker volumes are not directly accessible from host", runtime.ErrNotSupported)
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
