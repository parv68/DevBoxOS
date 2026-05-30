package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestURLCmd_ShowsPorts(t *testing.T) {
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
  api:
    image: node:20-alpine
    port: "3000:3000"
  db:
    image: postgres:16
    port: "5432:5432"
`
	os.WriteFile(filepath.Join(tmpDir, "devbox.yml"), []byte(configContent), 0644)

	output := captureStdout(func() {
		runURL(nil, nil)
	})

	if !strings.Contains(output, "http://localhost:8080") {
		t.Errorf("expected output to contain http://localhost:8080, got %q", output)
	}
}

func TestURLCmd_NoPorts(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	configContent := `name: test-app
version: "1.0"
services:
  worker:
    image: alpine:latest
`
	os.WriteFile(filepath.Join(tmpDir, "devbox.yml"), []byte(configContent), 0644)

	output := captureStdout(func() {
		runURL(nil, nil)
	})

	if !strings.Contains(output, "No services with port mappings") {
		t.Errorf("expected 'no ports' message, got %q", output)
	}
}

func TestURLCmd_NoConfig(t *testing.T) {
	err := runURL(nil, nil)
	if err == nil {
		t.Error("expected error when no devbox.yml, got nil")
	}
}

func TestURLCmd_CustomProtocol(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	configContent := `name: test-app
version: "1.0"
services:
  pgsql:
    image: postgres:16
    port: "5432:5432"
    protocol: postgres
`
	os.WriteFile(filepath.Join(tmpDir, "devbox.yml"), []byte(configContent), 0644)

	output := captureStdout(func() {
		runURL(nil, nil)
	})

	if !strings.Contains(output, "postgres://localhost:5432") {
		t.Errorf("expected output to contain postgres://localhost:5432, got %q", output)
	}
}
