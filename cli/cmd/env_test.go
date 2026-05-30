package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnvCmd_ShowsEnvVars(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	configContent := `name: test-app
version: "1.0"
services:
  web:
    image: nginx:alpine
    env:
      NGINX_HOST: localhost
      NGINX_PORT: "80"
`
	os.WriteFile(filepath.Join(tmpDir, "devbox.yml"), []byte(configContent), 0644)

	envReveal = true
	defer func() { envReveal = false }()

	output := captureStdout(func() {
		runEnv(nil, nil)
	})

	if !strings.Contains(output, "NGINX_HOST=localhost") {
		t.Errorf("expected output to contain NGINX_HOST=localhost, got %q", output)
	}
}

func TestEnvCmd_SpecificService(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	configContent := `name: test-app
version: "1.0"
services:
  web:
    image: nginx:alpine
    env:
      NGINX_HOST: localhost
  redis:
    image: redis:7-alpine
    env:
      REDIS_PORT: "6379"
`
	os.WriteFile(filepath.Join(tmpDir, "devbox.yml"), []byte(configContent), 0644)

	envReveal = true
	defer func() { envReveal = false }()

	output := captureStdout(func() {
		runEnv(nil, []string{"redis"})
	})

	if strings.Contains(output, "NGINX_HOST") {
		t.Errorf("expected only redis vars, got nginx in %q", output)
	}
	if !strings.Contains(output, "REDIS_PORT=6379") {
		t.Errorf("expected REDIS_PORT=6379, got %q", output)
	}
}

func TestEnvCmd_UnknownService(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	configContent := `name: test-app
version: "1.0"
services:
  web:
    image: nginx:alpine
`
	os.WriteFile(filepath.Join(tmpDir, "devbox.yml"), []byte(configContent), 0644)

	err := runEnv(nil, []string{"nonexistent"})
	if err == nil {
		t.Error("expected error for unknown service, got nil")
	}
}

func TestEnvCmd_NoEnv(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	configContent := `name: test-app
version: "1.0"
services:
  web:
    image: nginx:alpine
`
	os.WriteFile(filepath.Join(tmpDir, "devbox.yml"), []byte(configContent), 0644)

	output := captureStdout(func() {
		runEnv(nil, nil)
	})

	if !strings.Contains(output, "no environment variables") {
		t.Errorf("expected 'no environment variables' message, got %q", output)
	}
}

func TestEnvCmd_NoConfig(t *testing.T) {
	err := runEnv(nil, nil)
	if err == nil {
		t.Error("expected error when no devbox.yml, got nil")
	}
}

func TestMaskValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"abc", "****"},
		{"abcd", "****"},
		{"abcdef", "ab****ef"},
		{"abcdefgh", "ab****gh"},
		{"a", "****"},
		{"", "****"},
	}
	for _, tt := range tests {
		result := maskValue(tt.input)
		if result != tt.expected {
			t.Errorf("maskValue(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
