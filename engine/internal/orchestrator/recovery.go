package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/devboxos/devboxos/shared/config"
	"github.com/devboxos/devboxos/shared/runtime"
	"github.com/devboxos/devboxos/shared/secrets"
	"github.com/devboxos/devboxos/shared/types"
)

// RecoveryManager handles service restart and recovery policies.
type RecoveryManager struct {
	dockerRT    runtime.Runtime
	hostRT      runtime.Runtime
	resolver    *secrets.Resolver
	projectPath string
}

// NewRecoveryManager creates a new recovery manager.
func NewRecoveryManager(dockerRT, hostRT runtime.Runtime, resolver *secrets.Resolver, projectPath string) *RecoveryManager {
	return &RecoveryManager{dockerRT: dockerRT, hostRT: hostRT, resolver: resolver, projectPath: projectPath}
}

// runtimeForService picks the right runtime for a service.
func (r *RecoveryManager) runtimeForService(svc types.Service) runtime.Runtime {
	if config.NeedsDockerService(svc) && r.dockerRT != nil {
		return r.dockerRT
	}
	return r.hostRT
}

// ApplyRestartPolicy applies the restart policy for a failed service.
func (r *RecoveryManager) ApplyRestartPolicy(ctx context.Context, name string, svc types.Service, containerID string, networkName string) error {
	if svc.RestartPolicy == nil {
		return nil
	}

	policy := svc.RestartPolicy

	if !policy.OnFailure && !policy.Always {
		return nil
	}

	maxRetries := policy.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	backoff := policy.Backoff
	if backoff == "" {
		backoff = "linear"
	}

	rt := r.runtimeForService(svc)

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Calculate backoff duration
		var wait time.Duration
		switch backoff {
		case "exponential":
			wait = time.Duration(attempt*attempt) * time.Second
		case "linear":
			wait = time.Duration(attempt) * 5 * time.Second
		default:
			wait = 5 * time.Second
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}

		// Remove old container
		if err := rt.RemoveContainer(ctx, containerID, true); err != nil {
			// Container may already be removed
		}

		// Recreate and restart
		lifecycle := NewLifecycle(r.resolver)
		discardChan := make(chan string, 64)
		newID, err := lifecycle.StartService(ctx, rt, name, svc, networkName, r.projectPath, discardChan)
		if err != nil {
			if attempt == maxRetries {
				return fmt.Errorf("failed to restart %s after %d attempts: %w", name, maxRetries, err)
			}
			continue
		}

		// Wait for health
		if err := lifecycle.WaitForHealthy(ctx, rt, newID, svc); err != nil {
			if attempt == maxRetries {
				return fmt.Errorf("service %s failed health check after restart: %w", name, err)
			}
			containerID = newID
			continue
		}

		return nil
	}

	return fmt.Errorf("service %s exhausted all restart attempts", name)
}

// Monitor watches a service and applies restart policy on failure.
func (r *RecoveryManager) Monitor(ctx context.Context, name string, svc types.Service, containerID string, networkName string) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	rt := r.runtimeForService(svc)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			info, err := rt.GetContainerInfo(ctx, containerID)
			if err != nil {
				continue
			}

			if info.Status == "exited" || info.Status == "dead" {
				if err := r.ApplyRestartPolicy(ctx, name, svc, containerID, networkName); err != nil {
					// Log error but continue monitoring
					containerID = ""
				}
			}
		}
	}
}
