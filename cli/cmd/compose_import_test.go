package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestComposeImportCmd_Success(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	composeContent := `version: "3"
services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
`
	os.WriteFile(filepath.Join(tmpDir, "docker-compose.yml"), []byte(composeContent), 0644)

	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"init", "compose-import"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "Imported") {
		t.Errorf("expected import success message, got %q", output)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "devbox.yaml")); os.IsNotExist(err) {
		t.Error("devbox.yaml was not created")
	}
}

func TestComposeImportCmd_NoComposeFile(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	err := func() error {
		rootCmd.SetArgs([]string{"init", "compose-import"})
		return rootCmd.Execute()
	}()

	if err == nil {
		t.Error("expected error when no docker-compose.yml, got nil")
	}
}

func TestComposeImportCmd_SpecificFile(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	composeContent := `services:
  web:
    image: nginx:alpine
`
	customPath := filepath.Join(tmpDir, "custom-compose.yml")
	os.WriteFile(customPath, []byte(composeContent), 0644)

	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"init", "compose-import", customPath})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "Imported") {
		t.Errorf("expected import success message, got %q", output)
	}
}
