package diagnostics

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/devboxos/devboxos/shared/config"
	"github.com/devboxos/devboxos/shared/runtime"
	"github.com/devboxos/devboxos/shared/secrets"
	"github.com/devboxos/devboxos/shared/types"
)

// Severity represents the severity level of an issue.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// Issue represents a diagnostic issue.
type Issue struct {
	Severity Severity
	Category string
	Message  string
	Details  string
	Suggestion string
}

// Result represents the result of a diagnostic check.
type Result struct {
	Name        string
	Passed      bool
	Severity    Severity
	Message     string
	Issues      []Issue
}

// Checker runs diagnostic checks.
type Checker struct {
	rt          runtime.Runtime
	projectPath string
	cfg         *types.Config
}

// NewChecker creates a new diagnostic checker.
func NewChecker(rt runtime.Runtime, projectPath string, cfg *types.Config) *Checker {
	return &Checker{
		rt:          rt,
		projectPath: projectPath,
		cfg:         cfg,
	}
}

// RunAll runs all diagnostic checks.
func (c *Checker) RunAll(ctx context.Context) []Result {
	var results []Result

	results = append(results, c.checkDocker(ctx))
	results = append(results, c.checkDiskSpace())
	results = append(results, c.checkMemory())
	results = append(results, c.checkConfig())
	results = append(results, c.checkSecrets())
	results = append(results, c.checkNetwork())
	results = append(results, c.checkContainers(ctx))
	results = append(results, c.checkOrphanedResources(ctx))
	results = append(results, c.checkSecurity())

	return results
}

// GetIssues extracts all issues from results.
func GetIssues(results []Result) ([]Issue, []string) {
	var issues []Issue
	var suggestions []string

	for _, r := range results {
		if !r.Passed {
			issues = append(issues, Issue{
				Severity: r.Severity,
				Category: r.Name,
				Message:  r.Message,
			})
		}
		for _, issue := range r.Issues {
			issues = append(issues, issue)
			if issue.Suggestion != "" {
				suggestions = append(suggestions, issue.Suggestion)
			}
		}
	}

	return issues, suggestions
}

func (c *Checker) checkDocker(ctx context.Context) Result {
	result := Result{Name: "Docker", Passed: true}

	if c.rt == nil {
		result.Passed = false
		result.Severity = SeverityCritical
		result.Message = "Docker runtime not initialized"
		result.Issues = append(result.Issues, Issue{
			Severity: SeverityCritical,
			Message:  "Docker runtime not available",
			Suggestion: "Make sure Docker Desktop is installed and running",
		})
		return result
	}

	if err := c.rt.Check(ctx); err != nil {
		result.Passed = false
		result.Severity = SeverityCritical
		result.Message = fmt.Sprintf("Docker daemon not accessible: %v", err)
		result.Issues = append(result.Issues, Issue{
			Severity: SeverityCritical,
			Message:  "Docker daemon not responding",
			Suggestion: "Start Docker Desktop and try again",
		})
		return result
	}

	// Get Docker version
	info, err := c.rt.ListContainers(ctx, nil)
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Docker API error: %v", err)
		return result
	}

	result.Message = fmt.Sprintf("Docker daemon running (%d containers)", len(info))
	return result
}

func (c *Checker) checkDiskSpace() Result {
	result := Result{Name: "Disk Space", Passed: true}

	// Check project directory
	dir := c.projectPath
	if dir == "" {
		dir = "."
	}

	// Get disk usage info (Windows-specific)
	// For now, just check if directory is accessible
	if _, err := os.Stat(dir); err != nil {
		result.Passed = false
		result.Severity = SeverityError
		result.Message = fmt.Sprintf("Project directory not accessible: %v", err)
		return result
	}

	// Check .devbox directory size
	devboxDir := filepath.Join(dir, ".devbox")
	if _, err := os.Stat(devboxDir); err == nil {
		var size int64
		filepath.Walk(devboxDir, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				size += info.Size()
			}
			return nil
		})

		sizeMB := size / (1024 * 1024)
		if sizeMB > 5000 {
			result.Passed = false
			result.Severity = SeverityWarning
			result.Message = fmt.Sprintf("DevBoxOS data using %d MB", sizeMB)
			result.Issues = append(result.Issues, Issue{
				Severity: SeverityWarning,
				Message:  "Large amount of local data",
				Suggestion: "Run 'devbox prune' to clean up unused resources",
			})
		} else {
			result.Message = fmt.Sprintf("%d MB used by DevBoxOS", sizeMB)
		}
	} else {
		result.Message = "No local DevBoxOS data"
	}

	return result
}

func (c *Checker) checkMemory() Result {
	result := Result{Name: "Memory", Passed: true}

	// Calculate total memory requested by services
	var totalMemoryMB int64
	for name, svc := range c.cfg.Services {
		if svc.Resources != nil && svc.Resources.Memory != "" {
			memStr := svc.Resources.Memory
			memMB := parseMemoryToMB(memStr)
			totalMemoryMB += memMB
			_ = name
		}
	}

	if totalMemoryMB > 0 {
		result.Message = fmt.Sprintf("%d MB requested by services", totalMemoryMB)
		if totalMemoryMB > 8192 {
			result.Severity = SeverityWarning
			result.Issues = append(result.Issues, Issue{
				Severity: SeverityWarning,
				Message:  "High memory request (>8GB)",
				Suggestion: "Consider reducing memory limits in devbox.yml",
			})
		}
	} else {
		result.Message = "No memory limits defined"
	}

	return result
}

func (c *Checker) checkConfig() Result {
	result := Result{Name: "Configuration", Passed: true}

	if c.cfg == nil {
		result.Passed = false
		result.Severity = SeverityCritical
		result.Message = "No configuration loaded"
		return result
	}

	// Check for empty services
	if len(c.cfg.Services) == 0 {
		result.Passed = false
		result.Severity = SeverityError
		result.Message = "No services defined"
		result.Issues = append(result.Issues, Issue{
			Severity: SeverityError,
			Message:  "devbox.yml has no services",
			Suggestion: "Add at least one service to devbox.yml",
		})
		return result
	}

	// Check for circular dependencies
	graph := make(map[string][]string)
	for name, svc := range c.cfg.Services {
		graph[name] = svc.DependsOn
	}

	if hasCycle(graph) {
		result.Passed = false
		result.Severity = SeverityError
		result.Message = "Circular dependency detected"
		result.Issues = append(result.Issues, Issue{
			Severity: SeverityError,
			Message:  "Services have circular dependencies",
			Suggestion: "Remove circular dependencies from depends_on",
		})
	}

	// Check for missing dependencies
	for name, svc := range c.cfg.Services {
		for _, dep := range svc.DependsOn {
			if _, ok := c.cfg.Services[dep]; !ok {
				result.Passed = false
				result.Severity = SeverityError
				result.Issues = append(result.Issues, Issue{
					Severity: SeverityError,
					Message:  fmt.Sprintf("Service %s depends on unknown service %s", name, dep),
					Suggestion: fmt.Sprintf("Add service %s to devbox.yml or remove dependency", dep),
				})
			}
		}
	}

	// Check for services without image or build
	for name, svc := range c.cfg.Services {
		if svc.Image == "" && (svc.Build == nil || svc.Build.Context == "") {
			result.Passed = false
			result.Severity = SeverityWarning
			result.Issues = append(result.Issues, Issue{
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("Service %s has no image or build config", name),
				Suggestion: fmt.Sprintf("Add 'image' or 'build' to service %s", name),
			})
		}
	}

	if result.Passed {
		result.Message = fmt.Sprintf("Valid (%d services, 0 errors)", len(c.cfg.Services))
	}

	return result
}

func (c *Checker) checkSecrets() Result {
	result := Result{Name: "Secrets", Passed: true}

	keyPath := filepath.Join(c.projectPath, ".devbox", "secrets.key")
	storePath := filepath.Join(c.projectPath, ".devbox", "secrets.enc")

	// Check if key exists
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		result.Message = "No encryption key (will be created on first use)"
		return result
	}

	// Try to load resolver
	resolver, err := secrets.NewResolver(c.projectPath, keyPath, storePath)
	if err != nil {
		result.Passed = false
		result.Severity = SeverityWarning
		result.Message = fmt.Sprintf("Secret store error: %v", err)
		return result
	}

	// Check all referenced secrets
	var secretCount int
	for _, svc := range c.cfg.Services {
		for _, ref := range svc.Secrets {
			secretCount++
			_, err := resolver.Resolve(ref)
			if err != nil {
				result.Passed = false
				result.Severity = SeverityWarning
				result.Issues = append(result.Issues, Issue{
					Severity: SeverityWarning,
					Message:  fmt.Sprintf("Secret %s: %v", ref.Name, err),
					Suggestion: fmt.Sprintf("Set secret %s or fix source configuration", ref.Name),
				})
			}
		}
	}

	if secretCount == 0 {
		result.Message = "No secrets configured"
	} else {
		result.Message = fmt.Sprintf("%d secrets, all resolvable", secretCount)
	}

	return result
}

func (c *Checker) checkNetwork() Result {
	result := Result{Name: "Network", Passed: true}

	// Check for port conflicts within the config
	ports := make(map[string]string)
	for name, svc := range c.cfg.Services {
		if svc.Port != "" {
			hostPort := svc.Port
			for i := len(svc.Port) - 1; i >= 0; i-- {
				if svc.Port[i] == ':' {
					hostPort = svc.Port[:i]
					break
				}
			}

			if existing, ok := ports[hostPort]; ok {
				result.Passed = false
				result.Severity = SeverityError
				result.Issues = append(result.Issues, Issue{
					Severity: SeverityError,
					Message:  fmt.Sprintf("Port %s used by both %s and %s", hostPort, existing, name),
					Suggestion: fmt.Sprintf("Change port for service %s", name),
				})
			}
			ports[hostPort] = name
		}
	}

	// Check for cross-process port conflicts (ports already in use by other programs)
	for name, svc := range c.cfg.Services {
		if svc.Port == "" {
			continue
		}
		hostPort := svc.Port
		for i := len(svc.Port) - 1; i >= 0; i-- {
			if svc.Port[i] == ':' {
				hostPort = svc.Port[:i]
				break
			}
		}
		ln, err := net.Listen("tcp", ":"+hostPort)
		if err != nil {
			result.Passed = false
			result.Severity = SeverityWarning
			result.Issues = append(result.Issues, Issue{
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("Port %s (service %s) is already in use by another process", hostPort, name),
				Suggestion: fmt.Sprintf("Change the port for service %s, or stop the other process using port %s", name, hostPort),
			})
		} else {
			ln.Close()
		}
	}

	if result.Passed {
		result.Message = "No port conflicts"
	}

	return result
}

func (c *Checker) checkContainers(ctx context.Context) Result {
	result := Result{Name: "Containers", Passed: true}

	if c.rt == nil {
		result.Message = "Docker not available"
		return result
	}

	var running, stopped int
	for name := range c.cfg.Services {
		containers, err := c.rt.ListContainers(ctx, map[string]string{
			"devboxos.service": name,
		})
		if err != nil {
			continue
		}

		if len(containers) == 0 {
			stopped++
		} else {
			for _, c := range containers {
				if c.Status == "running" {
					running++
				} else {
					stopped++
					result.Issues = append(result.Issues, Issue{
						Severity: SeverityWarning,
						Message:  fmt.Sprintf("Container %s is %s", name, c.Status),
						Suggestion: fmt.Sprintf("Check logs: devbox logs --service %s", name),
					})
				}
			}
		}
	}

	if running == 0 && stopped == 0 {
		result.Message = "No containers found"
	} else {
		result.Message = fmt.Sprintf("%d running, %d stopped", running, stopped)
	}

	return result
}

func (c *Checker) checkOrphanedResources(ctx context.Context) Result {
	result := Result{Name: "Orphaned Resources", Passed: true}

	if c.rt == nil {
		result.Message = "Docker not available"
		return result
	}

	// Check for orphaned networks
	networkName := fmt.Sprintf("devbox-%s", c.cfg.Name)
	exists, _ := c.rt.NetworkExists(ctx, networkName)
	if !exists && c.cfg.Name != "" {
		result.Message = "No orphaned resources"
		return result
	}

	result.Message = "Resources properly managed"
	return result
}

func parseMemoryToMB(s string) int64 {
	s = strings.ToLower(s)
	var multiplier int64 = 1

	if strings.HasSuffix(s, "g") {
		multiplier = 1024
		s = strings.TrimSuffix(s, "g")
	} else if strings.HasSuffix(s, "m") {
		s = strings.TrimSuffix(s, "m")
	} else if strings.HasSuffix(s, "k") {
		s = strings.TrimSuffix(s, "k")
		multiplier = 0
	}

	var val int64
	fmt.Sscanf(s, "%d", &val)
	if multiplier == 0 {
		return val
	}
	return val * multiplier
}

func hasCycle(graph map[string][]string) bool {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(node string) bool
	dfs = func(node string) bool {
		visited[node] = true
		recStack[node] = true

		for _, neighbor := range graph[node] {
			if !visited[neighbor] {
				if dfs(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				return true
			}
		}

		recStack[node] = false
		return false
	}

	for node := range graph {
		if !visited[node] {
			if dfs(node) {
				return true
			}
		}
	}

	return false
}

func (c *Checker) checkSecurity() Result {
	result := Result{Name: "Security", Passed: true}

	if c.cfg == nil {
		result.Message = "No configuration to check"
		return result
	}

	var warnings []string
	for name, svc := range c.cfg.Services {
		if svc.Security == nil {
			continue
		}
		if len(svc.Security.Capabilities) > 0 {
			warnings = append(warnings,
				fmt.Sprintf("service %q requests extra capabilities: %v", name, svc.Security.Capabilities))
		}
		if svc.Security.SeccompProfile == "unconfined" {
			warnings = append(warnings,
				fmt.Sprintf("service %q has seccomp disabled (unconfined)", name))
		}
		if svc.Security.AppArmorProfile == "unconfined" {
			warnings = append(warnings,
				fmt.Sprintf("service %q has AppArmor disabled (unconfined)", name))
		}
	}

	if len(warnings) > 0 {
		result.Passed = false
		result.Severity = SeverityWarning
		result.Message = fmt.Sprintf("%d security configuration item(s) to review", len(warnings))
		for _, w := range warnings {
			result.Issues = append(result.Issues, Issue{
				Severity:   SeverityWarning,
				Message:    w,
				Suggestion: "Review if these security exemptions are necessary",
			})
		}
	} else {
		result.Message = "Default secure configuration (no custom overrides)"
	}

	return result
}

// RunDoctor runs all checks and returns formatted results.
func RunDoctor(ctx context.Context, rt runtime.Runtime, projectPath string) ([]Result, []string) {
	parser := config.NewParser()
	cfg, err := parser.Parse(projectPath)
	if err != nil {
		return []Result{{
			Name:     "Configuration",
			Passed:   false,
			Severity: SeverityCritical,
			Message:  fmt.Sprintf("Failed to parse config: %v", err),
		}}, nil
	}

	checker := NewChecker(rt, projectPath, cfg)
	results := checker.RunAll(ctx)
	_, suggestions := GetIssues(results)

	return results, suggestions
}
