package telemetry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

var (
	version = "0.1.0-dev"

	mu          sync.Mutex
	enabled     = true
	machineID   string
	logFile     *os.File
	initialized bool
)

// Event represents a single telemetry event.
type Event struct {
	EventType string `json:"event_type"`
	Command   string `json:"command"`
	DurationMs int64  `json:"duration_ms,omitempty"`
	Success   bool   `json:"success,omitempty"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	Version   string `json:"version"`
	MachineID string `json:"machine_id"`
	Timestamp string `json:"timestamp"`
}

// Init initializes the telemetry system. Reads machine ID and prepares log file.
// If telemetry dir cannot be created, it silently disables itself.
func Init(devboxVersion string) {
	mu.Lock()
	defer mu.Unlock()

	if initialized {
		return
	}

	if devboxVersion != "" {
		version = devboxVersion
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		enabled = false
		return
	}

	devboxDir := filepath.Join(homeDir, ".devbox")
	if err := os.MkdirAll(devboxDir, 0755); err != nil {
		enabled = false
		return
	}

	// Read or generate machine ID
	machineID = loadOrGenerateMachineID(devboxDir)

	// Open telemetry log file (append mode)
	logPath := filepath.Join(devboxDir, "telemetry.jsonl")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		enabled = false
		return
	}
	logFile = f
	initialized = true
}

// Disable turns off telemetry for this session.
func Disable() {
	mu.Lock()
	defer mu.Unlock()
	enabled = false
}

// SetEnabled sets the telemetry enabled state.
func SetEnabled(e bool) {
	mu.Lock()
	defer mu.Unlock()
	enabled = e
}

// IsEnabled returns whether telemetry is enabled.
func IsEnabled() bool {
	mu.Lock()
	defer mu.Unlock()
	return enabled && initialized
}

// Record records a telemetry event. Safe for concurrent use.
func Record(eventType, command string, durationMs int64, success bool) {
	mu.Lock()
	localEnabled := enabled
	localFile := logFile
	localID := machineID
	localVersion := version
	mu.Unlock()

	if !localEnabled || localFile == nil {
		return
	}

	evt := Event{
		EventType:  eventType,
		Command:    command,
		DurationMs: durationMs,
		Success:    success,
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		Version:    localVersion,
		MachineID:  localID,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(evt)
	if err != nil {
		return
	}

	mu.Lock()
	if localFile != logFile {
		mu.Unlock()
		return
	}
	_, _ = fmt.Fprintln(localFile, string(data))
	mu.Unlock()
}

// Close flushes and closes the telemetry log file.
func Close() {
	mu.Lock()
	defer mu.Unlock()
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
	initialized = false
}

// loadOrGenerateMachineID reads an existing machine ID or creates a new one.
func loadOrGenerateMachineID(devboxDir string) string {
	idPath := filepath.Join(devboxDir, "machine-id")
	data, err := os.ReadFile(idPath)
	if err == nil && len(data) > 0 {
		return string(data)
	}

	id := fmt.Sprintf("%x", time.Now().UnixNano())
	_ = os.WriteFile(idPath, []byte(id), 0644)
	return id
}
