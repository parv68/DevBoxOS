//go:build integration

package snapshot

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/devboxos/devboxos/shared/runtime"
	"github.com/devboxos/devboxos/shared/runtime/docker"
	"github.com/devboxos/devboxos/shared/types"
)

func TestSnapshot_VolumeExportImport(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker integration test in short mode")
	}
	if os.Getenv("DOCKER_TEST_PULL") == "" {
		t.Skip("Set DOCKER_TEST_PULL=1 to run integration tests")
	}

	rt := docker.NewDockerRuntime()
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	if err := rt.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer rt.Close()

	tmpDir := t.TempDir()
	m := NewManager(rt, tmpDir)

	volName := "devbox-test-vol-integration"
	rt.RemoveVolume(ctx, volName)
	if err := rt.CreateVolume(ctx, volName); err != nil {
		t.Fatalf("CreateVolume() failed: %v", err)
	}
	defer rt.RemoveVolume(ctx, volName)

	// Write some data to the volume via a temporary container
	writeCfg := runtime.ContainerConfig{
		Name:    "devbox-test-vol-writer",
		Image:   "alpine:latest",
		Command: []string{"sh", "-c", "echo 'test data' > /data/test.txt"},
		Volumes: map[string]string{volName: "/data"},
	}
	cid, err := rt.CreateContainer(ctx, writeCfg)
	if err != nil {
		t.Fatalf("CreateContainer() for volume write failed: %v", err)
	}
	if err := rt.StartContainer(ctx, cid); err != nil {
		t.Fatalf("StartContainer() failed: %v", err)
	}
	time.Sleep(2 * time.Second)
	rt.RemoveContainer(ctx, cid, true)

	// Create a valid config and save a snapshot
	cfg := &types.Config{
		Name:    "test-project",
		Version: "1.0",
		Services: map[string]types.Service{
			"web": {
				Image:   "nginx:alpine",
				Volumes: []string{volName + ":/data"},
			},
		},
	}

	statusChan := make(chan string, 64)
	go func() {
		for range statusChan {
		}
	}()
	defer close(statusChan)

	manifest, err := m.Save(ctx, cfg, "test-snapshot", false, statusChan)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	if manifest == nil {
		t.Fatal("Save() returned nil manifest")
	}
	if manifest.Name != "test-snapshot" {
		t.Errorf("unexpected manifest name: %s", manifest.Name)
	}
}

func TestSnapshot_ExportImport(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a store and snapshot data directly (no Docker needed for tarball ops)
	s := NewStore(tmpDir)
	s.SaveManifest(&Manifest{ID: "test-id", Name: "test", CreatedAt: time.Now()})

	snapDir := s.SnapshotDir("test-id")
	os.MkdirAll(snapDir, 0755)
	os.WriteFile(filepath.Join(snapDir, "data.txt"), []byte("snapshot content"), 0644)

	exportPath := filepath.Join(tmpDir, "export.tar.gz")

	// We need a Manager with a real Store but nil runtime
	m := &Manager{store: s, rt: nil, secrets: NewSecretsHandler("", "")}

	statusChan := make(chan string, 64)
	go func() {
		for range statusChan {
		}
	}()
	defer close(statusChan)

	if err := m.Export("test-id", exportPath, statusChan); err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	importDir := t.TempDir()
	importStore := NewStore(importDir)
	m2 := &Manager{store: importStore, rt: nil, secrets: NewSecretsHandler("", "")}

	if err := m2.Import(exportPath, statusChan); err != nil {
		t.Fatalf("Import() failed: %v", err)
	}

	loaded, err := importStore.LoadManifest("test-id")
	if err != nil {
		t.Fatalf("LoadManifest() after import failed: %v", err)
	}
	if loaded.Name != "test" {
		t.Errorf("expected name 'test', got '%s'", loaded.Name)
	}

	data, _ := os.ReadFile(filepath.Join(importStore.SnapshotDir("test-id"), "data.txt"))
	if string(data) != "snapshot content" {
		t.Errorf("unexpected data after import: %s", string(data))
	}
}
