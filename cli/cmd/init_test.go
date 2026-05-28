package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCmd_CreatesDevboxYml(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	rootCmd.SetArgs([]string{"init", "test-project"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "devbox.yml")); os.IsNotExist(err) {
		t.Error("devbox.yml was not created")
	}
}

func TestInitCmd_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	os.WriteFile(filepath.Join(tmpDir, "devbox.yml"), []byte("name: existing"), 0644)

	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"init", "test-project"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "already exists") {
		t.Errorf("expected 'already exists' message, got %q", output)
	}
}
