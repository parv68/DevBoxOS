package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestComposeImportCmd_Success(t *testing.T) {
	composeOutput = "devbox.yaml"
	composeOverwrite = false
	defer func() {
		composeOutput = "devbox.yaml"
		composeOverwrite = false
	}()

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	composeContent := `version: "3.8"
services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
  api:
    image: node:20-alpine
    environment:
      - NODE_ENV=production
`
	os.WriteFile(filepath.Join(tmpDir, "docker-compose.yml"), []byte(composeContent), 0644)

	output := captureStdout(func() {
		err := runComposeImport(composeImportCmd, nil)
		if err != nil {
			t.Fatalf("runComposeImport failed: %v", err)
		}
	})

	if !strings.Contains(output, "Imported 2 services") {
		t.Errorf("expected import success message, got %q", output)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "devbox.yaml")); os.IsNotExist(err) {
		t.Error("devbox.yaml was not created")
	}
}

func TestComposeImportCmd_NoComposeFile(t *testing.T) {
	composeOutput = "devbox.yaml"
	composeOverwrite = false
	defer func() {
		composeOutput = "devbox.yaml"
		composeOverwrite = false
	}()

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	err := runComposeImport(composeImportCmd, nil)
	if err == nil {
		t.Error("expected error when no docker-compose.yml, got nil")
	}
}

func TestComposeImportCmd_SpecificFile(t *testing.T) {
	composeOutput = "devbox.yaml"
	composeOverwrite = false
	defer func() {
		composeOutput = "devbox.yaml"
		composeOverwrite = false
	}()

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	composeContent := `version: "3.8"
services:
  redis:
    image: redis:7-alpine
`
	os.WriteFile(filepath.Join(tmpDir, "custom-compose.yml"), []byte(composeContent), 0644)

	output := captureStdout(func() {
		err := runComposeImport(composeImportCmd, []string{"custom-compose.yml"})
		if err != nil {
			t.Fatalf("runComposeImport failed: %v", err)
		}
	})

	if !strings.Contains(output, "Imported") {
		t.Errorf("expected import success message, got %q", output)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "devbox.yaml")); os.IsNotExist(err) {
		t.Error("devbox.yaml was not created")
	}
}
