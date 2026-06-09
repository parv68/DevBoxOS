package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	pb "github.com/devboxos/devboxos/engine/proto"
	"github.com/devboxos/devboxos/shared/platform"
	"github.com/devboxos/devboxos/shared/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// EngineBinPath returns the path to the devbox-engine binary,
// assumed to be in the same directory as the running CLI binary.
func EngineBinPath() (string, error) {
	cliExe, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(cliExe)
	bin := filepath.Join(dir, "devbox-engine")
	if platform.IsWindows() {
		bin += ".exe"
	}
	return bin, nil
}

// startEngineDaemon launches the engine daemon as a background process.
func startEngineDaemon() error {
	binPath, err := EngineBinPath()
	if err != nil {
		return fmt.Errorf("locate engine binary: %w", err)
	}
	cmd := exec.Command(binPath, "--daemon")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start engine daemon: %w", err)
	}
	return nil
}

func configPath() string {
	return filepath.Join(platform.ConfigDir(), "config.json")
}

func loadConfig() (map[string]string, error) {
	cfg := map[string]string{
		"telemetry": "true",
		"engine":    "auto",
	}
	data, err := os.ReadFile(configPath())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	var fileCfg map[string]string
	if err := json.Unmarshal(data, &fileCfg); err != nil {
		return cfg, nil
	}
	for k, v := range fileCfg {
		cfg[k] = v
	}
	return cfg, nil
}

func saveConfig(cfg map[string]string) error {
	dir := filepath.Dir(configPath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0644)
}

// Client is the gRPC client for communicating with the engine daemon.
type Client struct {
	conn   *grpc.ClientConn
	client pb.EngineServiceClient
}

// New creates a new engine client, auto-starting the engine daemon if needed.
func New() (*Client, error) {
	addr := platform.EngineAddress()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err == nil {
		return &Client{
			conn:   conn,
			client: pb.NewEngineServiceClient(conn),
		}, nil
	}

	if err := startEngineDaemon(); err != nil {
		return nil, fmt.Errorf("connect to engine daemon at %s: %w\n\nFailed to auto-start engine: %v", addr, err, err)
	}

	// Retry connection after starting the daemon
	time.Sleep(1 * time.Second)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()

	conn, err = grpc.DialContext(ctx2, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to engine daemon at %s after auto-start: %w", addr, err)
	}

	return &Client{
		conn:   conn,
		client: pb.NewEngineServiceClient(conn),
	}, nil
}

// Close closes the gRPC connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// Ping checks if the engine is responsive.
func (c *Client) Ping() (*pb.PingResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.client.Ping(ctx, &pb.PingRequest{})
}

// Init initializes a new project (local-only, no engine needed).
func (c *Client) Init(dir, name string) error {
	return nil
}

// Start starts all services with streaming status updates.
func (c *Client) Start(dir string, statusCallback func(status, msg string)) error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	stream, err := c.client.Start(ctx, &pb.StartRequest{
		ProjectPath: dir,
	})
	if err != nil {
		return fmt.Errorf("start: %w", err)
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("stream: %w", err)
		}
		if statusCallback != nil {
			statusCallback(resp.Status, resp.Message)
		}
		if resp.Done {
			if resp.Error != "" {
				return fmt.Errorf("%s", resp.Error)
			}
			if resp.Status == "error" {
				return fmt.Errorf("%s", resp.Message)
			}
			return nil
		}
	}
}

// Stop stops services.
func (c *Client) Stop(dir, service string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := c.client.Stop(ctx, &pb.StopRequest{
		ProjectPath: dir,
		Service:     service,
	})
	if err != nil {
		return fmt.Errorf("stop: %w", err)
	}
	if resp.Error != "" {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

// Status gets the environment status.
func (c *Client) Status(dir string) (*types.EnvironmentStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.Status(ctx, &pb.StatusRequest{
		ProjectPath: dir,
	})
	if err != nil {
		return nil, fmt.Errorf("status: %w", err)
	}

	status := &types.EnvironmentStatus{
		Path:   dir,
		Status: resp.Status,
	}

	for _, svc := range resp.Services {
		status.Services = append(status.Services, types.ServiceStatus{
			Name:         svc.Name,
			Status:       svc.Status,
			Health:       svc.Health,
			Port:         int(svc.Port),
			ContainerID:  svc.ContainerId,
			RestartCount: int(svc.RestartCount),
		})
	}

	return status, nil
}

// Logs streams logs from a service.
func (c *Client) Logs(dir, service string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	stream, err := c.client.Logs(ctx, &pb.LogsRequest{
		ProjectPath: dir,
		Service:     service,
		Follow:      true,
	})
	if err != nil {
		return fmt.Errorf("logs: %w", err)
	}

	for {
		entry, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("stream: %w", err)
		}
		fmt.Printf("[%s] %s\n", entry.Service, entry.Message)
	}
}

// Doctor runs diagnostics.
func (c *Client) Doctor(dir string) (*pb.DoctorResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return c.client.Doctor(ctx, &pb.DoctorRequest{
		ProjectPath: dir,
	})
}

// Reset tears down and rebuilds the environment.
func (c *Client) Reset(dir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	stream, err := c.client.Reset(ctx, &pb.ResetRequest{
		ProjectPath: dir,
	})
	if err != nil {
		return fmt.Errorf("reset: %w", err)
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("stream: %w", err)
		}
		if resp.Done {
			if resp.Error != "" {
				return fmt.Errorf("%s", resp.Error)
			}
			if resp.Status == "error" {
				return fmt.Errorf("%s", resp.Message)
			}
			return nil
		}
	}
}

// SecretSet stores a secret via the engine.
func (c *Client) SecretSet(projectPath, name, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.SecretSet(ctx, &pb.SecretSetRequest{
		ProjectPath: projectPath,
		Name:        name,
		Value:       value,
	})
	if err != nil {
		return fmt.Errorf("secret set: %w", err)
	}
	if resp.Error != "" {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

// SecretGet retrieves a secret value via the engine.
func (c *Client) SecretGet(projectPath, name string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.SecretGet(ctx, &pb.SecretGetRequest{
		ProjectPath: projectPath,
		Name:        name,
	})
	if err != nil {
		return "", fmt.Errorf("secret get: %w", err)
	}
	if resp.Error != "" {
		return "", fmt.Errorf("%s", resp.Error)
	}
	return resp.Value, nil
}

// SecretList lists all secrets via the engine.
func (c *Client) SecretList(projectPath string) ([]*pb.SecretEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.SecretList(ctx, &pb.SecretListRequest{
		ProjectPath: projectPath,
	})
	if err != nil {
		return nil, fmt.Errorf("secret list: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("%s", resp.Error)
	}
	return resp.Secrets, nil
}

// SecretDelete removes a secret via the engine.
func (c *Client) SecretDelete(projectPath, name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.SecretDelete(ctx, &pb.SecretDeleteRequest{
		ProjectPath: projectPath,
		Name:        name,
	})
	if err != nil {
		return fmt.Errorf("secret delete: %w", err)
	}
	if resp.Error != "" {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

// SecretRotate rotates a secret via the engine.
func (c *Client) SecretRotate(projectPath, name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.SecretRotate(ctx, &pb.SecretRotateRequest{
		ProjectPath: projectPath,
		Name:        name,
	})
	if err != nil {
		return fmt.Errorf("secret rotate: %w", err)
	}
	if resp.Error != "" {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

// SnapshotSave saves a snapshot with streaming status updates.
func (c *Client) SnapshotSave(projectPath, name string, includeLogs bool, statusCb func(string)) error {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	stream, err := c.client.SnapshotSave(ctx, &pb.SnapshotSaveRequest{
		ProjectPath: projectPath,
		Name:        name,
		IncludeLogs: includeLogs,
	})
	if err != nil {
		return fmt.Errorf("snapshot save: %w", err)
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("snapshot save stream: %w", err)
		}
		if statusCb != nil {
			statusCb(resp.Message)
		}
		if resp.Done {
			if resp.Error != "" {
				return fmt.Errorf("%s", resp.Error)
			}
			if resp.Status == "error" {
				return fmt.Errorf("%s", resp.Message)
			}
			return nil
		}
	}
}

// SnapshotLoad loads a snapshot with streaming status updates.
func (c *Client) SnapshotLoad(projectPath, snapshotId string, force bool, statusCb func(string)) error {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	stream, err := c.client.SnapshotLoad(ctx, &pb.SnapshotLoadRequest{
		ProjectPath: projectPath,
		SnapshotId:  snapshotId,
		Force:       force,
	})
	if err != nil {
		return fmt.Errorf("snapshot load: %w", err)
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("snapshot load stream: %w", err)
		}
		if statusCb != nil {
			statusCb(resp.Message)
		}
		if resp.Done {
			if resp.Error != "" {
				return fmt.Errorf("%s", resp.Error)
			}
			if resp.Status == "error" {
				return fmt.Errorf("%s", resp.Message)
			}
			return nil
		}
	}
}

// SnapshotList lists all snapshots for a project.
func (c *Client) SnapshotList(projectPath string) ([]*pb.Snapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.SnapshotList(ctx, &pb.SnapshotListRequest{
		ProjectPath: projectPath,
	})
	if err != nil {
		return nil, fmt.Errorf("snapshot list: %w", err)
	}

	return resp.Snapshots, nil
}

// SnapshotDelete deletes a snapshot.
func (c *Client) SnapshotDelete(projectPath, snapshotId string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.SnapshotDelete(ctx, &pb.SnapshotDeleteRequest{
		ProjectPath: projectPath,
		SnapshotId:  snapshotId,
	})
	if err != nil {
		return fmt.Errorf("snapshot delete: %w", err)
	}
	if resp.Error != "" {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

// SnapshotExport saves a snapshot to a local file with streaming status updates.
func (c *Client) SnapshotExport(projectPath, exportPath, snapshotID string, statusCb func(string)) error {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	stream, err := c.client.SnapshotExport(ctx, &pb.SnapshotExportRequest{
		ProjectPath: projectPath,
		ExportPath:  exportPath,
		SnapshotId:  snapshotID,
	})
	if err != nil {
		return fmt.Errorf("snapshot export: %w", err)
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("snapshot export stream: %w", err)
		}
		if statusCb != nil {
			statusCb(resp.Message)
		}
		if resp.Done {
			if resp.Error != "" {
				return fmt.Errorf("%s", resp.Error)
			}
			if resp.Status == "error" {
				return fmt.Errorf("%s", resp.Message)
			}
			return nil
		}
	}
}

// SnapshotImport loads a snapshot from a local file with streaming status updates.
func (c *Client) SnapshotImport(projectPath, importPath string, force bool, statusCb func(string)) error {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	stream, err := c.client.SnapshotImport(ctx, &pb.SnapshotImportRequest{
		ProjectPath: projectPath,
		ImportPath:  importPath,
		Force:       force,
	})
	if err != nil {
		return fmt.Errorf("snapshot import: %w", err)
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("snapshot import stream: %w", err)
		}
		if statusCb != nil {
			statusCb(resp.Message)
		}
		if resp.Done {
			if resp.Error != "" {
				return fmt.Errorf("%s", resp.Error)
			}
			if resp.Status == "error" {
				return fmt.Errorf("%s", resp.Message)
			}
			return nil
		}
	}
}

// Build builds service images via the engine with streaming status updates.
func (c *Client) Build(projectPath, service string, noCache, pull bool, statusCallback func(status, msg string)) error {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	stream, err := c.client.Build(ctx, &pb.BuildRequest{
		ProjectPath: projectPath,
		Service:     service,
		NoCache:     noCache,
		Pull:        pull,
	})
	if err != nil {
		return fmt.Errorf("build: %w", err)
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("build stream: %w", err)
		}
		if statusCallback != nil {
			statusCallback(resp.Status, resp.Message)
		}
		if resp.Done {
			if resp.Error != "" {
				return fmt.Errorf("%s", resp.Error)
			}
			if resp.Status == "error" {
				return fmt.Errorf("%s", resp.Message)
			}
			return nil
		}
	}
}

// Shutdown gracefully stops the engine daemon.
func (c *Client) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.client.Shutdown(ctx, &pb.ShutdownRequest{})
	return err
}

// Exec runs a command inside a service container via the engine.
func (c *Client) Exec(projectPath, service, command string, args []string) (string, string, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := c.client.Exec(ctx, &pb.ExecRequest{
		ProjectPath: projectPath,
		Service:     service,
		Command:     command,
		Args:        args,
	})
	if err != nil {
		return "", "", -1, fmt.Errorf("exec: %w", err)
	}

	return resp.Stdout, resp.Stderr, int(resp.ExitCode), nil
}

// GetConfig returns the current CLI configuration.
func (c *Client) GetConfig() (map[string]string, error) {
	return loadConfig()
}

// GetConfigKey returns a specific configuration value.
func (c *Client) GetConfigKey(key string) (string, error) {
	cfg, err := loadConfig()
	if err != nil {
		return "", err
	}
	if val, ok := cfg[key]; ok {
		return val, nil
	}
	return "", fmt.Errorf("unknown config key: %s", key)
}

// SetConfigKey sets a specific configuration value.
func (c *Client) SetConfigKey(key, value string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	cfg[key] = value
	return saveConfig(cfg)
}
