package orchestrator

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/devboxos/devboxos/engine/internal/networking"
	"github.com/devboxos/devboxos/shared/logging"
	"github.com/devboxos/devboxos/shared/plugins"
	"github.com/devboxos/devboxos/shared/runtime"
	"github.com/devboxos/devboxos/shared/secrets"
	"github.com/devboxos/devboxos/shared/types"
)

// Orchestrator manages the full lifecycle of a DevBoxOS environment.
type Orchestrator struct {
	rt          runtime.Runtime
	lifecycle   *Lifecycle
	resolver    *secrets.Resolver
	pluginMgr   *plugins.Manager
	logStore    *logging.Store
	collectors  map[string]*logging.Collector
	environment *types.EnvironmentStatus
	mu          sync.Mutex
}

// NewOrchestrator creates a new environment orchestrator.
func NewOrchestrator(rt runtime.Runtime, projectPath string, cfg *types.Config) (*Orchestrator, error) {
	keyPath := filepath.Join(projectPath, ".devbox", "secrets.key")
	storePath := filepath.Join(projectPath, ".devbox", "secrets.enc")

	resolver, err := secrets.NewResolver(projectPath, keyPath, storePath)
	if err != nil {
		return nil, fmt.Errorf("create secrets resolver: %w", err)
	}

	return &Orchestrator{
		rt:         rt,
		lifecycle:  NewLifecycle(rt, resolver),
		resolver:   resolver,
		pluginMgr:  plugins.NewManager(projectPath, cfg.Plugins),
		logStore:   logging.NewStore(projectPath),
		collectors: make(map[string]*logging.Collector),
		environment: &types.EnvironmentStatus{
			Status: "stopped",
		},
	}, nil
}

// Start starts all services in dependency order.
func (o *Orchestrator) Start(ctx context.Context, cfg *types.Config, projectPath string, statusChan chan<- string) error {
	o.mu.Lock()
	o.environment.Name = cfg.Name
	o.environment.Path = projectPath
	o.environment.Status = "starting"
	o.environment.Services = nil
	o.mu.Unlock()

	// Step 1: Build dependency graph
	graph := NewGraph()
	for name, svc := range cfg.Services {
		graph.AddNode(name, svc.DependsOn)
	}

	startOrder, err := graph.Resolve()
	if err != nil {
		return fmt.Errorf("resolve dependencies: %w", err)
	}

	// Step 2: Check port conflicts
	statusChan <- "Checking port availability..."
	if err := o.checkPortConflicts(cfg); err != nil {
		return fmt.Errorf("port conflict: %w", err)
	}

	// Step 2.5: Run pre-start plugins
	if o.pluginMgr.HasHook(plugins.HookPreStart) {
		statusChan <- "Running pre-start plugins..."
		if err := o.pluginMgr.ExecuteHook(ctx, plugins.HookPreStart, map[string]string{
			"DEVBOX_PROJECT_NAME": cfg.Name,
		}); err != nil {
			statusChan <- fmt.Sprintf("Warning: pre-start plugin failed: %v", err)
		}
	}

	// Step 3: Create/verify project network
	statusChan <- fmt.Sprintf("Setting up network for %s...", cfg.Name)
	netMgr := networking.NewManager(o.rt)
	nw, err := netMgr.EnsureNetwork(ctx, cfg.Name)
	if err != nil {
		return fmt.Errorf("setup network: %w", err)
	}

	statusChan <- fmt.Sprintf("Network: %s (%s)", nw.Name, nw.Subnet)

	// Step 4: Initialize DNS resolver
	dns := networking.NewDNSResolver()

	// Step 5: Initialize mTLS (if enabled)
	if cfg.Security.TLS == "" || cfg.Security.TLS == "mTLS" {
		statusChan <- "Generating mTLS certificates..."
		_, err = networking.NewMTLSManager(cfg.Name)
		if err != nil {
			statusChan <- fmt.Sprintf("Warning: mTLS setup failed: %v", err)
		} else {
			statusChan <- "mTLS certificates generated"
		}
	} else {
		statusChan <- "mTLS disabled"
	}

	// Step 6: Initialize egress policy
	egressMode := "default-deny"
	if cfg.Networking.Egress == "allow-all" {
		egressMode = "allow-all"
	}
	egress := networking.NewEgressPolicy(egressMode)
	statusChan <- fmt.Sprintf("Egress policy: %s", egress.GetMode())

	// Step 7: Start services in order
	containerIDs := make(map[string]string)
	for _, name := range startOrder {
		svc, ok := cfg.Services[name]
		if !ok {
			continue
		}

		statusChan <- fmt.Sprintf("Starting service: %s", name)

		// Check port conflicts for this specific service
		if svc.Port != "" {
			hostPort := svc.Port
			if idx := len(svc.Port) - 1; idx > 0 {
				for i := idx; i >= 0; i-- {
					if svc.Port[i] == ':' {
						hostPort = svc.Port[:i]
						break
					}
				}
			}
			if err := networking.CheckPortAvailability(hostPort); err != nil {
				o.cleanup(ctx, containerIDs)
				return fmt.Errorf("service %s: %w", name, err)
			}
		}

		// Build container config with networking
		containerID, err := o.lifecycle.StartService(ctx, name, svc, nw.Name, projectPath, statusChan)
		if err != nil {
			o.cleanup(ctx, containerIDs)
			return fmt.Errorf("start service %s: %w", name, err)
		}

		containerIDs[name] = containerID
		nw.RegisterContainer(name, containerID)

		// Register DNS entry
		dns.RegisterService(name, "127.0.0.1", nw.Domain)

		// Wait for health check
		statusChan <- fmt.Sprintf("Waiting for %s to be healthy...", name)
		if err := o.lifecycle.WaitForHealthy(ctx, containerID, svc); err != nil {
			statusChan <- fmt.Sprintf("Warning: %s health check: %v", name, err)
		}

		// Print hostname
		hostname := nw.GetHostname(name)
		if svc.Port != "" {
			statusChan <- fmt.Sprintf("Service %s started: %s", name, hostname)
		} else {
			statusChan <- fmt.Sprintf("Service %s started: %s", name, hostname)
		}

		// Start log collector (use background context since request context will be cancelled)
		collector := logging.NewCollector(o.logStore, cfg.Name, name)
		o.mu.Lock()
		o.collectors[name] = collector
		o.mu.Unlock()

		// Start collection safely; the orchestrator tracks collectors via the mutex above.
		collector.Start(context.Background(), o.rt, containerID)
	}

	// Update status
	o.updateStatus(ctx, containerIDs)
	o.mu.Lock()
	o.environment.Status = "running"
	o.mu.Unlock()

	// Run post-start plugins
	if o.pluginMgr.HasHook(plugins.HookPostStart) {
		statusChan <- "Running post-start plugins..."
		if err := o.pluginMgr.ExecuteHook(ctx, plugins.HookPostStart, map[string]string{
			"DEVBOX_PROJECT_NAME": cfg.Name,
		}); err != nil {
			statusChan <- fmt.Sprintf("Warning: post-start plugin failed: %v", err)
		}
	}

	statusChan <- "All services started"
	return nil
}

// Stop stops all services in reverse dependency order.
func (o *Orchestrator) Stop(ctx context.Context, cfg *types.Config, gracePeriod int, statusChan chan<- string) error {
	o.mu.Lock()
	o.environment.Status = "stopping"
	o.mu.Unlock()

	// Run pre-stop plugins
	if o.pluginMgr.HasHook(plugins.HookPreStop) {
		statusChan <- "Running pre-stop plugins..."
		if err := o.pluginMgr.ExecuteHook(ctx, plugins.HookPreStop, map[string]string{
			"DEVBOX_PROJECT_NAME": cfg.Name,
		}); err != nil {
			statusChan <- fmt.Sprintf("Warning: pre-stop plugin failed: %v", err)
		}
	}

	// Build dependency graph for reverse order
	graph := NewGraph()
	for name, svc := range cfg.Services {
		graph.AddNode(name, svc.DependsOn)
	}

	stopOrder, err := graph.Reverse()
	if err != nil {
		return fmt.Errorf("resolve stop order: %w", err)
	}

	// Stop services in reverse order
	for _, name := range stopOrder {
		_, ok := cfg.Services[name]
		if !ok {
			continue
		}

		statusChan <- fmt.Sprintf("Stopping service: %s", name)

		containers, err := o.rt.ListContainers(ctx, map[string]string{
			"devboxos.service": name,
		})
		if err != nil {
			statusChan <- fmt.Sprintf("Warning: could not list containers for %s: %v", name, err)
			continue
		}

		for _, c := range containers {
			if err := o.lifecycle.StopService(ctx, c.ID, gracePeriod); err != nil {
				statusChan <- fmt.Sprintf("Warning: could not stop %s: %v", name, err)
			}
		}

		statusChan <- fmt.Sprintf("Service %s stopped", name)

		// Stop log collector
		o.mu.Lock()
		if collector, ok := o.collectors[name]; ok {
			collector.Stop()
			delete(o.collectors, name)
		}
		o.mu.Unlock()
	}

	o.mu.Lock()
	o.environment.Status = "stopped"
	o.environment.Services = nil
	o.mu.Unlock()

	// Run post-stop plugins
	if o.pluginMgr.HasHook(plugins.HookPostStop) {
		statusChan <- "Running post-stop plugins..."
		if err := o.pluginMgr.ExecuteHook(ctx, plugins.HookPostStop, map[string]string{
			"DEVBOX_PROJECT_NAME": cfg.Name,
		}); err != nil {
			statusChan <- fmt.Sprintf("Warning: post-stop plugin failed: %v", err)
		}
	}

	statusChan <- "All services stopped"
	return nil
}

// Status returns the current environment status.
func (o *Orchestrator) Status(ctx context.Context, cfg *types.Config) (*types.EnvironmentStatus, error) {
	containerIDs := make(map[string]string)
	runningCount := 0

	for name := range cfg.Services {
		containers, err := o.rt.ListContainers(ctx, map[string]string{
			"devboxos.service": name,
		})
		if err != nil {
			continue
		}
		for _, c := range containers {
			containerIDs[name] = c.ID
			if c.Status == "running" {
				runningCount++
			}
		}
	}

	o.updateStatus(ctx, containerIDs)

	if runningCount == 0 {
		o.environment.Status = "stopped"
	} else if runningCount == len(cfg.Services) {
		o.environment.Status = "running"
	} else {
		o.environment.Status = "partial"
	}

	return o.environment, nil
}

// updateStatus refreshes the environment status from Docker.
func (o *Orchestrator) updateStatus(ctx context.Context, containerIDs map[string]string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.environment.Services = nil

	for name, id := range containerIDs {
		info, err := o.rt.GetContainerInfo(ctx, id)
		if err != nil {
			o.environment.Services = append(o.environment.Services, types.ServiceStatus{
				Name:   name,
				Status: "unknown",
				Health: "unknown",
			})
			continue
		}

		var port int
		if len(info.Ports) > 0 {
			fmt.Sscanf(info.Ports[0].HostPort, "%d", &port)
		}

		o.environment.Services = append(o.environment.Services, types.ServiceStatus{
			Name:        name,
			Status:      info.Status,
			Health:      info.Health,
			Port:        port,
			ContainerID: info.ID[:12],
		})
	}
}

// checkPortConflicts checks all ports in the config for conflicts.
func (o *Orchestrator) checkPortConflicts(cfg *types.Config) error {
	for name, svc := range cfg.Services {
		if svc.Port == "" {
			continue
		}
		// Extract host port from "host:container" or just "port"
		hostPort := svc.Port
		if idx := len(svc.Port) - 1; idx > 0 {
			for i := idx; i >= 0; i-- {
				if svc.Port[i] == ':' {
					hostPort = svc.Port[:i]
					break
				}
			}
		}
		if err := networking.CheckPortAvailability(hostPort); err != nil {
			return fmt.Errorf("service %s: %w", name, err)
		}
	}
	return nil
}

// cleanup stops and removes started containers on failure.
func (o *Orchestrator) cleanup(ctx context.Context, containerIDs map[string]string) {
	for name, id := range containerIDs {
		_ = o.lifecycle.StopService(ctx, id, 10)
		_ = o.lifecycle.RemoveService(ctx, id)
		_ = fmt.Sprintf("Cleaned up container: %s", name)
	}
}
