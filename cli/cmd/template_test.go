package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitTemplate_InvalidTemplate(t *testing.T) {
	initTemplate = "nonexistent"
	defer func() { initTemplate = "" }()

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	err := runInit(initCmd, nil)
	if err == nil {
		t.Error("expected error for invalid template, got nil")
	}
}

func TestInitTemplate_GoAPI(t *testing.T) {
	initTemplate = "go-api"
	defer func() { initTemplate = "" }()

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	err := runInit(initCmd, nil)
	if err != nil {
		t.Fatalf("runInit with go-api template failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "devbox.yml")); os.IsNotExist(err) {
		t.Error("devbox.yml was not created")
	}
	data, err := os.ReadFile(filepath.Join(tmpDir, "devbox.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "go-api") {
		t.Errorf("devbox.yml should contain 'go-api', got %s", string(data))
	}
}
