package autodetect

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/devboxos/devboxos/shared/types"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestAutoDetectNodeJS(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"test-app","scripts":{"dev":"node server.js"}}`)
	writeFile(t, filepath.Join(dir, "index.js"), `const app = require('express')(); app.listen(3000);`)

	cfg, err := AutoDetect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Services) == 0 {
		t.Fatal("expected at least 1 service")
	}
	for name, svc := range cfg.Services {
		if svc.Port == "" {
			t.Fatalf("service %s has no port", name)
		}
		if svc.Command == "" {
			t.Fatalf("service %s has no command", name)
		}
	}
}

func TestAutoDetectGo(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module test")
	writeFile(t, filepath.Join(dir, "main.go"), `package main; import "net/http"; func main() { http.ListenAndServe(":8080", nil) }`)

	cfg, err := AutoDetect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Services) == 0 {
		t.Fatal("expected at least 1 service")
	}
	for _, svc := range cfg.Services {
		if svc.Port == "" {
			t.Fatal("expected service to have a port")
		}
	}
}

func TestAutoDetectEmptyDir(t *testing.T) {
	dir := t.TempDir()
	_, err := AutoDetect(dir)
	if err == nil {
		t.Fatal("expected error for empty directory with no services detected")
	}
}

func TestAutoDetectMonorepo(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "frontend", "package.json"), `{"name":"frontend"}`)
	writeFile(t, filepath.Join(dir, "frontend", "index.js"), `const app = require('express')(); app.listen(3000);`)

	writeFile(t, filepath.Join(dir, "backend", "go.mod"), "module backend")
	writeFile(t, filepath.Join(dir, "backend", "main.go"), `package main; import "net/http"; func main() { http.ListenAndServe(":8080", nil) }`)

	cfg, err := AutoDetect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Services) < 2 {
		t.Fatalf("expected at least 2 services for monorepo, got %d", len(cfg.Services))
	}
	if cfg.Networking.Discovery != true {
		t.Fatal("expected discovery to be enabled")
	}
	if cfg.Networking.Egress != "default-deny" {
		t.Fatal("expected egress to be default-deny")
	}
}

func TestAutoDetectPortConflictResolution(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "api", "go.mod"), "module api")
	writeFile(t, filepath.Join(dir, "api", "main.go"), `package main; import "net/http"; func main() { http.ListenAndServe(":8080", nil) }`)

	writeFile(t, filepath.Join(dir, "admin", "go.mod"), "module admin")
	writeFile(t, filepath.Join(dir, "admin", "main.go"), `package main; import "net/http"; func main() { http.ListenAndServe(":8080", nil) }`)

	cfg, err := AutoDetect(dir)
	if err != nil {
		t.Fatal(err)
	}

	ports := make(map[string]bool)
	for name, svc := range cfg.Services {
		if svc.Port != "" {
			if ports[svc.Port] {
				t.Fatalf("duplicate port %s after resolution for service %s", svc.Port, name)
			}
			ports[svc.Port] = true
		}
	}
}

func TestAutoDetectNetworkingExpose(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"test"}`)
	writeFile(t, filepath.Join(dir, "index.js"), `const app = require('express')(); app.listen(3000);`)

	cfg, err := AutoDetect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Networking.Expose) == 0 {
		t.Fatal("expected exposed ports to be populated")
	}
	found3000 := false
	for _, p := range cfg.Networking.Expose {
		if p == 3000 {
			found3000 = true
			break
		}
	}
	if !found3000 {
		t.Fatalf("expected port 3000 in exposed ports, got %v", cfg.Networking.Expose)
	}
}

func TestAutoDetectDotEnv(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".env"), `PORT=4000`)
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"test"}`)

	cfg, err := AutoDetect(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, svc := range cfg.Services {
		if svc.Port == "4000" {
			return
		}
	}
	t.Fatalf("expected a service with port 4000 from .env, got services: %v", cfg.Services)
}

// Verify AutoDetect returns types.Config matching the expected structure
func TestAutoDetectConfigStructure(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"test"}`)
	writeFile(t, filepath.Join(dir, "index.js"), `const app = require('express')(); app.listen(3000);`)

	cfg, err := AutoDetect(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Name is set by the caller (init.go), not by AutoDetect itself
	if cfg.Version != "1.0" {
		t.Fatalf("expected version 1.0, got %s", cfg.Version)
	}

	var svc *types.Service
	for _, s := range cfg.Services {
		svc = &s
		break
	}
	if svc == nil {
		t.Fatal("expected at least one service")
	}
	if svc.Port == "" {
		t.Fatal("expected service to have a port")
	}
}

func TestAutoDetectJavaSpring(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "pom.xml"), `<project><modelVersion>4.0.0</modelVersion>
<parent><groupId>org.springframework.boot</groupId><artifactId>spring-boot-starter-parent</artifactId><version>3.2.0</version></parent>
<artifactId>demo</artifactId></project>`)
	writeFile(t, filepath.Join(dir, "src", "main", "resources", "application.properties"), "server.port=8080\nspring.application.name=demo\n")

	cfg, err := AutoDetect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Services) == 0 {
		t.Fatal("expected at least 1 service")
	}
	for name, svc := range cfg.Services {
		if svc.Port == "" {
			t.Fatalf("service %s has no port", name)
		}
		if svc.Command == "" {
			t.Fatalf("service %s has no command", name)
		}
	}
	if _, ok := cfg.Runtimes["java"]; !ok {
		t.Fatal("expected java runtime in config")
	}
}

func TestAutoDetectRubyRails(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "Gemfile"), `source "https://rubygems.org"\ngem "rails"\ngem "puma"\n`)
	writeFile(t, filepath.Join(dir, "config", "puma.rb"), "port ENV.fetch(\"PORT\") { 3000 }\n")

	cfg, err := AutoDetect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Services) == 0 {
		t.Fatal("expected at least 1 service")
	}
	for name, svc := range cfg.Services {
		if svc.Port == "" {
			t.Fatalf("service %s has no port", name)
		}
	}
	if _, ok := cfg.Runtimes["ruby"]; !ok {
		t.Fatal("expected ruby runtime in config")
	}
}

func TestAutoDetectPHPLaravel(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "composer.json"), `{"name":"laravel/laravel","scripts":{"dev":"php artisan serve --port=8000"}}`)
	writeFile(t, filepath.Join(dir, ".env"), "APP_PORT=8000\n")

	cfg, err := AutoDetect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Services) == 0 {
		t.Fatal("expected at least 1 service")
	}
	for name, svc := range cfg.Services {
		if svc.Port == "" {
			t.Fatalf("service %s has no port", name)
		}
	}
	if _, ok := cfg.Runtimes["php"]; !ok {
		t.Fatal("expected php runtime in config")
	}
}
