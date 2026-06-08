package host

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/devboxos/devboxos/shared/runtime"
	"github.com/google/uuid"
)

type processInfo struct {
	Cmd       *exec.Cmd
	Name      string
	Service   string
	Status    string
	Port      string
	Labels    map[string]string
	StartedAt time.Time
	memReader io.ReadCloser
	memWriter io.WriteCloser
	done      chan struct{}
}

type logBuffer struct {
	buf   bytes.Buffer
	mu    sync.Mutex
	max   int
}

func (lb *logBuffer) Write(p []byte) (int, error) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	// Trim from front if buffer exceeds max
	if lb.buf.Len()+len(p) > lb.max {
		excess := lb.buf.Len() + len(p) - lb.max
		if excess < lb.buf.Len() {
			lb.buf.Next(excess)
		}
	}
	return lb.buf.Write(p)
}

func (lb *logBuffer) Read(p []byte) (int, error) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	return lb.buf.Read(p)
}

type logStream struct {
	rc      io.ReadCloser
	buf     *logBuffer
	mu      sync.Mutex
	readers map[chan []byte]struct{}
}

func (ls *logStream) Read(p []byte) (int, error) {
	return ls.buf.Read(p)
}

func (ls *logStream) Close() error {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	for ch := range ls.readers {
		close(ch)
	}
	ls.readers = nil
	return nil
}

type StreamInfo struct {
	ID        string
	Name      string
	Image     string
	Status    string
	Ports     []runtime.PortMapping
	Networks  []string
	StartedAt string
	Health    string
	Labels    map[string]string
}

func mapStatus(proc *processInfo) string {
	if proc == nil {
		return "unknown"
	}
	if proc.Cmd == nil || proc.Cmd.Process == nil {
		return "created"
	}
	if proc.Cmd.ProcessState != nil && proc.Cmd.ProcessState.Exited() {
		return "exited"
	}
	return "running"
}

// HostRuntime implements runtime.Runtime using host OS processes.
type HostRuntime struct {
	mu        sync.Mutex
	nextID    int
	processes map[string]*processInfo
	logs      map[string]*logStream
	volumeRoot string // Directory for volume storage
}

// NewHostRuntime creates a new host process runtime.
func NewHostRuntime() *HostRuntime {
	return &HostRuntime{
		processes: make(map[string]*processInfo),
		logs:      make(map[string]*logStream),
		volumeRoot: filepath.Join(os.TempDir(), "devbox-volumes"),
	}
}

// SetVolumeRoot sets the directory where volumes are stored.
func (h *HostRuntime) SetVolumeRoot(root string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.volumeRoot = root
}

func (h *HostRuntime) Connect(ctx context.Context) error {
	return nil
}

func (h *HostRuntime) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, proc := range h.processes {
		if proc.Cmd != nil && proc.Cmd.Process != nil {
			proc.Cmd.Process.Kill()
		}
		delete(h.processes, id)
	}
	return nil
}

func (h *HostRuntime) Check(ctx context.Context) error {
	return nil
}

func (h *HostRuntime) PullImage(ctx context.Context, image string) error {
	return fmt.Errorf("%w: PullImage requires Docker", runtime.ErrNotSupported)
}

func (h *HostRuntime) BuildImage(ctx context.Context, cfg runtime.BuildConfig, statusChan chan<- string) (string, error) {
	return "", fmt.Errorf("%w: BuildImage requires Docker", runtime.ErrNotSupported)
}

func (h *HostRuntime) CreateContainer(ctx context.Context, cfg runtime.ContainerConfig) (string, error) {
	// Host processes need a command
	if len(cfg.Command) == 0 {
		return "", fmt.Errorf("host runtime requires command for service %s", cfg.Name)
	}

	id := uuid.New().String()
	proc := &processInfo{
		Name:    cfg.Name,
		Service: strings.TrimPrefix(cfg.Name, "devbox-"),
		Status:  "created",
		Labels:  cfg.Labels,
		done:    make(chan struct{}),
	}
	proc.Port = ""
	for hp := range cfg.Ports {
		proc.Port = hp
		break
	}

	// Build exec command
	var cmd *exec.Cmd
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "cmd"
		if _, err := os.Stat("/bin/sh"); err == nil {
			shell = "/bin/sh"
		}
	}
	switch filepath.Base(shell) {
	case "cmd", "cmd.exe":
		cmd = exec.CommandContext(ctx, shell, "/C", cfg.Command[0])
	case "sh", "bash", "zsh":
		cmd = exec.CommandContext(ctx, shell, "-c", cfg.Command[0])
	default:
		cmd = exec.CommandContext(ctx, cfg.Command[0], cfg.Command[1:]...)
	}

	// Working directory
	if cfg.WorkingDir != "" {
		cmd.Dir = cfg.WorkingDir
	}

	// Environment variables
	cmd.Env = os.Environ()
	for k, v := range cfg.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Capture stdout/stderr
	lb := &logBuffer{max: 256 * 1024}
	stdoutPipe, _ := cmd.StdoutPipe()
	stderrPipe, _ := cmd.StderrPipe()

	// Combined log reader/writer
	ls := &logStream{
		buf:     lb,
		readers: make(map[chan []byte]struct{}),
	}

	h.mu.Lock()
	h.processes[id] = proc
	h.logs[id] = ls
	h.mu.Unlock()

	// Goroutine to merge stdout+stderr into log buffer
	go func() {
		var wg sync.WaitGroup
		wg.Add(2)
		copy := func(r io.Reader) {
			defer wg.Done()
			scanner := bufio.NewScanner(r)
			for scanner.Scan() {
				line := scanner.Bytes()
				lineCopy := make([]byte, len(line))
				copy(lineCopy, line)
				lb.Write(append(lineCopy, '\n'))
			}
		}
		if stdoutPipe != nil {
			go copy(stdoutPipe)
		}
		if stderrPipe != nil {
			go copy(stderrPipe)
		}
		wg.Wait()
	}()

	return id, nil
}

func (h *HostRuntime) StartContainer(ctx context.Context, id string) error {
	h.mu.Lock()
	proc := h.processes[id]
	h.mu.Unlock()
	if proc == nil {
		return fmt.Errorf("process %s not found", id)
	}
	if proc.Cmd == nil {
		return fmt.Errorf("process %s has no command (did you create it first?)", id)
	}
	if proc.Cmd.Process != nil {
		return nil
	}
	if err := proc.Cmd.Start(); err != nil {
		proc.Status = "failed"
		return fmt.Errorf("start process %s: %w", proc.Name, err)
	}
	proc.Status = "running"
	proc.StartedAt = time.Now()

	go func() {
		proc.Cmd.Wait()
		proc.Status = "exited"
		close(proc.done)
	}()

	return nil
}

func (h *HostRuntime) StopContainer(ctx context.Context, id string, timeoutSeconds int) error {
	h.mu.Lock()
	proc := h.processes[id]
	h.mu.Unlock()
	if proc == nil {
		return fmt.Errorf("process %s not found", id)
	}
	if proc.Cmd == nil || proc.Cmd.Process == nil {
		return nil
	}
	if proc.Cmd.ProcessState != nil && proc.Cmd.ProcessState.Exited() {
		return nil
	}

	pid := proc.Cmd.Process.Pid

	// SIGTERM first, then SIGKILL after timeout
	if timeoutSeconds > 0 {
		proc.Cmd.Process.Signal(os.Interrupt)
		select {
		case <-proc.done:
			killProcessTree(pid)
			return nil
		case <-time.After(time.Duration(timeoutSeconds) * time.Second):
		}
	}

	// Kill the main process
	proc.Cmd.Process.Kill()
	// Kill child processes (cmd /C forks children on Windows)
	killProcessTree(pid)
	return nil
}

func killProcessTree(pid int) {
	if goruntime.GOOS == "windows" {
		exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid)).Run()
	}
}

func (h *HostRuntime) RemoveContainer(ctx context.Context, id string, force bool) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	proc := h.processes[id]
	if proc == nil {
		return nil
	}
	if force && proc.Cmd != nil && proc.Cmd.Process != nil {
		proc.Cmd.Process.Kill()
	}
	delete(h.processes, id)
	if ls := h.logs[id]; ls != nil {
		ls.Close()
		delete(h.logs, id)
	}
	return nil
}

func (h *HostRuntime) GetContainerInfo(ctx context.Context, id string) (runtime.ContainerInfo, error) {
	h.mu.Lock()
	proc := h.processes[id]
	h.mu.Unlock()
	if proc == nil {
		return runtime.ContainerInfo{}, fmt.Errorf("process %s not found", id)
	}

	status := mapStatus(proc)
	var ports []runtime.PortMapping
	if proc.Port != "" {
		portNum, _ := strconv.Atoi(proc.Port)
		ports = append(ports, runtime.PortMapping{
			HostPort:      proc.Port,
			ContainerPort: strconv.Itoa(portNum),
			Protocol:      "tcp",
		})
	}

	return runtime.ContainerInfo{
		ID:     id,
		Name:   proc.Name,
		Status: status,
		Ports:  ports,
		Labels: proc.Labels,
		Health: "none",
	}, nil
}

func (h *HostRuntime) ListContainers(ctx context.Context, labels map[string]string) ([]runtime.ContainerInfo, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	var result []runtime.ContainerInfo
	for id, proc := range h.processes {
		if labels != nil {
			match := true
			for k, v := range labels {
				if proc.Labels[k] != v {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}

		status := mapStatus(proc)
		var ports []runtime.PortMapping
		if proc.Port != "" {
			portNum, _ := strconv.Atoi(proc.Port)
			ports = append(ports, runtime.PortMapping{
				HostPort:      proc.Port,
				ContainerPort: strconv.Itoa(portNum),
				Protocol:      "tcp",
			})
		}

		startedAt := ""
		if !proc.StartedAt.IsZero() {
			startedAt = proc.StartedAt.Format(time.RFC3339)
		}

		result = append(result, runtime.ContainerInfo{
			ID:        id,
			Name:      proc.Name,
			Status:    status,
			Ports:     ports,
			StartedAt: startedAt,
			Health:    "none",
			Labels:    proc.Labels,
		})
	}
	return result, nil
}

func (h *HostRuntime) StreamLogs(ctx context.Context, id string, opts runtime.LogOptions) (io.ReadCloser, error) {
	h.mu.Lock()
	ls := h.logs[id]
	h.mu.Unlock()
	if ls == nil {
		return nil, fmt.Errorf("no logs available for %s", id)
	}
	return ls, nil
}

func (h *HostRuntime) CreateNetwork(ctx context.Context, name string) error {
	return nil
}

func (h *HostRuntime) RemoveNetwork(ctx context.Context, name string) error {
	return nil
}

func (h *HostRuntime) NetworkExists(ctx context.Context, name string) (bool, error) {
	return false, nil
}

func (h *HostRuntime) CreateVolume(ctx context.Context, name string) error {
	h.mu.Lock()
	root := h.volumeRoot
	h.mu.Unlock()

	volPath := filepath.Join(root, sanitizeVolumeName(name))
	if err := os.MkdirAll(volPath, 0755); err != nil {
		return fmt.Errorf("create volume directory %s: %w", volPath, err)
	}
	return nil
}

func (h *HostRuntime) RemoveVolume(ctx context.Context, name string) error {
	h.mu.Lock()
	root := h.volumeRoot
	h.mu.Unlock()

	volPath := filepath.Join(root, sanitizeVolumeName(name))
	return os.RemoveAll(volPath)
}

func (h *HostRuntime) VolumeExists(ctx context.Context, name string) (bool, error) {
	h.mu.Lock()
	root := h.volumeRoot
	h.mu.Unlock()

	volPath := filepath.Join(root, sanitizeVolumeName(name))
	_, err := os.Stat(volPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (h *HostRuntime) VolumePath(ctx context.Context, name string) (string, error) {
	h.mu.Lock()
	root := h.volumeRoot
	h.mu.Unlock()

	volPath := filepath.Join(root, sanitizeVolumeName(name))
	_, err := os.Stat(volPath)
	if err != nil {
		return "", err
	}
	return volPath, nil
}

// sanitizeVolumeName replaces path separators with underscores to prevent
// directory traversal and ensure legal directory names.
func sanitizeVolumeName(name string) string {
	name = strings.ReplaceAll(name, "..", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, ":", "_")
	return name
}
