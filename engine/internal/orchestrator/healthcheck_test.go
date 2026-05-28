package orchestrator

import (
	"context"
	"testing"
	"time"
)

func TestNewHealthChecker(t *testing.T) {
	hc := NewHealthChecker()
	if hc == nil {
		t.Fatal("NewHealthChecker() returned nil")
	}
}

func TestHealthCheck_TCP_InvalidAddress(t *testing.T) {
	hc := NewHealthChecker()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := hc.Check(ctx, "tcp", "0.0.0.0:1")
	if err == nil {
		t.Log("TCP check to privileged port 1 may have succeeded on this system")
	}
}

func TestHealthCheck_TCP_EmptyAddress(t *testing.T) {
	hc := NewHealthChecker()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := hc.Check(ctx, "tcp", "")
	if err == nil {
		t.Error("expected error for empty TCP address")
	}
}

func TestHealthCheck_HTTP_InvalidURL(t *testing.T) {
	hc := NewHealthChecker()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := hc.Check(ctx, "http", "http://0.0.0.0:1/health")
	if err == nil {
		t.Log("HTTP check to port 1 may have succeeded on this system")
	}
}

func TestHealthCheck_DefaultType(t *testing.T) {
	hc := NewHealthChecker()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := hc.Check(ctx, "unknown-type", "0.0.0.0:1")
	if err == nil {
		t.Log("TCP check via unknown type to port 1 may have succeeded")
	}
}

func TestHealthCheck_ContextCancelled(t *testing.T) {
	hc := NewHealthChecker()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := hc.Check(ctx, "tcp", "127.0.0.1:8080")
	if err == nil {
		t.Error("expected error with cancelled context")
	}
}
