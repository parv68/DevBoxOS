package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestComposeExportCmd_Success(t *testing.T) {
	composeExportOutput = "docker-compose.yml"
	composeExportForce = false
	composeVersion = "3.8"
	defer func() {
		composeExportOutput = "docker-compose.yml"
		composeExportForce = false
		composeVersion = "3.8"
	}()

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
    env:
      NODE_ENV: production
    depends_on: [db]
  db:
    image: postgres:16
    port: "5432:5432"
    env:
      POSTGRES_DB: app
      POSTGRES_PASSWORD: secret
`
	os.WriteFile(filepath.Join(tmpDir, "devbox.yml"), []byte(configContent), 0644)

	output := captureStdout(func() {
		err := runComposeExport(composeExportCmd, nil)
		if err != nil {
			t.Fatalf("runComposeExport failed: %v", err)
		}
	})

	if !strings.Contains(output, "Exported 3 services") {
		t.Errorf("expected export success message, got %q", output)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "docker-compose.yml")); os.IsNotExist(err) {
		t.Error("docker-compose.yml was not created")
	}
}

func TestComposeExportCmd_ExistingFile(t *testing.T) {
	composeExportOutput = "docker-compose.yml"
	composeExportForce = false
	composeVersion = "3.8"
	defer func() {
		composeExportOutput = "docker-compose.yml"
		composeExportForce = false
		composeVersion = "3.8"
	}()

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
	os.WriteFile(filepath.Join(tmpDir, "docker-compose.yml"), []byte("existing\n"), 0644)

	err := runComposeExport(composeExportCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "exists") {
		t.Errorf("expected 'exists' error, got %v", err)
	}
}

func TestComposeExportCmd_Force(t *testing.T) {
	composeExportOutput = "docker-compose.yml"
	composeExportForce = true
	composeVersion = "3.8"
	defer func() {
		composeExportOutput = "docker-compose.yml"
		composeExportForce = false
		composeVersion = "3.8"
	}()

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
	os.WriteFile(filepath.Join(tmpDir, "docker-compose.yml"), []byte("existing\n"), 0644)

	err := runComposeExport(composeExportCmd, nil)
	if err != nil {
		t.Fatalf("runComposeExport with --force failed: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(tmpDir, "docker-compose.yml"))
	if strings.Contains(string(data), "existing") {
		t.Error("existing file was not overwritten")
	}
}

func TestComposeExportCmd_CustomOutput(t *testing.T) {
	composeExportOutput = "docker-compose.yml"
	composeExportForce = false
	composeVersion = "3.8"
	defer func() {
		composeExportOutput = "docker-compose.yml"
		composeExportForce = false
		composeVersion = "3.8"
	}()

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

	err := runComposeExport(composeExportCmd, []string{"custom-compose.yml"})
	if err != nil {
		t.Fatalf("runComposeExport failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "custom-compose.yml")); os.IsNotExist(err) {
		t.Error("custom-compose.yml was not created")
	}
}

func TestComposeExportCmd_NoConfig(t *testing.T) {
	composeExportOutput = "docker-compose.yml"
	composeExportForce = false
	composeVersion = "3.8"
	defer func() {
		composeExportOutput = "docker-compose.yml"
		composeExportForce = false
		composeVersion = "3.8"
	}()

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	err := runComposeExport(composeExportCmd, nil)
	if err == nil {
		t.Error("expected error when no devbox.yml, got nil")
	}
}

func TestComposeExportCmd_OutputContent(t *testing.T) {
	composeExportOutput = "docker-compose.yml"
	composeExportForce = false
	composeVersion = "3.8"
	defer func() {
		composeExportOutput = "docker-compose.yml"
		composeExportForce = false
		composeVersion = "3.8"
	}()

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	configContent := `name: test-app
version: "1.0"
services:
  api:
    image: node:20-alpine
    port: "3000:3000"
    env:
      NODE_ENV: production
    depends_on: [db]
  db:
    image: postgres:16
    env:
      POSTGRES_DB: app
    healthcheck:
      path: /health
    volumes:
      - pgdata:/var/lib/postgresql/data
`
	os.WriteFile(filepath.Join(tmpDir, "devbox.yml"), []byte(configContent), 0644)

	err := runComposeExport(composeExportCmd, nil)
	if err != nil {
		t.Fatalf("runComposeExport failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "docker-compose.yml"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "node:20-alpine") {
		t.Error("output should contain image reference")
	}
	if !strings.Contains(content, "NODE_ENV=production") {
		t.Error("output should contain environment variable")
	}
	if !strings.Contains(content, "depends_on") {
		t.Error("output should contain depends_on")
	}
	if !strings.Contains(content, "healthcheck") {
		t.Error("output should contain healthcheck")
	}
	if !strings.Contains(content, "pgdata:") {
		t.Error("output should contain volumes")
	}
}

func TestComposeExportCmd_CustomVersion(t *testing.T) {
	composeExportOutput = "docker-compose.yml"
	composeExportForce = false
	composeVersion = "3.9"
	defer func() {
		composeExportOutput = "docker-compose.yml"
		composeExportForce = false
		composeVersion = "3.8"
	}()

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

	err := runComposeExport(composeExportCmd, nil)
	if err != nil {
		t.Fatalf("runComposeExport failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "docker-compose.yml"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, `version: "3.9"`) {
		t.Errorf("expected version 3.9 in output, got %q", content)
	}
}
