package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateCmd_ValidConfig(t *testing.T) {
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
`
	os.WriteFile(filepath.Join(tmpDir, "devbox.yml"), []byte(configContent), 0644)

	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"validate"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "valid") {
		t.Errorf("expected valid message, got %q", output)
	}
}

func TestValidateCmd_InvalidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	os.WriteFile(filepath.Join(tmpDir, "devbox.yml"), []byte("invalid: yaml: ["), 0644)

	err := func() error {
		rootCmd.SetArgs([]string{"validate"})
		return rootCmd.Execute()
	}()

	if err == nil {
		t.Error("expected error for invalid config, got nil")
	}
}

func TestValidateCmd_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	err := func() error {
		rootCmd.SetArgs([]string{"validate"})
		return rootCmd.Execute()
	}()

	if err == nil {
		t.Error("expected error when no devbox.yml, got nil")
	}
}
