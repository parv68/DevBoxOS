package platform

import (
	"strings"
	"testing"
)

func TestDetect(t *testing.T) {
	result := Detect()
	if result == "" {
		t.Error("Detect() returned empty")
	}
	switch result {
	case "windows", "darwin", "linux":
		// valid
	default:
		t.Errorf("Detect() = %q, expected windows/darwin/linux", result)
	}
}

func TestIsWindows(t *testing.T) {
	// Just verify it doesn't panic
	_ = IsWindows()
}

func TestConfigDir(t *testing.T) {
	dir := ConfigDir()
	if dir == "" {
		t.Error("ConfigDir() returned empty")
	}
}

func TestDataDir(t *testing.T) {
	dir := DataDir()
	if dir == "" {
		t.Error("DataDir() returned empty")
	}
}

func TestEngineSocketPath(t *testing.T) {
	path := EngineSocketPath()
	if IsWindows() {
		if path != "" {
			t.Errorf("EngineSocketPath() on Windows should be empty, got %q", path)
		}
	} else {
		if path == "" {
			t.Error("EngineSocketPath() on Unix should not be empty")
		}
		if !strings.HasSuffix(path, "engine.sock") {
			t.Errorf("EngineSocketPath() should end with engine.sock, got %q", path)
		}
	}
}

func TestEngineAddress(t *testing.T) {
	addr := EngineAddress()
	if addr == "" {
		t.Error("EngineAddress() returned empty")
	}
	if IsWindows() {
		if !strings.Contains(addr, "127.0.0.1") {
			t.Errorf("EngineAddress() on Windows should contain 127.0.0.1, got %q", addr)
		}
	} else {
		if !strings.HasPrefix(addr, "unix://") {
			t.Errorf("EngineAddress() on Unix should start with unix://, got %q", addr)
		}
	}
}

func TestDefaultEnginePort(t *testing.T) {
	port := DefaultEnginePort()
	if port != "51000" {
		t.Errorf("DefaultEnginePort() = %q, want \"51000\"", port)
	}
}

func TestDevBoxDir(t *testing.T) {
	dir := DevBoxDir("/tmp/testproject")
	if !strings.Contains(dir, ".devbox") {
		t.Errorf("DevBoxDir() should contain .devbox, got %q", dir)
	}
}

func TestDockerSocketPath(t *testing.T) {
	path := DockerSocketPath()
	if path == "" {
		t.Error("DockerSocketPath() returned empty")
	}
}

func TestNormalizePath(t *testing.T) {
	path := NormalizePath("/foo/bar")
	if path == "" {
		t.Error("NormalizePath() returned empty")
	}
}

func TestHomeDir(t *testing.T) {
	home := HomeDir()
	if home == "" {
		t.Error("HomeDir() returned empty")
	}
}
