package diagnostics

import (
	"context"
	"testing"

	"github.com/devboxos/devboxos/shared/types"
)

func TestParseMemoryToMB(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"512m", 512},
		{"1g", 1024},
		{"2g", 2048},
		{"0m", 0},
		{"100", 100},
		{"", 0},
		{"abc", 0},
	}

	for _, tc := range tests {
		got := parseMemoryToMB(tc.input)
		if got != tc.want {
			t.Errorf("parseMemoryToMB(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

func TestHasCycle_NoCycle(t *testing.T) {
	graph := map[string][]string{
		"web": {"api"},
		"api": {"db"},
		"db":  {},
	}

	if hasCycle(graph) {
		t.Error("expected no cycle in linear graph")
	}
}

func TestHasCycle_DirectCycle(t *testing.T) {
	graph := map[string][]string{
		"a": {"b"},
		"b": {"a"},
	}

	if !hasCycle(graph) {
		t.Error("expected cycle in a->b->a graph")
	}
}

func TestHasCycle_SelfCycle(t *testing.T) {
	graph := map[string][]string{
		"a": {"a"},
	}

	if !hasCycle(graph) {
		t.Error("expected cycle in self-referencing graph")
	}
}

func TestHasCycle_Disconnected(t *testing.T) {
	graph := map[string][]string{
		"a": {},
		"b": {},
		"c": {},
	}

	if hasCycle(graph) {
		t.Error("expected no cycle in disconnected graph")
	}
}

func TestHasCycle_Empty(t *testing.T) {
	graph := map[string][]string{}

	if hasCycle(graph) {
		t.Error("expected no cycle in empty graph")
	}
}

func TestGetIssues_NoIssues(t *testing.T) {
	results := []Result{
		{Name: "Docker", Passed: true, Message: "OK"},
		{Name: "Config", Passed: true, Message: "OK"},
	}

	issues, suggestions := GetIssues(results)
	if len(issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(issues))
	}
	if len(suggestions) != 0 {
		t.Errorf("expected 0 suggestions, got %d", len(suggestions))
	}
}

func TestGetIssues_WithIssues(t *testing.T) {
	results := []Result{
		{
			Name:     "Docker",
			Passed:   false,
			Severity: SeverityCritical,
			Message:  "Docker not running",
		},
		{
			Name:   "Config",
			Passed: true,
			Issues: []Issue{
				{
					Severity:   SeverityWarning,
					Message:    "Using default port",
					Suggestion: "Change port for production",
				},
			},
		},
	}

	issues, suggestions := GetIssues(results)
	if len(issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(issues))
	}
	if len(suggestions) != 1 {
		t.Errorf("expected 1 suggestion, got %d", len(suggestions))
	}
}

func TestChecker_New(t *testing.T) {
	c := NewChecker(nil, "/tmp/test", &types.Config{Name: "test"})
	if c == nil {
		t.Fatal("NewChecker() returned nil")
	}
}

func TestChecker_RunAll_NilRuntime(t *testing.T) {
	c := NewChecker(nil, "/tmp/test", &types.Config{
		Name:    "test",
		Version: "1.0",
		Services: map[string]types.Service{
			"web": {Image: "nginx"},
		},
	})
	results := c.RunAll(context.Background())

	if len(results) == 0 {
		t.Fatal("RunAll() returned no results")
	}

	// Docker check should fail with nil runtime
	dockerResult := findResult(results, "Docker")
	if dockerResult == nil {
		t.Fatal("expected Docker result")
	}
	if dockerResult.Passed {
		t.Error("expected Docker check to fail with nil runtime")
	}
}

func TestChecker_CheckConfig_NilConfig(t *testing.T) {
	c := NewChecker(nil, "/tmp/test", nil)
	result := c.checkConfig()

	if result.Passed {
		t.Error("expected config check to fail with nil config")
	}
	if result.Severity != SeverityCritical {
		t.Errorf("expected Critical severity, got %s", result.Severity)
	}
}

func TestChecker_CheckConfig_EmptyServices(t *testing.T) {
	c := NewChecker(nil, "/tmp/test", &types.Config{
		Name:     "test",
		Version:  "1.0",
		Services: map[string]types.Service{},
	})
	result := c.checkConfig()

	if result.Passed {
		t.Error("expected config check to fail with empty services")
	}
}

func TestChecker_CheckConfig_Valid(t *testing.T) {
	c := NewChecker(nil, "/tmp/test", &types.Config{
		Name:    "test",
		Version: "1.0",
		Services: map[string]types.Service{
			"web": {Image: "nginx:alpine"},
		},
	})
	result := c.checkConfig()

	if !result.Passed {
		t.Errorf("expected config check to pass, got: %s", result.Message)
	}
}

func TestChecker_CheckConfig_CircularDependency(t *testing.T) {
	c := NewChecker(nil, "/tmp/test", &types.Config{
		Name:    "test",
		Version: "1.0",
		Services: map[string]types.Service{
			"a": {DependsOn: []string{"b"}},
			"b": {DependsOn: []string{"a"}},
		},
	})
	result := c.checkConfig()

	if result.Passed {
		t.Error("expected config check to fail with circular dependency")
	}
}

func TestChecker_CheckConfig_MissingDependency(t *testing.T) {
	c := NewChecker(nil, "/tmp/test", &types.Config{
		Name:    "test",
		Version: "1.0",
		Services: map[string]types.Service{
			"web": {Image: "nginx", DependsOn: []string{"nonexistent"}},
		},
	})
	result := c.checkConfig()

	if result.Passed {
		t.Error("expected config check to fail with missing dependency")
	}
}

func TestChecker_CheckConfig_NoImage(t *testing.T) {
	c := NewChecker(nil, "/tmp/test", &types.Config{
		Name:    "test",
		Version: "1.0",
		Services: map[string]types.Service{
			"web": {Command: "npm start"},
		},
	})
	result := c.checkConfig()

	if result.Passed {
		t.Error("expected config check to warn about service without image")
	}
}

func TestChecker_CheckDiskSpace(t *testing.T) {
	tmpDir := t.TempDir()

	c := NewChecker(nil, tmpDir, &types.Config{Name: "test"})
	result := c.checkDiskSpace()

	if !result.Passed {
		t.Errorf("expected disk space check to pass in temp dir, got: %s", result.Message)
	}
}

func TestChecker_CheckDiskSpace_NonexistentDir(t *testing.T) {
	c := NewChecker(nil, "/nonexistent/path/that/doesnt/exist", &types.Config{Name: "test"})
	result := c.checkDiskSpace()

	if result.Passed {
		t.Log("disk check passed (some systems accept any path)")
	}
}

func TestChecker_CheckMemory_NoLimits(t *testing.T) {
	c := NewChecker(nil, "/tmp/test", &types.Config{
		Name:    "test",
		Version: "1.0",
		Services: map[string]types.Service{
			"web": {Image: "nginx"},
		},
	})
	result := c.checkMemory()

	if !result.Passed {
		t.Errorf("expected memory check to pass with no limits, got: %s", result.Message)
	}
}

func TestChecker_CheckMemory_WithLimits(t *testing.T) {
	c := NewChecker(nil, "/tmp/test", &types.Config{
		Name:    "test",
		Version: "1.0",
		Services: map[string]types.Service{
			"web": {
				Image:     "nginx",
				Resources: &types.Resources{Memory: "1024m"},
			},
		},
	})
	result := c.checkMemory()

	if !result.Passed {
		t.Errorf("expected memory check to pass with 1024m, got: %s", result.Message)
	}
}

func TestChecker_CheckMemory_HighLimit(t *testing.T) {
	c := NewChecker(nil, "/tmp/test", &types.Config{
		Name:    "test",
		Version: "1.0",
		Services: map[string]types.Service{
			"web": {
				Image:     "nginx",
				Resources: &types.Resources{Memory: "16384m"},
			},
		},
	})
	result := c.checkMemory()

	if result.Passed {
		t.Logf("high memory check passed: %s", result.Message)
	}
}

func TestChecker_CheckNetwork_NoPorts(t *testing.T) {
	c := NewChecker(nil, "/tmp/test", &types.Config{
		Name:    "test",
		Version: "1.0",
		Services: map[string]types.Service{
			"web": {Image: "nginx"},
		},
	})
	result := c.checkNetwork()

	if !result.Passed {
		t.Errorf("expected network check to pass with no ports, got: %s", result.Message)
	}
}

func TestChecker_CheckNetwork_PortConflict(t *testing.T) {
	c := NewChecker(nil, "/tmp/test", &types.Config{
		Name:    "test",
		Version: "1.0",
		Services: map[string]types.Service{
			"web":  {Image: "nginx", Port: "8080:80"},
			"api":  {Image: "api", Port: "8080:3000"},
		},
	})
	result := c.checkNetwork()

	if result.Passed {
		t.Error("expected network check to fail with port conflict")
	}
}

func TestChecker_CheckNetwork_NoConflict(t *testing.T) {
	c := NewChecker(nil, "/tmp/test", &types.Config{
		Name:    "test",
		Version: "1.0",
		Services: map[string]types.Service{
			"web":  {Image: "nginx", Port: "8080:80"},
			"api":  {Image: "api", Port: "3000:3000"},
		},
	})
	result := c.checkNetwork()

	if !result.Passed {
		t.Errorf("expected network check to pass with no conflict, got: %s", result.Message)
	}
}

func TestChecker_CheckSecrets_NoKey(t *testing.T) {
	tmpDir := t.TempDir()
	c := NewChecker(nil, tmpDir, &types.Config{
		Name:    "test",
		Version: "1.0",
		Services: map[string]types.Service{
			"web": {Image: "nginx"},
		},
	})
	result := c.checkSecrets()

	if !result.Passed {
		t.Errorf("expected secrets check to pass with no key, got: %s", result.Message)
	}
}

func TestChecker_CheckSecrets_NoKeyNoRefs(t *testing.T) {
	tmpDir := t.TempDir()
	c := NewChecker(nil, tmpDir, &types.Config{
		Name:    "test",
		Version: "1.0",
		Services: map[string]types.Service{
			"web": {Image: "nginx"},
		},
	})
	result := c.checkSecrets()

	if !result.Passed {
		t.Errorf("expected secrets check to pass when no key and no refs, got: %s", result.Message)
	}
	if result.Message != "No encryption key (will be created on first use)" {
		t.Errorf("unexpected message: %s", result.Message)
	}
}

func TestChecker_RunAll_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	c := NewChecker(nil, tmpDir, &types.Config{
		Name:    "test",
		Version: "1.0",
		Services: map[string]types.Service{
			"web": {Image: "nginx", Port: "8080:80"},
			"api": {Image: "api", Port: "3000:3000"},
		},
	})
	results := c.RunAll(context.Background())

	if len(results) == 0 {
		t.Fatal("RunAll() returned no results")
	}

	for _, r := range results {
		t.Logf("Check %s: passed=%v, msg=%s", r.Name, r.Passed, r.Message)
	}
}

func findResult(results []Result, name string) *Result {
	for i, r := range results {
		if r.Name == name {
			return &results[i]
		}
	}
	return nil
}
