package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	pb "github.com/devboxos/devboxos/engine/proto"
	"github.com/devboxos/devboxos/shared/platform"
	"github.com/devboxos/devboxos/shared/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

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

// New creates a new engine client.
func New() (*Client, error) {
	addr := platform.EngineAddress()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to engine daemon at %s: %w\n\nIs the engine running? Run: devbox-engine --daemon", addr, err)
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
		if err != nil {
			return fmt.Errorf("stream: %w", err)
		}
		if resp.Done {
			if resp.Error != "" {
				return fmt.Errorf("%s", resp.Error)
			}
			return nil
		}
	}
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
