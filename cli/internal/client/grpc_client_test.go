package client

import (
	"testing"
)

func setupTest(t *testing.T) (*Client, *mockServer, func()) {
	t.Helper()
	ms, pbClient, cleanup, err := startMockServer()
	if err != nil {
		t.Fatalf("startMockServer() failed: %v", err)
	}

	// Create real Client wrapping the pb client
	c := &Client{
		client: pbClient,
	}

	return c, ms, cleanup
}

func TestClient_Ping(t *testing.T) {
	c, _, cleanup := setupTest(t)
	defer cleanup()

	resp, err := c.Ping()
	if err != nil {
		t.Fatalf("Ping() failed: %v", err)
	}
	if resp.Version != "0.1.0-test" {
		t.Errorf("expected version '0.1.0-test', got '%s'", resp.Version)
	}
	if resp.Uptime != 42 {
		t.Errorf("expected uptime 42, got %d", resp.Uptime)
	}
}

func TestClient_Start(t *testing.T) {
	c, _, cleanup := setupTest(t)
	defer cleanup()

	var messages []string
	err := c.Start("/tmp/test", func(status, msg string) {
		messages = append(messages, msg)
	})
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	if len(messages) == 0 {
		t.Error("expected status messages, got none")
	}
}

func TestClient_Stop(t *testing.T) {
	c, _, cleanup := setupTest(t)
	defer cleanup()

	err := c.Stop("/tmp/test", "")
	if err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}
}

func TestClient_Status(t *testing.T) {
	c, _, cleanup := setupTest(t)
	defer cleanup()

	status, err := c.Status("/tmp/test")
	if err != nil {
		t.Fatalf("Status() failed: %v", err)
	}
	if status.Status != "running" {
		t.Errorf("expected status 'running', got '%s'", status.Status)
	}
	if len(status.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(status.Services))
	}
}

func TestClient_Doctor(t *testing.T) {
	c, _, cleanup := setupTest(t)
	defer cleanup()

	resp, err := c.Doctor("/tmp/test")
	if err != nil {
		t.Fatalf("Doctor() failed: %v", err)
	}
	if len(resp.Issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(resp.Issues))
	}
}

func TestClient_Reset(t *testing.T) {
	c, _, cleanup := setupTest(t)
	defer cleanup()

	err := c.Reset("/tmp/test")
	if err != nil {
		t.Fatalf("Reset() failed: %v", err)
	}
}

func TestClient_SecretCRUD(t *testing.T) {
	c, ms, cleanup := setupTest(t)
	defer cleanup()

	err := c.SecretSet("/tmp/test", "DB_PASS", "s3cret!")
	if err != nil {
		t.Fatalf("SecretSet() failed: %v", err)
	}

	value, err := c.SecretGet("/tmp/test", "DB_PASS")
	if err != nil {
		t.Fatalf("SecretGet() failed: %v", err)
	}
	if value != "s3cret!" {
		t.Errorf("expected 's3cret!', got '%s'", value)
	}

	entries, err := c.SecretList("/tmp/test")
	if err != nil {
		t.Fatalf("SecretList() failed: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}

	err = c.SecretDelete("/tmp/test", "DB_PASS")
	if err != nil {
		t.Fatalf("SecretDelete() failed: %v", err)
	}

	// Verify it's deleted on the server
	ms.mu.Lock()
	_, exists := ms.secrets["DB_PASS"]
	ms.mu.Unlock()
	if exists {
		t.Error("expected secret to be deleted")
	}
}

func TestClient_SecretRotate_ManualFails(t *testing.T) {
	c, _, cleanup := setupTest(t)
	defer cleanup()

	c.SecretSet("/tmp/test", "MY_KEY", "old-value")

	err := c.SecretRotate("/tmp/test", "MY_KEY")
	if err == nil {
		t.Error("expected error rotating manual secret, got nil")
	}
}

func TestClient_Config(t *testing.T) {
	c, _, cleanup := setupTest(t)
	defer cleanup()

	err := c.SetConfigKey("test_key", "test_value")
	if err != nil {
		t.Fatalf("SetConfigKey() failed: %v", err)
	}

	val, err := c.GetConfigKey("test_key")
	if err != nil {
		t.Fatalf("GetConfigKey() failed: %v", err)
	}
	if val != "test_value" {
		t.Errorf("expected 'test_value', got '%s'", val)
	}

	cfg, err := c.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig() failed: %v", err)
	}
	if cfg["test_key"] != "test_value" {
		t.Errorf("expected test_key=test_value in config, got %v", cfg)
	}
}

func TestClient_Logs(t *testing.T) {
	c, _, cleanup := setupTest(t)
	defer cleanup()

	err := c.Logs("/tmp/test", "web")
	if err != nil {
		t.Fatalf("Logs() failed: %v", err)
	}
}

func TestClient_GetConfigKey_Unknown(t *testing.T) {
	c, _, cleanup := setupTest(t)
	defer cleanup()

	_, err := c.GetConfigKey("nonexistent-key")
	if err == nil {
		t.Error("expected error for unknown key")
	}
}

func TestClient_SnapshotSave(t *testing.T) {
	c, _, cleanup := setupTest(t)
	defer cleanup()

	var messages []string
	err := c.SnapshotSave("/tmp/test", "nightly-backup", false, func(msg string) {
		messages = append(messages, msg)
	})
	if err != nil {
		t.Fatalf("SnapshotSave() failed: %v", err)
	}
	if len(messages) == 0 {
		t.Error("expected status messages, got none")
	}
}

func TestClient_SnapshotLoad(t *testing.T) {
	c, _, cleanup := setupTest(t)
	defer cleanup()

	// First save one
	c.SnapshotSave("/tmp/test", "test-snap", false, nil)

	// Now list to find the ID
	snaps, err := c.SnapshotList("/tmp/test")
	if err != nil {
		t.Fatalf("SnapshotList() failed: %v", err)
	}
	if len(snaps) == 0 {
		t.Fatal("expected at least 1 snapshot")
	}

	var messages []string
	err = c.SnapshotLoad("/tmp/test", snaps[0].Id, false, func(msg string) {
		messages = append(messages, msg)
	})
	if err != nil {
		t.Fatalf("SnapshotLoad() failed: %v", err)
	}
	if len(messages) == 0 {
		t.Error("expected status messages, got none")
	}
}

func TestClient_SnapshotList(t *testing.T) {
	c, _, cleanup := setupTest(t)
	defer cleanup()

	// No snapshots yet
	snaps, err := c.SnapshotList("/tmp/test")
	if err != nil {
		t.Fatalf("SnapshotList() failed: %v", err)
	}
	if len(snaps) != 0 {
		t.Errorf("expected 0 snapshots, got %d", len(snaps))
	}

	// Save one
	c.SnapshotSave("/tmp/test", "test-snap", false, nil)

	snaps, err = c.SnapshotList("/tmp/test")
	if err != nil {
		t.Fatalf("SnapshotList() failed: %v", err)
	}
	if len(snaps) != 1 {
		t.Errorf("expected 1 snapshot, got %d", len(snaps))
	}
	if snaps[0].Name != "test-snap" {
		t.Errorf("expected name 'test-snap', got '%s'", snaps[0].Name)
	}
}

func TestClient_SnapshotDelete(t *testing.T) {
	c, ms, cleanup := setupTest(t)
	defer cleanup()

	// Save one
	c.SnapshotSave("/tmp/test", "to-delete", false, nil)

	snaps, _ := c.SnapshotList("/tmp/test")
	if len(snaps) != 1 {
		t.Fatal("expected 1 snapshot before delete")
	}

	err := c.SnapshotDelete("/tmp/test", snaps[0].Id)
	if err != nil {
		t.Fatalf("SnapshotDelete() failed: %v", err)
	}

	// Verify on server
	ms.mu.Lock()
	_, exists := ms.snapshots[snaps[0].Id]
	ms.mu.Unlock()
	if exists {
		t.Error("expected snapshot to be deleted")
	}
}
