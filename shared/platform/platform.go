package platform

import (
	"os"
	"path/filepath"
	"runtime"
)

// OS represents the detected operating system.
type OS string

const (
	OSWindows OS = "windows"
	OSDarwin  OS = "darwin"
	OSLinux   OS = "linux"
)

// Detect returns the current operating system.
func Detect() OS {
	switch runtime.GOOS {
	case "windows":
		return OSWindows
	case "darwin":
		return OSDarwin
	case "linux":
		return OSLinux
	default:
		return OS(runtime.GOOS)
	}
}

// IsWindows returns true if running on Windows.
func IsWindows() bool {
	return Detect() == OSWindows
}

// IsUnix returns true if running on macOS or Linux.
func IsUnix() bool {
	os := Detect()
	return os == OSDarwin || os == OSLinux
}

// ConfigDir returns the platform-specific config directory.
// Windows: %APPDATA%\devboxos
// macOS/Linux: ~/.config/devboxos
func ConfigDir() string {
	if IsWindows() {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
		return filepath.Join(appData, "devboxos")
	}

	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "devboxos")
}

// DataDir returns the platform-specific data directory.
// Windows: %LOCALAPPDATA%\devboxos
// macOS/Linux: ~/.local/share/devboxos
func DataDir() string {
	if IsWindows() {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
		}
		return filepath.Join(localAppData, "devboxos")
	}

	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "devboxos")
}

// EngineSocketPath returns the path for the engine IPC socket.
// Windows: TCP (127.0.0.1:51000) - returned as empty string to indicate TCP
// macOS/Linux: Unix socket (~/.devbox/engine.sock)
func EngineSocketPath() string {
	if IsWindows() {
		return "" // Use TCP
	}

	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".devbox", "engine.sock")
}

// EngineAddress returns the engine connection address for gRPC dial.
// Windows: 127.0.0.1:51000 (TCP)
// macOS/Linux: unix:///home/user/.devbox/engine.sock
func EngineAddress() string {
	if IsWindows() {
		return "127.0.0.1:" + DefaultEnginePort()
	}

	return "unix://" + EngineSocketPath()
}

// DockerSocketPath returns the platform-specific Docker socket path.
func DockerSocketPath() string {
	if IsWindows() {
		return "npipe:////./pipe/docker_engine"
	}
	return "unix:///var/run/docker.sock"
}

// DefaultEnginePort returns the default TCP port for the engine.
func DefaultEnginePort() string {
	return "51000"
}

// PathSeparator returns the platform-specific path separator.
func PathSeparator() string {
	if IsWindows() {
		return "\\"
	}
	return "/"
}

// NormalizePath converts a path to the platform-specific format.
func NormalizePath(path string) string {
	if IsWindows() {
		return filepath.FromSlash(path)
	}
	return filepath.Clean(path)
}

// HomeDir returns the user's home directory.
func HomeDir() string {
	home, _ := os.UserHomeDir()
	return home
}

// DevBoxDir returns the .devbox directory path for a project.
func DevBoxDir(projectPath string) string {
	return filepath.Join(projectPath, ".devbox")
}
