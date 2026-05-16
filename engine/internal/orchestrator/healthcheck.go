package orchestrator

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"time"
)

// HealthChecker performs health checks on services.
type HealthChecker struct{}

// NewHealthChecker creates a new health checker.
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{}
}

// Check performs a health check based on the configuration.
func (h *HealthChecker) Check(ctx context.Context, checkType, target string) error {
	switch checkType {
	case "http":
		return h.checkHTTP(ctx, target)
	case "tcp":
		return h.checkTCP(ctx, target)
	case "cmd":
		return h.checkCmd(ctx, target)
	default:
		return h.checkTCP(ctx, target) // Default to TCP
	}
}

// checkHTTP performs an HTTP health check.
func (h *HealthChecker) checkHTTP(ctx context.Context, url string) error {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create HTTP request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	return nil
}

// checkTCP performs a TCP port check.
func (h *HealthChecker) checkTCP(ctx context.Context, address string) error {
	dialer := net.Dialer{
		Timeout: 5 * time.Second,
	}

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("TCP connection failed: %w", err)
	}
	conn.Close()

	return nil
}

// checkCmd performs a command-based health check.
func (h *HealthChecker) checkCmd(ctx context.Context, command string) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}
	return nil
}
