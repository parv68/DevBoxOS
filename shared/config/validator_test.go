package config

import (
	"testing"

	"github.com/devboxos/devboxos/shared/types"
)

func TestValidator_New(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator() failed: %v", err)
	}
	if v == nil {
		t.Fatal("NewValidator() returned nil")
	}
}

func TestValidator_ValidConfig(t *testing.T) {
	v, _ := NewValidator()
	cfg := &types.Config{
		Name:    "test-app",
		Version: "1.0",
		Services: map[string]types.Service{
			"web": {
				Image: "nginx:alpine",
				Port:  "8080:80",
			},
		},
	}

	errs := v.Validate(cfg)
	if len(errs) > 0 {
		t.Errorf("expected no errors for valid config, got %v", errs)
	}
}

func TestValidator_ValidConfigWithBuild(t *testing.T) {
	v, _ := NewValidator()
	cfg := &types.Config{
		Name:    "test-app",
		Version: "1.0",
		Services: map[string]types.Service{
			"web": {
				Build: &types.BuildConfig{
					Context:    ".",
					Dockerfile: "Dockerfile",
				},
			},
		},
	}

	errs := v.Validate(cfg)
	if len(errs) > 0 {
		t.Errorf("expected no errors for config with build, got %v", errs)
	}
}

func TestValidator_ValidConfigWithRuntime(t *testing.T) {
	v, _ := NewValidator()
	cfg := &types.Config{
		Name:    "test-app",
		Version: "1.0",
		Services: map[string]types.Service{
			"worker": {
				Runtime: "node18",
				Command: "npm start",
			},
		},
	}

	errs := v.Validate(cfg)
	if len(errs) > 0 {
		t.Errorf("expected no errors for config with runtime, got %v", errs)
	}
}

func TestValidator_ValidMultiService(t *testing.T) {
	v, _ := NewValidator()
	cfg := &types.Config{
		Name:    "multi-service",
		Version: "2.0",
		Services: map[string]types.Service{
			"web": {
				Image: "nginx:alpine",
				Port:  "8080:80",
				DependsOn: []string{"api"},
			},
			"api": {
				Image: "node:20",
				Port:  "3000:3000",
				DependsOn: []string{"db"},
			},
			"db": {
				Image: "postgres:16",
				Port:  "5432:5432",
			},
		},
	}

	errs := v.Validate(cfg)
	if len(errs) > 0 {
		t.Errorf("expected no errors for valid multi-service config, got %v", errs)
	}
}

func TestValidator_MissingName(t *testing.T) {
	v, _ := NewValidator()
	cfg := &types.Config{
		Version: "1.0",
		Services: map[string]types.Service{
			"web": {Image: "nginx:alpine"},
		},
	}

	errs := v.Validate(cfg)
	if !containsError(errs, "name") {
		t.Errorf("expected error about missing 'name', got %v", errs)
	}
}

func TestValidator_MissingVersion(t *testing.T) {
	v, _ := NewValidator()
	cfg := &types.Config{
		Name: "test-app",
		Services: map[string]types.Service{
			"web": {Image: "nginx:alpine"},
		},
	}

	errs := v.Validate(cfg)
	if !containsError(errs, "version") {
		t.Errorf("expected error about missing 'version', got %v", errs)
	}
}

func TestValidator_NoServices(t *testing.T) {
	v, _ := NewValidator()
	cfg := &types.Config{
		Name:     "test-app",
		Version:  "1.0",
		Services: map[string]types.Service{},
	}

	errs := v.Validate(cfg)
	if !containsError(errs, "service") {
		t.Errorf("expected error about missing services, got %v", errs)
	}
}

func TestValidator_ServiceCommandOnlyIsValid(t *testing.T) {
	v, _ := NewValidator()
	cfg := &types.Config{
		Name:    "test-app",
		Version: "1.0",
		Services: map[string]types.Service{
			"web": {
				Command: "npm start",
			},
		},
	}

	errs := v.Validate(cfg)
	if containsError(errs, "image") {
		t.Errorf("command-only service should be valid, got errors: %v", errs)
	}
}

func TestValidator_ServiceEmptyDefinition(t *testing.T) {
	v, _ := NewValidator()
	cfg := &types.Config{
		Name:    "test-app",
		Version: "1.0",
		Services: map[string]types.Service{
			"web": {},
		},
	}

	errs := v.Validate(cfg)
	if !containsError(errs, "web") {
		t.Errorf("expected error about empty service 'web', got %v", errs)
	}
}

func TestValidator_EmptyServicesMap(t *testing.T) {
	v, _ := NewValidator()
	cfg := &types.Config{
		Name:     "test-app",
		Version:  "1.0",
		Services: nil,
	}

	errs := v.Validate(cfg)
	if !containsError(errs, "service") {
		t.Errorf("expected error about no services, got %v", errs)
	}
}

func TestValidator_MultipleErrors(t *testing.T) {
	v, _ := NewValidator()
	cfg := &types.Config{
		Name:    "",
		Version: "",
		Services: map[string]types.Service{
			"web": {},
		},
	}

	errs := v.Validate(cfg)
	if len(errs) < 3 {
		t.Errorf("expected at least 3 errors (name, version, service), got %d: %v", len(errs), errs)
	}
}

func TestValidator_ConfigWithAllFeatures(t *testing.T) {
	v, _ := NewValidator()
	cfg := &types.Config{
		Name:    "full-featured",
		Version: "1.0",
		Services: map[string]types.Service{
			"web": {
				Image: "nginx:alpine",
				Port:  "8080:80",
				Env:   map[string]string{"NODE_ENV": "production"},
				Healthcheck: &types.Healthcheck{
					Type:    "http",
					Path:    "/health",
					Retries: 3,
				},
				Resources: &types.Resources{
					Memory: "512m",
					CPU:    "0.5",
					Disk:   "1g",
				},
				DependsOn: []string{"db"},
			},
			"db": {
				Image: "postgres:16",
				Port:  "5432:5432",
				Security: &types.ServiceSecurity{
					TLS:  true,
					ReadOnly: true,
				},
			},
		},
	}

	errs := v.Validate(cfg)
	if len(errs) > 0 {
		t.Errorf("expected no errors for full-featured config, got %v", errs)
	}
}

func containsError(errs []error, substr string) bool {
	for _, e := range errs {
		if e != nil && contains(e.Error(), substr) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
