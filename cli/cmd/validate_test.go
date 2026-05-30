package cmd

import (
	"os"
	"path/filepath"
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

	err := runValidate(nil, nil)
	if err != nil {
		t.Errorf("expected no error for valid config, got %v", err)
	}
}

func TestValidateCmd_InvalidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	configContent := `name: test-app
version: "1.0"
services:
  web:
    port: "8080:80"
`
	os.WriteFile(filepath.Join(tmpDir, "devbox.yml"), []byte(configContent), 0644)

	err := runValidate(nil, nil)
	if err == nil {
		t.Error("expected error for invalid config, got nil")
	}
}

func TestValidateCmd_NoFile(t *testing.T) {
	err := runValidate(nil, nil)
	if err == nil {
		t.Error("expected error when no devbox.yml, got nil")
	}
}
