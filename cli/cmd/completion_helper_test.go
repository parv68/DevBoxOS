package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestCompleteServiceName_WithConfig(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	configContent := `name: test-app
version: "1.0"
services:
  web:
    image: nginx:alpine
  api:
    image: node:20-alpine
  db:
    image: postgres:16
`
	os.WriteFile(filepath.Join(tmpDir, "devbox.yml"), []byte(configContent), 0644)

	names, directive := completeServiceName(nil, nil, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected NoFileComp directive, got %v", directive)
	}

	if len(names) != 3 {
		t.Errorf("expected 3 service names, got %d: %v", len(names), names)
	}

	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}
	if !nameSet["web"] {
		t.Error("expected 'web' in completion list")
	}
	if !nameSet["api"] {
		t.Error("expected 'api' in completion list")
	}
	if !nameSet["db"] {
		t.Error("expected 'db' in completion list")
	}
}

func TestCompleteServiceName_NoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	names, directive := completeServiceName(nil, nil, "")

	if directive != cobra.ShellCompDirectiveError {
		t.Errorf("expected Error directive, got %v", directive)
	}
	if names != nil {
		t.Errorf("expected nil names, got %v", names)
	}
}

func TestCompleteServicePath_WithConfig(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	configContent := `name: test-app
version: "1.0"
services:
  web:
    image: nginx:alpine
  redis:
    image: redis:7-alpine
`
	os.WriteFile(filepath.Join(tmpDir, "devbox.yml"), []byte(configContent), 0644)

	names, directive := completeServicePath(nil, nil, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected NoFileComp directive, got %v", directive)
	}

	if len(names) != 2 {
		t.Errorf("expected 2 service paths, got %d: %v", len(names), names)
	}

	hasWeb := false
	hasRedis := false
	for _, n := range names {
		if n == "web:" {
			hasWeb = true
		}
		if n == "redis:" {
			hasRedis = true
		}
	}
	if !hasWeb {
		t.Error("expected 'web:' in completion list")
	}
	if !hasRedis {
		t.Error("expected 'redis:' in completion list")
	}
}
