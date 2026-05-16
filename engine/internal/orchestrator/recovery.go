package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/devboxos/devboxos/engine/internal/runtime"
	"github.com/devboxos/devboxos/shared/secrets"
	"github.com/devboxos/devboxos/shared/types"
)

// RecoveryManager handles service restart and recovery policies.
type RecoveryManager struct {
	rt       runtime.Runtime
	resolver *secrets.Resolver
}

// NewRecoveryManager creates a new recovery manager.
func NewRecoveryManager(rt runtime.Runtime, resolver *secrets.Resolver) *RecoveryManager {
	return &RecoveryManager{rt: rt, resolver: resolver}
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
		if err := r.rt.RemoveContainer(ctx, containerID, true); err != nil {
			// Container may already be removed
		}

		// Recreate and restart
		lifecycle := NewLifecycle(r.rt, r.resolver)
		newID, err := lifecycle.StartService(ctx, name, svc, networkName)
		if err != nil {
			if attempt == maxRetries {
				return fmt.Errorf("failed to restart %s after %d attempts: %w", name, maxRetries, err)
			}
			continue
		}

		// Wait for health
		if err := lifecycle.WaitForHealthy(ctx, newID, svc); err != nil {
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

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			info, err := r.rt.GetContainerInfo(ctx, containerID)
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
