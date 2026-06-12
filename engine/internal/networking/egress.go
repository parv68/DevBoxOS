package networking

import (
	"fmt"
	"net"
	"sync"
)

// EgressPolicy manages outbound traffic rules for services.
type EgressPolicy struct {
	mode       string // "default-deny" or "allow-all"
	allowed    map[string][]string // service -> allowed destinations
	mu         sync.RWMutex
}

// NewEgressPolicy creates a new egress policy manager.
func NewEgressPolicy(mode string) *EgressPolicy {
	if mode == "" {
		mode = "default-deny"
	}
	return &EgressPolicy{
		mode:    mode,
		allowed: make(map[string][]string),
	}
}

// IsAllowed checks if a service can reach a destination.
func (e *EgressPolicy) IsAllowed(service, destination string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.mode == "allow-all" {
		return true
	}

	// Default-deny: check explicit allow list
	allowed, ok := e.allowed[service]
	if !ok {
		return false
	}

	for _, dest := range allowed {
		if dest == "*" || dest == destination {
			return true
		}
	}

	return false
}

// AddRule adds an egress rule for a service.
func (e *EgressPolicy) AddRule(service, destination string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.allowed[service] = append(e.allowed[service], destination)
}

// GetMode returns the policy mode.
func (e *EgressPolicy) GetMode() string {
	return e.mode
}

// CheckPortAvailability checks if a port is available on the host.
func CheckPortAvailability(port string) error {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("port %s is already in use", port)
	}
	ln.Close()
	return nil
}

// FindFreePort finds the next available port starting from startPort.
func FindFreePort(startPort int) (int, error) {
	for port := startPort; port < startPort+1000; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			ln.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no free port found in range %d-%d", startPort, startPort+1000)
}
