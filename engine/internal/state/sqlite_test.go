package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome)
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserProfile)
	}()

	m, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}
	defer m.Close()

	if m == nil {
		t.Fatal("NewManager() returned nil")
	}

	db := m.DB()
	if db == nil {
		t.Fatal("DB() returned nil")
	}

	// Verify state.db was created
	dbPath := filepath.Join(tmpHome, ".devbox", "state.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("state.db not created at %s", dbPath)
	}
}

func TestNewManager_CreatesTables(t *testing.T) {
	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome)
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserProfile)
	}()

	m, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}
	defer m.Close()

	tables := []string{"environments", "services", "snapshots", "locks", "telemetry"}
	for _, table := range tables {
		var count int
		err := m.DB().QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count)
		if err != nil {
			t.Errorf("table %s query failed: %v", table, err)
		}
	}
}

func TestManager_Close(t *testing.T) {
	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome)
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserProfile)
	}()

	m, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	if err := m.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	// DB() should still return non-nil, but queries should fail
	db := m.DB()
	if db == nil {
		t.Fatal("DB() returned nil after Close()")
	}
}

func TestManager_SingleWriter(t *testing.T) {
	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome)
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserProfile)
	}()

	m, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}
	defer m.Close()

	_, err = m.DB().Exec("INSERT INTO environments (id, name, path, version, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, datetime('now'), datetime('now'))",
		"test-id", "test-env", "/tmp/test", "1.0", "stopped")
	if err != nil {
		t.Errorf("insert failed: %v", err)
	}

	var name string
	err = m.DB().QueryRow("SELECT name FROM environments WHERE id = ?", "test-id").Scan(&name)
	if err != nil {
		t.Errorf("query failed: %v", err)
	}
	if name != "test-env" {
		t.Errorf("expected test-env, got %s", name)
	}
}
