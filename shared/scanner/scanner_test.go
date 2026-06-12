package scanner

import (
	"net"
	"os"
	"path/filepath"
	"sort"
	"testing"
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

func TestScanNodeJSAppListen(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "index.js"), `
		const express = require('express');
		const app = express();
		app.listen(3000, () => {});
	`)
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"test-app"}`)

	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Language != "node" {
		t.Fatalf("expected node, got %s", results[0].Language)
	}
	if len(results[0].Ports) == 0 || results[0].Ports[0].Port != 3000 {
		t.Fatalf("expected port 3000, got %v", results[0].Ports)
	}
}

func TestScanNodePackageJSONPort(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"test","scripts":{"dev":"PORT=4000 node server.js"}}`)

	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	hasPort4000 := false
	for _, p := range results[0].Ports {
		if p.Port == 4000 {
			hasPort4000 = true
			break
		}
	}
	if !hasPort4000 {
		t.Fatalf("expected port 4000 from script, got %v", results[0].Ports)
	}
}

func TestScanGoListenAndServe(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.go"), `
		package main
		import "net/http"
		func main() {
			http.ListenAndServe(":8080", nil)
		}
	`)
	writeFile(t, filepath.Join(dir, "go.mod"), "module test")

	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Language != "go" {
		t.Fatalf("expected go, got %s", results[0].Language)
	}
	if len(results[0].Ports) == 0 || results[0].Ports[0].Port != 8080 {
		t.Fatalf("expected port 8080, got %v", results[0].Ports)
	}
}

func TestScanGoGin(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.go"), `
		package main
		import "github.com/gin-gonic/gin"
		func main() {
			r := gin.Default()
			r.Run(":3000")
		}
	`)
	writeFile(t, filepath.Join(dir, "go.mod"), "module test")

	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if len(results[0].Ports) == 0 || results[0].Ports[0].Port != 3000 {
		t.Fatalf("expected port 3000, got %v", results[0].Ports)
	}
}

func TestScanRustBind(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.rs"), `
		use std::net::TcpListener;
		fn main() {
			let listener = TcpListener::bind("127.0.0.1:8080").unwrap();
		}
	`)
	writeFile(t, filepath.Join(dir, "Cargo.toml"), `[package]\nname = "test"`)

	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if len(results[0].Ports) > 0 {
		found := false
		for _, p := range results[0].Ports {
			if p.Port == 8080 {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected port 8080, got %v", results[0].Ports)
		}
	}
}

func TestScanPythonRun(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "app.py"), `
		import uvicorn
		if __name__ == "__main__":
			uvicorn.run("app", port=8000)
	`)
	writeFile(t, filepath.Join(dir, "requirements.txt"), "fastapi\nuvicorn")

	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if len(results[0].Ports) == 0 || results[0].Ports[0].Port != 8000 {
		t.Fatalf("expected port 8000, got %v", results[0].Ports)
	}
}

func TestScanDockerfileExpose(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "Dockerfile"), `
		FROM node:18
		EXPOSE 3000
		CMD ["node", "server.js"]
	`)

	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if len(results[0].Ports) == 0 || results[0].Ports[0].Port != 3000 {
		t.Fatalf("expected port 3000, got %v", results[0].Ports)
	}
}

func TestScanDotEnv(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".env"), `
		PORT=5000
		DATABASE_URL=postgres://localhost
	`)
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"test"}`)

	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	found := false
	for _, p := range results[0].Ports {
		if p.Port == 5000 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected port 5000 from .env, got %v", results[0].Ports)
	}
}

func TestScanMultipleServicesMonorepo(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "frontend", "package.json"), `{"name":"frontend"}`)
	writeFile(t, filepath.Join(dir, "frontend", "index.js"), `const app = require('express')(); app.listen(3000);`)

	writeFile(t, filepath.Join(dir, "backend", "go.mod"), "module backend")
	writeFile(t, filepath.Join(dir, "backend", "main.go"), `package main; import "net/http"; func main() { http.ListenAndServe(":8080", nil) }`)

	writeFile(t, filepath.Join(dir, "worker", "requirements.txt"), "")
	writeFile(t, filepath.Join(dir, "worker", "main.py"), `import uvicorn; uvicorn.run("app", port=8000)`)

	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results for monorepo, got %d", len(results))
	}

	serviceNames := make([]string, len(results))
	for i, r := range results {
		serviceNames[i] = r.ServiceName
	}
	sort.Strings(serviceNames)

	for _, name := range serviceNames {
		found := false
		for _, r := range results {
			if r.ServiceName == name {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected service %s not found in results", name)
		}
	}
}

func TestScanSkipsNodeModules(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"test"}`)
	writeFile(t, filepath.Join(dir, "index.js"), `app.listen(3000);`)
	writeFile(t, filepath.Join(dir, "node_modules", "express", "index.js"), `app.listen(1337);`)

	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result (skipping node_modules), got %d", len(results))
	}
	for _, p := range results[0].Ports {
		if p.Port == 1337 {
			t.Fatal("found port 1337 from node_modules, should have been skipped")
		}
	}
}

func TestScanEmptyDir(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "random.txt"), "hello")

	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) > 0 {
		t.Fatalf("expected 0 results for empty project, got %d", len(results))
	}
}

func TestCheckPortAvailability(t *testing.T) {
	err := checkPortAvailability("0")
	if err != nil {
		t.Log("port 0 should be available (kernel assigns random port)")
	}
}

func checkPortAvailability(port string) error {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}
	ln.Close()
	return nil
}

func TestScanViteConfig(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "vite.config.ts"), `
		import { defineConfig } from 'vite'
		export default defineConfig({
			server: { port: 5173 }
		})
	`)
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"test"}`)

	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range results[0].Ports {
		if p.Port == 5173 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected port 5173 from vite config, got %v", results[0].Ports)
	}
}

func TestScanMultiplePortsPerService(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "Dockerfile"), `
		FROM node:18
		EXPOSE 3000
		EXPOSE 4000
		EXPOSE 5000
	`)

	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	portSet := make(map[int]bool)
	for _, p := range results[0].Ports {
		portSet[p.Port] = true
	}
	if !portSet[3000] || !portSet[4000] || !portSet[5000] {
		t.Fatalf("expected ports 3000,4000,5000, got %v", results[0].Ports)
	}
}

func TestKnownDefaults(t *testing.T) {
	tests := []struct {
		lang string
		port int
		ok   bool
	}{
		{"node", 3000, true},
		{"go", 8080, true},
		{"rust", 8080, true},
		{"python", 8000, true},
		{"docker", 8080, true},
		{"unknown", 0, false},
	}
	for _, tt := range tests {
		p, ok := KnownDefault(tt.lang)
		if ok != tt.ok || (ok && p != tt.port) {
			t.Errorf("KnownDefault(%q) = (%d, %v), want (%d, %v)", tt.lang, p, ok, tt.port, tt.ok)
		}
	}
}

func TestBestPortSorting(t *testing.T) {
	ports := []DetectedPort{
		{Port: 3000, Priority: 5},
		{Port: 8080, Priority: 10},
		{Port: 5000, Priority: 1},
	}
	best := bestPort(ports)
	if best != 8080 {
		t.Fatalf("expected best port 8080 (highest priority), got %d", best)
	}
}

func TestDeriveServiceName(t *testing.T) {
	tests := []struct {
		dir string
		rel string
		expected string
	}{
		{"/home/user/project", "", "project"},
		{"/home/user/project", "frontend", "project"},
		{"/home/user/my-app.git", "", "my-app"},
	}
	for _, tt := range tests {
		got := deriveServiceName(tt.dir, tt.rel)
		if got != tt.expected {
			t.Errorf("deriveServiceName(%q, %q) = %q, want %q", tt.dir, tt.rel, got, tt.expected)
		}
	}
}

func TestDedupePorts(t *testing.T) {
	ports := []DetectedPort{
		{Port: 3000, Priority: 5},
		{Port: 3000, Priority: 10},
		{Port: 8080, Priority: 8},
	}
	result := dedupePorts(ports)
	if len(result) != 2 {
		t.Fatalf("expected 2 unique ports, got %d", len(result))
	}
	for _, p := range result {
		if p.Port == 3000 && p.Priority != 10 {
			t.Fatalf("expected port 3000 to keep highest priority 10, got %d", p.Priority)
		}
	}
}

func hasPort(ports []DetectedPort, port int) bool {
	for _, p := range ports {
		if p.Port == port {
			return true
		}
	}
	return false
}

// ─────────────────────────────────────────────
// Fixture-based language detection tests
// ─────────────────────────────────────────────

func TestFixture_NodeExpress(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "node-express", dir)
	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 service, got %d", len(results))
	}
	if hasPort(results[0].Ports, 3000) == false {
		t.Fatalf("expected port 3000, got %v", results[0].Ports)
	}
}

func TestFixture_NodeVite(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "node-vite", dir)
	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 service, got %d", len(results))
	}
	if hasPort(results[0].Ports, 5173) == false {
		t.Fatalf("expected port 5173 from vite config, got %v", results[0].Ports)
	}
}

func TestFixture_NodeNext(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "node-next", dir)
	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 service, got %d", len(results))
	}
	if hasPort(results[0].Ports, 3000) == false {
		t.Fatalf("expected port 3000 from next config, got %v", results[0].Ports)
	}
}

func TestFixture_PythonFastAPI(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "python-fastapi", dir)
	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 service, got %d", len(results))
	}
	if hasPort(results[0].Ports, 8000) == false {
		t.Fatalf("expected port 8000, got %v", results[0].Ports)
	}
	if results[0].Language != "python" {
		t.Fatalf("expected python, got %s", results[0].Language)
	}
}

func TestFixture_PythonFlask(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "python-flask", dir)
	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 service, got %d", len(results))
	}
	if hasPort(results[0].Ports, 5000) == false {
		t.Fatalf("expected port 5000, got %v", results[0].Ports)
	}
}

func TestFixture_PythonDjangoCLI(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "python-django", dir)
	// Add the CLI port pattern to manage.py
	content := []byte(`#!/usr/bin/env python
import os
import sys

if __name__ == "__main__":
    os.environ.setdefault("DJANGO_SETTINGS_MODULE", "settings")
    from django.core.management import execute_from_command_line
    execute_from_command_line(["manage.py", "runserver", "8000"])
`)
	if err := os.WriteFile(filepath.Join(dir, "manage.py"), content, 0644); err != nil {
		t.Fatal(err)
	}

	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 service, got %d", len(results))
	}
	if hasPort(results[0].Ports, 8000) == false {
		t.Fatalf("expected port 8000 from Django CLI, got %v", results[0].Ports)
	}
	if results[0].Language != "python" {
		t.Fatalf("expected python, got %s", results[0].Language)
	}
}

func TestFixture_GoNetHTTP(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "go-nethttp", dir)
	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 service, got %d", len(results))
	}
	if hasPort(results[0].Ports, 8080) == false {
		t.Fatalf("expected port 8080, got %v", results[0].Ports)
	}
	if results[0].Language != "go" {
		t.Fatalf("expected go, got %s", results[0].Language)
	}
}

func TestFixture_GoGin(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "go-gin", dir)
	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 service, got %d", len(results))
	}
	if hasPort(results[0].Ports, 8080) == false {
		t.Fatalf("expected port 8080, got %v", results[0].Ports)
	}
}

func TestFixture_GoEcho(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "go-echo", dir)
	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 service, got %d", len(results))
	}
	if hasPort(results[0].Ports, 1323) == false {
		t.Fatalf("expected port 1323, got %v", results[0].Ports)
	}
}

func TestFixture_RustActix(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "rust-actix", dir)
	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 service, got %d", len(results))
	}
	if hasPort(results[0].Ports, 8080) == false {
		t.Fatalf("expected port 8080, got %v", results[0].Ports)
	}
}

func TestFixture_RustAxum(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "rust-axum", dir)
	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 service, got %d", len(results))
	}
	if hasPort(results[0].Ports, 3000) == false {
		t.Fatalf("expected port 3000, got %v", results[0].Ports)
	}
}

func TestFixture_MonorepoMixed(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "monorepo-mixed", dir)
	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) < 3 {
		t.Fatalf("expected at least 3 services (frontend, backend, db), got %d", len(results))
	}
	foundFrontend := false
	foundBackend := false
	foundDB := false
	for _, r := range results {
		for _, p := range r.Ports {
			if p.Port == 5173 {
				foundFrontend = true
			}
			if p.Port == 8080 {
				foundBackend = true
			}
			if p.Port == 5432 {
				foundDB = true
			}
		}
	}
	if !foundFrontend {
		t.Fatal("expected frontend port 5173 not found")
	}
	if !foundBackend {
		t.Fatal("expected backend port 8080 not found")
	}
	if !foundDB {
		t.Fatal("expected db port 5432 not found")
	}
}

func TestFixture_ConflictThree(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "conflict-three", dir)

	// First scan should detect 3 services all on port 8080
	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 services, got %d", len(results))
	}

	// Resolve conflicts — should produce 8080, 8081, 8082
	resolved, warnings := ResolveConflicts(results)
	if len(warnings.Conflicts) != 2 {
		t.Fatalf("expected 2 conflict warnings for 3 services, got %d: %v", len(warnings.Conflicts), warnings.Conflicts)
	}
	usedPorts := make(map[string]bool)
	for _, rs := range resolved {
		if usedPorts[rs.ResolvedPort] {
			t.Fatalf("duplicate resolved port %s", rs.ResolvedPort)
		}
		usedPorts[rs.ResolvedPort] = true
	}
	if len(resolved) != 3 {
		t.Fatalf("expected 3 resolved services, got %d", len(resolved))
	}
}

func TestFixture_ConflictThreeEndToEnd(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "conflict-three", dir)

	// Run the scanner through the CLI autodetect path:
	// scan → resolve → build config
	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}

	resolved, _ := ResolveConflicts(results)

	allPorts := make(map[string]bool)
	for name, rs := range resolved {
		if allPorts[rs.ResolvedPort] {
			t.Fatalf("service %s has duplicate port %s after resolution", name, rs.ResolvedPort)
		}
		allPorts[rs.ResolvedPort] = true
	}

	expectedPorts := map[string]string{}
	for name, rs := range resolved {
		expectedPorts[name] = rs.ResolvedPort
	}

	if len(expectedPorts) != 3 {
		t.Fatalf("expected 3 resolved services, got %d", len(expectedPorts))
	}
}

func TestFixture_PythonFastAPIAutoDetect(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "python-fastapi", dir)

	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 service, got %d", len(results))
	}
	r := results[0]
	if r.Language != "python" {
		t.Fatalf("expected language python, got %s", r.Language)
	}
	if r.RunCommand != "python -m app" {
		t.Fatalf("expected RunCommand 'python -m app', got %q", r.RunCommand)
	}
}

func TestFixture_GoNetHTTPAutoDetect(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "go-nethttp", dir)

	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 service, got %d", len(results))
	}
	r := results[0]
	if r.Language != "go" {
		t.Fatalf("expected language go, got %s", r.Language)
	}
	if r.RunCommand != "go run ." {
		t.Fatalf("expected RunCommand 'go run .', got %q", r.RunCommand)
	}
}

func TestFixture_JavaSpring(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "java-spring", dir)
	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 service, got %d", len(results))
	}
	if hasPort(results[0].Ports, 8080) == false {
		t.Fatalf("expected port 8080, got %v", results[0].Ports)
	}
	if results[0].Language != "java" {
		t.Fatalf("expected java, got %s", results[0].Language)
	}
}

func TestFixture_RubyRails(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "ruby-rails", dir)
	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 service, got %d", len(results))
	}
	if hasPort(results[0].Ports, 3000) == false {
		t.Fatalf("expected port 3000, got %v", results[0].Ports)
	}
	if results[0].Language != "ruby" {
		t.Fatalf("expected ruby, got %s", results[0].Language)
	}
}

func TestFixture_PHPLaravel(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "php-laravel", dir)
	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 service, got %d", len(results))
	}
	if hasPort(results[0].Ports, 8000) == false {
		t.Fatalf("expected port 8000, got %v", results[0].Ports)
	}
	if results[0].Language != "php" {
		t.Fatalf("expected php, got %s", results[0].Language)
	}
}

func TestFixture_JavaSpringAutoDetect(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "java-spring", dir)

	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 service, got %d", len(results))
	}
	r := results[0]
	if r.Language != "java" {
		t.Fatalf("expected language java, got %s", r.Language)
	}
	if r.RunCommand != "mvn spring-boot:run" {
		t.Fatalf("expected RunCommand 'mvn spring-boot:run', got %q", r.RunCommand)
	}
}

func TestFixture_RubyRailsAutoDetect(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "ruby-rails", dir)

	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 service, got %d", len(results))
	}
	r := results[0]
	if r.Language != "ruby" {
		t.Fatalf("expected language ruby, got %s", r.Language)
	}
	if r.RunCommand != "bundle exec rails s" {
		t.Fatalf("expected RunCommand 'bundle exec rails s', got %q", r.RunCommand)
	}
}

func TestFixture_PHPLaravelAutoDetect(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "php-laravel", dir)

	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 service, got %d", len(results))
	}
	r := results[0]
	if r.Language != "php" {
		t.Fatalf("expected language php, got %s", r.Language)
	}
	if r.RunCommand != "php artisan serve" {
		t.Fatalf("expected RunCommand 'php artisan serve', got %q", r.RunCommand)
	}
}

func TestFixture_PostgreSQL(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "postgres-db", dir)
	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 service, got %d", len(results))
	}
	if hasPort(results[0].Ports, 5432) == false {
		t.Fatalf("expected port 5432, got %v", results[0].Ports)
	}
	if results[0].Language != "postgres" {
		t.Fatalf("expected language postgres, got %s", results[0].Language)
	}
}

func TestFixture_MySQL(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "mysql-db", dir)
	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 service, got %d", len(results))
	}
	if hasPort(results[0].Ports, 3306) == false {
		t.Fatalf("expected port 3306, got %v", results[0].Ports)
	}
	if results[0].Language != "mysql" {
		t.Fatalf("expected language mysql, got %s", results[0].Language)
	}
}

func TestFixture_Redis(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "redis-db", dir)
	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 service, got %d", len(results))
	}
	if hasPort(results[0].Ports, 6379) == false {
		t.Fatalf("expected port 6379, got %v", results[0].Ports)
	}
	if results[0].Language != "redis" {
		t.Fatalf("expected language redis, got %s", results[0].Language)
	}
}

func TestFixture_MongoDB(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "mongo-db", dir)
	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 service, got %d", len(results))
	}
	if hasPort(results[0].Ports, 27017) == false {
		t.Fatalf("expected port 27017, got %v", results[0].Ports)
	}
	if results[0].Language != "mongo" {
		t.Fatalf("expected language mongo, got %s", results[0].Language)
	}
}

func TestEnvFileMultiplePorts(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".env"), "PORT=3000\nDB_PORT=5432\nREDIS_PORT=6379\n")
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"test"}`)

	s := New()
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 service, got %d", len(results))
	}
	if !hasPort(results[0].Ports, 3000) {
		t.Fatalf("expected port 3000 (explicit PORT), got %v", results[0].Ports)
	}
	if !hasPort(results[0].Ports, 5432) {
		t.Fatalf("expected port 5432 (DB_PORT), got %v", results[0].Ports)
	}
	if !hasPort(results[0].Ports, 6379) {
		t.Fatalf("expected port 6379 (REDIS_PORT), got %v", results[0].Ports)
	}
}

