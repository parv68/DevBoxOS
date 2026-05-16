package networking

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/devboxos/devboxos/engine/internal/runtime"
)

// Manager handles per-project Docker network lifecycle.
type Manager struct {
	rt       runtime.Runtime
	networks map[string]*ProjectNetwork
	mu       sync.Mutex
}

// ProjectNetwork represents a project's isolated network.
type ProjectNetwork struct {
	Name        string
	Subnet      string
	Gateway     string
	Domain      string
	Containers  map[string]string // service name -> container ID
	DNSServer   string
}

// NewManager creates a new network manager.
func NewManager(rt runtime.Runtime) *Manager {
	return &Manager{
		rt:       rt,
		networks: make(map[string]*ProjectNetwork),
	}
}

// EnsureNetwork creates or verifies a project network exists.
func (m *Manager) EnsureNetwork(ctx context.Context, projectName string) (*ProjectNetwork, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check cache
	if nw, ok := m.networks[projectName]; ok {
		return nw, nil
	}

	networkName := fmt.Sprintf("devbox-%s", projectName)

	// Check if network already exists in Docker
	exists, err := m.rt.NetworkExists(ctx, networkName)
	if err != nil {
		return nil, fmt.Errorf("check network: %w", err)
	}

	if exists {
		// Network exists, load it
		nw := &ProjectNetwork{
			Name:       networkName,
			Domain:     fmt.Sprintf("%s.local", projectName),
			Containers: make(map[string]string),
		}
		m.networks[projectName] = nw
		return nw, nil
	}

	// Allocate a non-conflicting subnet
	subnet, gateway, err := m.allocateSubnet(ctx)
	if err != nil {
		return nil, fmt.Errorf("allocate subnet: %w", err)
	}

	// Create the network
	if err := m.rt.CreateNetwork(ctx, networkName); err != nil {
		return nil, fmt.Errorf("create network: %w", err)
	}

	nw := &ProjectNetwork{
		Name:       networkName,
		Subnet:     subnet,
		Gateway:    gateway,
		Domain:     fmt.Sprintf("%s.local", projectName),
		Containers: make(map[string]string),
	}

	m.networks[projectName] = nw
	return nw, nil
}

// RemoveNetwork removes a project network.
func (m *Manager) RemoveNetwork(ctx context.Context, projectName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	networkName := fmt.Sprintf("devbox-%s", projectName)
	if err := m.rt.RemoveNetwork(ctx, networkName); err != nil {
		return fmt.Errorf("remove network: %w", err)
	}

	delete(m.networks, projectName)
	return nil
}

// GetHostname returns the .local hostname for a service.
func (nw *ProjectNetwork) GetHostname(serviceName string) string {
	return fmt.Sprintf("%s.%s", serviceName, nw.Domain)
}

// RegisterContainer registers a service container in the network.
func (nw *ProjectNetwork) RegisterContainer(serviceName, containerID string) {
	nw.Containers[serviceName] = containerID
}

// allocateSubnet finds an available 172.x.0.0/16 subnet.
func (m *Manager) allocateSubnet(ctx context.Context) (string, string, error) {
	// Try subnets from 172.20.0.0/16 upwards (Docker default is 172.17.0.0/16)
	for i := 20; i < 30; i++ {
		subnet := fmt.Sprintf("172.%d.0.0/16", i)
		gateway := fmt.Sprintf("172.%d.0.1", i)

		// Check if this subnet conflicts with any existing network
		if m.isSubnetAvailable(ctx, subnet) {
			return subnet, gateway, nil
		}
	}

	return "", "", fmt.Errorf("no available subnets found")
}

// isSubnetAvailable checks if a subnet conflicts with existing networks.
func (m *Manager) isSubnetAvailable(ctx context.Context, targetSubnet string) bool {
	_, targetNet, err := net.ParseCIDR(targetSubnet)
	if err != nil {
		return false
	}

	// Get all Docker networks
	// For now, check against known Docker defaults
	defaults := []string{
		"172.17.0.0/16", // Docker default bridge
		"172.18.0.0/16",
		"172.19.0.0/16",
		"192.168.0.0/16",
		"10.0.0.0/8",
	}

	for _, d := range defaults {
		_, defaultNet, err := net.ParseCIDR(d)
		if err != nil {
			continue
		}
		// Check overlap
		if targetNet.Contains(defaultNet.IP) || defaultNet.Contains(targetNet.IP) {
			return false
		}
	}

	return true
}
