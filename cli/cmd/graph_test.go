package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGraphCmd_ShowsDependencies(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	configContent := `name: test-app
version: "1.0"
services:
  web:
    image: nginx:alpine
    port: "8080:80"
    depends_on: [api]
  api:
    image: node:20-alpine
    port: "3000:3000"
    depends_on: [db, redis]
  db:
    image: postgres:16
    port: "5432:5432"
  redis:
    image: redis:7-alpine
    port: "6379:6379"
`
	os.WriteFile(filepath.Join(tmpDir, "devbox.yml"), []byte(configContent), 0644)

	output := captureStdout(func() {
		runGraph(nil, nil)
	})

	if !strings.Contains(output, "db") {
		t.Errorf("expected output to contain 'db', got %q", output)
	}
	if !strings.Contains(output, "redis") {
		t.Errorf("expected output to contain 'redis', got %q", output)
	}
}

func TestGraphCmd_NoServices(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	configContent := `name: empty-app
version: "1.0"
services: {}
`
	os.WriteFile(filepath.Join(tmpDir, "devbox.yml"), []byte(configContent), 0644)

	output := captureStdout(func() {
		runGraph(nil, nil)
	})

	if !strings.Contains(output, "No services") {
		t.Errorf("expected 'No services' message, got %q", output)
	}
}

func TestGraphCmd_NoConfig(t *testing.T) {
	err := runGraph(nil, nil)
	if err == nil {
		t.Error("expected error when no devbox.yml, got nil")
	}
}
