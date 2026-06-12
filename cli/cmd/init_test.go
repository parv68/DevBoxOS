package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitCmd_CreatesDevboxYml(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Create minimal project files so autodetect finds something
	os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"name":"test"}`), 0644)
	os.WriteFile(filepath.Join(tmpDir, "index.js"), []byte(`const app = require('express')(); app.listen(3000);`), 0644)

	err := runInit(initCmd, nil)
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
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

	os.WriteFile(filepath.Join(tmpDir, "devbox.yml"), []byte("name: existing\n"), 0644)

	err := runInit(initCmd, nil)
	// runInit returns nil (just warns) when file exists
	if err != nil {
		t.Fatalf("runInit should return nil when file exists, got: %v", err)
	}
	// Verify file wasn't overwritten
	data, _ := os.ReadFile(filepath.Join(tmpDir, "devbox.yml"))
	if string(data) != "name: existing\n" {
		t.Errorf("existing file was overwritten: got %q", string(data))
	}
}

func TestInitCmd_NamedProject(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Create minimal project files so autodetect finds something
	os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"name":"test"}`), 0644)
	os.WriteFile(filepath.Join(tmpDir, "index.js"), []byte(`const app = require('express')(); app.listen(3000);`), 0644)

	err := runInit(initCmd, []string{"my-project"})
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "devbox.yml")); os.IsNotExist(err) {
		t.Error("devbox.yml was not created")
	}
}
