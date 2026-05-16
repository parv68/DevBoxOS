package plugins

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/devboxos/devboxos/shared/types"
)

// Hook represents an execution point for plugins.
type Hook string

const (
	HookPreStart   Hook = "pre-start"
	HookPostStart  Hook = "post-start"
	HookPreStop    Hook = "pre-stop"
	HookPostStop   Hook = "post-stop"
	HookOnFailure  Hook = "on-failure"
)

// Plugin represents a single plugin configuration.
type Plugin struct {
	Name    string
	Command string
	On      []Hook
	Env     map[string]string
	Timeout time.Duration
}

// Manager handles plugin execution.
type Manager struct {
	plugins   []Plugin
	projectPath string
}

// NewManager creates a new plugin manager from config.
func NewManager(projectPath string, pluginConfigs []types.Plugin) *Manager {
	m := &Manager{
		projectPath: projectPath,
	}

	for _, pc := range pluginConfigs {
		plugin := Plugin{
			Name:    pc.Name,
			Command: pc.Command,
			Timeout: 30 * time.Second,
		}

		if pc.Timeout > 0 {
			plugin.Timeout = time.Duration(pc.Timeout) * time.Second
		}

		// Parse "on" field
		if len(pc.On) > 0 {
			for _, hook := range pc.On {
				plugin.On = append(plugin.On, Hook(hook))
			}
		}

		// Parse env from config
		if pc.Config != nil {
			if env, ok := pc.Config["env"].(map[string]interface{}); ok {
				plugin.Env = make(map[string]string)
				for k, v := range env {
					plugin.Env[k] = fmt.Sprintf("%v", v)
				}
			}
		}

		m.plugins = append(m.plugins, plugin)
	}

	return m
}

// ExecuteHook runs all plugins registered for a specific hook.
func (m *Manager) ExecuteHook(ctx context.Context, hook Hook, envVars map[string]string) error {
	var executed []string

	for _, plugin := range m.plugins {
		if !plugin.hooksInto(hook) {
			continue
		}

		executed = append(executed, plugin.Name)

		cmdCtx, cancel := context.WithTimeout(ctx, plugin.Timeout)
		defer cancel()

		var cmd *exec.Cmd
		if os.PathSeparator == '\\' {
			cmd = exec.CommandContext(cmdCtx, "cmd", "/c", plugin.Command)
		} else {
			cmd = exec.CommandContext(cmdCtx, "sh", "-c", plugin.Command)
		}
		cmd.Dir = m.projectPath

		// Set environment
		cmd.Env = os.Environ()
		for k, v := range plugin.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
		for k, v := range envVars {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}

		// Set DevBoxOS-specific env vars
		cmd.Env = append(cmd.Env, fmt.Sprintf("DEVBOX_HOOK=%s", hook))
		cmd.Env = append(cmd.Env, fmt.Sprintf("DEVBOX_PROJECT=%s", m.projectPath))
		cmd.Env = append(cmd.Env, fmt.Sprintf("DEVBOX_PLUGIN=%s", plugin.Name))

		output, err := cmd.CombinedOutput()
		if err != nil {
			if cmdCtx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("plugin %s timed out after %s", plugin.Name, plugin.Timeout)
			}
			return fmt.Errorf("plugin %s failed: %v\nOutput: %s", plugin.Name, err, string(output))
		}
	}

	if len(executed) > 0 {
		fmt.Printf("  Executed plugins: %s\n", strings.Join(executed, ", "))
	}

	return nil
}

func (p *Plugin) hooksInto(hook Hook) bool {
	for _, h := range p.On {
		if h == hook {
			return true
		}
	}
	return false
}

// List returns all registered plugins.
func (m *Manager) List() []Plugin {
	return m.plugins
}

// HasHook checks if any plugin is registered for a hook.
func (m *Manager) HasHook(hook Hook) bool {
	for _, plugin := range m.plugins {
		if plugin.hooksInto(hook) {
			return true
		}
	}
	return false
}
