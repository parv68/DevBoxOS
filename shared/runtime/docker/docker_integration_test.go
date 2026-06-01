//go:build integration

package docker

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/devboxos/devboxos/shared/runtime"
)

func TestDockerRuntime_Connect(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker integration test in short mode")
	}

	rt := NewDockerRuntime()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := rt.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer rt.Close()
}

func TestDockerRuntime_Check(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker integration test in short mode")
	}

	rt := NewDockerRuntime()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := rt.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer rt.Close()

	if err := rt.Check(ctx); err != nil {
		t.Fatalf("Check() failed: %v", err)
	}
}

func TestDockerRuntime_PullAndRunContainer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker integration test in short mode")
	}
	if os.Getenv("DOCKER_TEST_PULL") == "" {
		t.Skip("Set DOCKER_TEST_PULL=1 to run pull tests")
	}

	rt := NewDockerRuntime()
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	if err := rt.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer rt.Close()

	if err := rt.PullImage(ctx, "alpine:latest"); err != nil {
		t.Fatalf("PullImage() failed: %v", err)
	}

	containerID, err := rt.CreateContainer(ctx, runtime.ContainerConfig{
		Name:    "devbox-test-integration",
		Image:   "alpine:latest",
		Command: []string{"sh", "-c", "echo hello && sleep 1"},
		Labels: map[string]string{
			"devboxos.test": "integration",
		},
	})
	if err != nil {
		t.Fatalf("CreateContainer() failed: %v", err)
	}

	if err := rt.StartContainer(ctx, containerID); err != nil {
		t.Fatalf("StartContainer() failed: %v", err)
	}

	time.Sleep(3 * time.Second)

	info, err := rt.GetContainerInfo(ctx, containerID)
	if err != nil {
		t.Fatalf("GetContainerInfo() failed: %v", err)
	}
	if info.Status != "exited" && info.Status != "Exited" {
		t.Logf("Container status: %s", info.Status)
	}

	containers, err := rt.ListContainers(ctx, map[string]string{
		"devboxos.test": "integration",
	})
	if err != nil {
		t.Fatalf("ListContainers() failed: %v", err)
	}
	if len(containers) == 0 {
		t.Error("expected at least 1 container matching label")
	}

	if err := rt.RemoveContainer(ctx, containerID, true); err != nil {
		t.Fatalf("RemoveContainer() failed: %v", err)
	}
}

func TestDockerRuntime_NetworkLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker integration test in short mode")
	}

	rt := NewDockerRuntime()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := rt.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer rt.Close()

	networkName := "devbox-test-network-integration"

	exists, err := rt.NetworkExists(ctx, networkName)
	if err != nil {
		t.Fatalf("NetworkExists() failed: %v", err)
	}
	if exists {
		t.Logf("Network %s already exists, removing first", networkName)
		rt.RemoveNetwork(ctx, networkName)
	}

	if err := rt.CreateNetwork(ctx, networkName); err != nil {
		t.Fatalf("CreateNetwork() failed: %v", err)
	}

	exists, err = rt.NetworkExists(ctx, networkName)
	if err != nil {
		t.Fatalf("NetworkExists() failed: %v", err)
	}
	if !exists {
		t.Error("expected network to exist after creation")
	}

	if err := rt.RemoveNetwork(ctx, networkName); err != nil {
		t.Fatalf("RemoveNetwork() failed: %v", err)
	}

	exists, err = rt.NetworkExists(ctx, networkName)
	if err != nil {
		t.Fatalf("NetworkExists() failed: %v", err)
	}
	if exists {
		t.Error("expected network to not exist after removal")
	}
}

func TestDockerRuntime_VolumeLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker integration test in short mode")
	}

	rt := NewDockerRuntime()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := rt.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer rt.Close()

	volName := "devbox-test-volume-integration"

	exists, err := rt.VolumeExists(ctx, volName)
	if err != nil {
		t.Fatalf("VolumeExists() failed: %v", err)
	}
	if exists {
		t.Logf("Volume %s already exists, removing first", volName)
		rt.RemoveVolume(ctx, volName)
	}

	if err := rt.CreateVolume(ctx, volName); err != nil {
		t.Fatalf("CreateVolume() failed: %v", err)
	}

	exists, err = rt.VolumeExists(ctx, volName)
	if err != nil {
		t.Fatalf("VolumeExists() failed: %v", err)
	}
	if !exists {
		t.Error("expected volume to exist after creation")
	}

	if err := rt.RemoveVolume(ctx, volName); err != nil {
		t.Fatalf("RemoveVolume() failed: %v", err)
	}

	exists, err = rt.VolumeExists(ctx, volName)
	if err != nil {
		t.Fatalf("VolumeExists() failed: %v", err)
	}
	if exists {
		t.Error("expected volume to not exist after removal")
	}
}

func TestDockerRuntime_StreamLogs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker integration test in short mode")
	}
	if os.Getenv("DOCKER_TEST_PULL") == "" {
		t.Skip("Set DOCKER_TEST_PULL=1 to run log tests")
	}

	rt := NewDockerRuntime()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := rt.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer rt.Close()

	containerID, err := rt.CreateContainer(ctx, runtime.ContainerConfig{
		Name:    "devbox-test-logs",
		Image:   "alpine:latest",
		Command: []string{"sh", "-c", "echo line1 && echo line2 && echo line3"},
		Labels:  map[string]string{"devboxos.test": "logs"},
	})
	if err != nil {
		t.Fatalf("CreateContainer() failed: %v", err)
	}

	if err := rt.StartContainer(ctx, containerID); err != nil {
		t.Fatalf("StartContainer() failed: %v", err)
	}

	time.Sleep(3 * time.Second)

	reader, err := rt.StreamLogs(ctx, containerID, runtime.LogOptions{
		Follow: false,
		Tail:   10,
	})
	if err != nil {
		t.Fatalf("StreamLogs() failed: %v", err)
	}
	defer reader.Close()

	data := make([]byte, 4096)
	n, _ := reader.Read(data)
	if n == 0 {
		t.Log("no log data returned (container may not have started log stream)")
	}

	rt.RemoveContainer(ctx, containerID, true)
}
