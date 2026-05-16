package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParser_Parse(t *testing.T) {
	dir, err := os.MkdirTemp("", "devbox-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	content := []byte(`
name: myapp
version: "1.0"
services:
  web:
    image: nginx:latest
    port: "8080"
  db:
    image: postgres:16
    depends_on:
      - web
`)
	if err := os.WriteFile(filepath.Join(dir, "devbox.yml"), content, 0644); err != nil {
		t.Fatal(err)
	}

	parser := NewParser()
	cfg, err := parser.Parse(dir)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if cfg.Name != "myapp" {
		t.Errorf("Name = %q, want %q", cfg.Name, "myapp")
	}
	if len(cfg.Services) != 2 {
		t.Errorf("Services = %d, want 2", len(cfg.Services))
	}
	if svc, ok := cfg.Services["web"]; !ok {
		t.Error("Missing service: web")
	} else if svc.Image != "nginx:latest" {
		t.Errorf("web.Image = %q, want %q", svc.Image, "nginx:latest")
	}
	if svc, ok := cfg.Services["db"]; !ok {
		t.Error("Missing service: db")
	} else if len(svc.DependsOn) != 1 || svc.DependsOn[0] != "web" {
		t.Errorf("db.DependsOn = %v, want [web]", svc.DependsOn)
	}
}

func TestParser_Parse_NotFound(t *testing.T) {
	dir, err := os.MkdirTemp("", "devbox-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	parser := NewParser()
	_, err = parser.Parse(dir)
	if err == nil {
		t.Fatal("Expected error for missing devbox.yml")
	}
}

func TestParser_Parse_InvalidYAML(t *testing.T) {
	dir, err := os.MkdirTemp("", "devbox-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	content := []byte(`invalid: yaml: [unclosed`)
	if err := os.WriteFile(filepath.Join(dir, "devbox.yml"), content, 0644); err != nil {
		t.Fatal(err)
	}

	parser := NewParser()
	_, err = parser.Parse(dir)
	if err == nil {
		t.Fatal("Expected error for invalid YAML")
	}
}

func TestParser_ParseFile(t *testing.T) {
	dir, err := os.MkdirTemp("", "devbox-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "custom.yml")
	content := []byte(`name: customapp\n`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	parser := NewParser()
	_, err = parser.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
}

func TestParser_Generate(t *testing.T) {
	dir, err := os.MkdirTemp("", "devbox-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	parser := NewParser()
	if err := parser.Generate(dir, "testproj"); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filepath.Join(dir, "devbox.yml")); err != nil {
		t.Errorf("devbox.yml not created: %v", err)
	}

	// Second generate should fail
	if err := parser.Generate(dir, "testproj"); err == nil {
		t.Error("Expected error on second Generate")
	}
}
