package runtime

import (
	"context"
	"errors"
	"io"
)

// ErrNotSupported is returned when the runtime does not support an operation.
var ErrNotSupported = errors.New("operation not supported by this runtime")

// ContainerConfig holds the configuration for creating a container.
type ContainerConfig struct {
	Name       string
	Image      string
	Command    []string
	WorkingDir string
	Env        map[string]string
	Ports      map[string]string // "host:container"
	Volumes    map[string]string // "host:container"
	Network    string
	Labels     map[string]string
	Memory     string // e.g. "512m"
	CPU        string // e.g. "0.5"
	ReadOnly   bool
}

// ContainerInfo holds runtime information about a container.
type ContainerInfo struct {
	ID        string
	Name      string
	Image     string
	Status    string // created, running, exited, dead
	Ports     []PortMapping
	Networks  []string
	StartedAt string
	Health    string // healthy, unhealthy, starting, none
	Labels    map[string]string
	PID       int32  // host OS process ID (0 if not applicable)
}

// PortMapping represents a port binding.
type PortMapping struct {
	HostIP   string
	HostPort string
	ContainerPort string
	Protocol string
}

// BuildConfig holds the configuration for building an image.
type BuildConfig struct {
	ContextDir string            // Path to build context
	Dockerfile string            // Dockerfile name (default: "Dockerfile")
	BuildArgs  map[string]string // Build-time variables
	Target     string            // Target stage for multi-stage builds
	Tags       []string          // Image tags
	NoCache    bool              // Do not use cache
	Pull       bool              // Always pull base image
}

// LogOptions controls log streaming behavior.
type LogOptions struct {
	Follow bool
	Tail   int
	Since  string
}

// Runtime defines the interface for container runtimes.
type Runtime interface {
	// Connect establishes a connection to the runtime daemon.
	Connect(ctx context.Context) error

	// Close closes the connection.
	Close() error

	// Check verifies the runtime is accessible.
	Check(ctx context.Context) error

	// PullImage pulls a container image.
	PullImage(ctx context.Context, image string) error

	// BuildImage builds a container image from a Dockerfile.
	BuildImage(ctx context.Context, cfg BuildConfig, statusChan chan<- string) (string, error)

	// CreateContainer creates a new container.
	CreateContainer(ctx context.Context, cfg ContainerConfig) (string, error)

	// StartContainer starts a container by ID.
	StartContainer(ctx context.Context, id string) error

	// StopContainer stops a container by ID.
	StopContainer(ctx context.Context, id string, timeoutSeconds int) error

	// RemoveContainer removes a container by ID.
	RemoveContainer(ctx context.Context, id string, force bool) error

	// GetContainerInfo returns information about a container.
	GetContainerInfo(ctx context.Context, id string) (ContainerInfo, error)

	// ListContainers returns all containers with the given labels.
	ListContainers(ctx context.Context, labels map[string]string) ([]ContainerInfo, error)

	// StreamLogs streams logs from a container.
	StreamLogs(ctx context.Context, id string, opts LogOptions) (io.ReadCloser, error)

	// CreateNetwork creates a new network.
	CreateNetwork(ctx context.Context, name string) error

	// RemoveNetwork removes a network.
	RemoveNetwork(ctx context.Context, name string) error

	// NetworkExists checks if a network exists.
	NetworkExists(ctx context.Context, name string) (bool, error)

	// CreateVolume creates a named volume.
	CreateVolume(ctx context.Context, name string) error

	// RemoveVolume removes a named volume.
	RemoveVolume(ctx context.Context, name string) error

	// VolumeExists checks if a volume exists.
	VolumeExists(ctx context.Context, name string) (bool, error)

	// VolumePath returns the host filesystem path for a volume, if directly accessible.
	// Returns ErrNotSupported if the volume cannot be accessed from the host filesystem
	// (e.g. Docker named volumes in internal storage).
	VolumePath(ctx context.Context, name string) (string, error)
}
