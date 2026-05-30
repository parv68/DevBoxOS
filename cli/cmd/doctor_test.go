package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorCmd_NoEngineWithConfig(t *testing.T) {
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
		runDoctor(nil, nil)
	})

	if !strings.Contains(output, "Docker") && !strings.Contains(output, "Diagnostics") {
		t.Errorf("expected output to mention Docker or Diagnostics, got %q", output)
	}
}

func TestDoctorCmd_NoEngineNoConfig(t *testing.T) {
	output := captureStdout(func() {
		runDoctor(nil, nil)
	})

	if !strings.Contains(output, "Docker") && !strings.Contains(output, "Diagnostics") {
		t.Errorf("expected output to mention Docker or Diagnostics, got %q", output)
	}
}
