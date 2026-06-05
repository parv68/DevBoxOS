package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/devboxos/devboxos/shared/runtime"
)

type containerEntry struct {
	info   runtime.ContainerInfo
	status string
	labels map[string]string
}

// MockRuntime implements runtime.Runtime for testing.
type MockRuntime struct {
	mu           sync.Mutex
	containers   map[string]*containerEntry
	networks     map[string]bool
	volumes      map[string]bool
	connectErr   error
	checkErr     error
	pullErr      error
	callCount    map[string]int
}

func NewMockRuntime() *MockRuntime {
	return &MockRuntime{
		containers: make(map[string]*containerEntry),
		networks:   make(map[string]bool),
		volumes:    make(map[string]bool),
		callCount:  make(map[string]int),
	}
}

func (m *MockRuntime) inc(method string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount[method]++
}

func (m *MockRuntime) CallCount(method string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount[method]
}

func (m *MockRuntime) SetConnectErr(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connectErr = err
}

func (m *MockRuntime) SetCheckErr(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkErr = err
}

func (m *MockRuntime) AddContainer(id, name, image, status string, labels map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.containers[id] = &containerEntry{
		info: runtime.ContainerInfo{
			ID:     id,
			Name:   name,
			Image:  image,
			Status: status,
			Labels: labels,
		},
		status: status,
		labels: labels,
	}
}

func (m *MockRuntime) Connect(ctx context.Context) error {
	m.inc("Connect")
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connectErr
}

func (m *MockRuntime) Close() error {
	m.inc("Close")
	return nil
}

func (m *MockRuntime) Check(ctx context.Context) error {
	m.inc("Check")
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.checkErr
}

func (m *MockRuntime) PullImage(ctx context.Context, image string) error {
	m.inc("PullImage")
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pullErr
}

func (m *MockRuntime) BuildImage(ctx context.Context, cfg runtime.BuildConfig, statusChan chan<- string) (string, error) {
	m.inc("BuildImage")
	return "mock-image-id", nil
}

func (m *MockRuntime) CreateContainer(ctx context.Context, cfg runtime.ContainerConfig) (string, error) {
	m.inc("CreateContainer")
	id := fmt.Sprintf("mock-%d", time.Now().UnixNano())
	m.mu.Lock()
	defer m.mu.Unlock()
	m.containers[id] = &containerEntry{
		info: runtime.ContainerInfo{
			ID:     id,
			Name:   cfg.Name,
			Image:  cfg.Image,
			Status: "created",
			Labels: cfg.Labels,
		},
		status: "created",
		labels: cfg.Labels,
	}
	return id, nil
}

func (m *MockRuntime) StartContainer(ctx context.Context, id string) error {
	m.inc("StartContainer")
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.containers[id]
	if !ok {
		return fmt.Errorf("container %s not found", id)
	}
	e.status = "running"
	e.info.Status = "running"
	e.info.Health = "healthy"
	return nil
}

func (m *MockRuntime) StopContainer(ctx context.Context, id string, timeoutSeconds int) error {
	m.inc("StopContainer")
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.containers[id]
	if !ok {
		return fmt.Errorf("container %s not found", id)
	}
	e.status = "exited"
	e.info.Status = "exited"
	return nil
}

func (m *MockRuntime) RemoveContainer(ctx context.Context, id string, force bool) error {
	m.inc("RemoveContainer")
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.containers, id)
	return nil
}

func (m *MockRuntime) GetContainerInfo(ctx context.Context, id string) (runtime.ContainerInfo, error) {
	m.inc("GetContainerInfo")
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.containers[id]
	if !ok {
		return runtime.ContainerInfo{}, fmt.Errorf("container %s not found", id)
	}
	return e.info, nil
}

func (m *MockRuntime) ListContainers(ctx context.Context, labels map[string]string) ([]runtime.ContainerInfo, error) {
	m.inc("ListContainers")
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []runtime.ContainerInfo
outer:
	for _, e := range m.containers {
		if labels != nil {
			for k, v := range labels {
				if e.labels[k] != v {
					continue outer
				}
			}
		}
		result = append(result, e.info)
	}
	return result, nil
}

func (m *MockRuntime) StreamLogs(ctx context.Context, id string, opts runtime.LogOptions) (io.ReadCloser, error) {
	m.inc("StreamLogs")
	return io.NopCloser(strings.NewReader("mock log line 1\nmock log line 2\n")), nil
}

func (m *MockRuntime) CreateNetwork(ctx context.Context, name string) error {
	m.inc("CreateNetwork")
	m.mu.Lock()
	defer m.mu.Unlock()
	m.networks[name] = true
	return nil
}

func (m *MockRuntime) RemoveNetwork(ctx context.Context, name string) error {
	m.inc("RemoveNetwork")
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.networks, name)
	return nil
}

func (m *MockRuntime) NetworkExists(ctx context.Context, name string) (bool, error) {
	m.inc("NetworkExists")
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.networks[name]
	return ok, nil
}

func (m *MockRuntime) CreateVolume(ctx context.Context, name string) error {
	m.inc("CreateVolume")
	m.mu.Lock()
	defer m.mu.Unlock()
	m.volumes[name] = true
	return nil
}

func (m *MockRuntime) RemoveVolume(ctx context.Context, name string) error {
	m.inc("RemoveVolume")
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.volumes, name)
	return nil
}

func (m *MockRuntime) VolumeExists(ctx context.Context, name string) (bool, error) {
	m.inc("VolumeExists")
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.volumes[name]
	return ok, nil
}

func (m *MockRuntime) VolumePath(ctx context.Context, name string) (string, error) {
	m.inc("VolumePath")
	return "", nil
}
