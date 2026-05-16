package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/devboxos/devboxos/engine/internal/state"
	pb "github.com/devboxos/devboxos/engine/proto"
	"github.com/devboxos/devboxos/shared/config"
	"github.com/devboxos/devboxos/shared/platform"
	"github.com/devboxos/devboxos/shared/runtime"
	"github.com/devboxos/devboxos/shared/runtime/docker"
	"github.com/devboxos/devboxos/shared/snapshot"
	"github.com/devboxos/devboxos/engine/internal/orchestrator"
	"google.golang.org/grpc"
)

var version = "0.1.0-dev"

type server struct {
	pb.UnimplementedEngineServiceServer
	startedAt    time.Time
	stateMgr     *state.Manager
	orchestrator *orchestrator.Orchestrator
	rt           runtime.Runtime
	mu           sync.Mutex
}

func (s *server) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{
		Version: version,
		Uptime:  int64(time.Since(s.startedAt).Seconds()),
	}, nil
}

func (s *server) Start(req *pb.StartRequest, stream pb.EngineService_StartServer) error {
	send := func(status, msg string, done bool) {
		stream.Send(&pb.StreamResponse{
			Status:  status,
			Message: msg,
			Done:    done,
		})
	}

	// Parse config
	parser := config.NewParser()
	cfg, err := parser.Parse(req.ProjectPath)
	if err != nil {
		send("error", fmt.Sprintf("Failed to parse config: %v", err), true)
		return err
	}

	// Validate config
	validator, err := config.NewValidator()
	if err != nil {
		send("warning", fmt.Sprintf("Schema validation skipped: %v", err), false)
	} else {
		if errs := validator.Validate(cfg); len(errs) > 0 {
			for _, e := range errs {
				send("error", e.Error(), false)
			}
			send("error", "Configuration validation failed", true)
			return fmt.Errorf("validation failed")
		}
	}

	send("info", fmt.Sprintf("Loaded %s (%d services)", cfg.Name, len(cfg.Services)), false)

	// Create Docker runtime (persist for log collection)
	s.mu.Lock()
	if s.rt == nil {
		s.mu.Unlock()
		rt := docker.NewDockerRuntime()
		if err := rt.Connect(stream.Context()); err != nil {
			send("error", fmt.Sprintf("Docker not available: %v", err), true)
			return err
		}
		s.mu.Lock()
		s.rt = rt
	}
	s.mu.Unlock()

	send("info", "Connected to Docker daemon", false)

	// Create orchestrator
	orch, err := orchestrator.NewOrchestrator(s.rt, req.ProjectPath, cfg)
	if err != nil {
		send("error", fmt.Sprintf("Failed to create orchestrator: %v", err), true)
		return err
	}

	// Store orchestrator for log collection and status tracking
	s.mu.Lock()
	s.orchestrator = orch
	s.mu.Unlock()

	// Status channel for streaming updates
	statusChan := make(chan string, 64)

	// Start in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- orch.Start(stream.Context(), cfg, req.ProjectPath, statusChan)
	}()

	// Stream status updates
	for {
		select {
		case msg := <-statusChan:
			done := msg == "All services started"
			send("info", msg, done)
			if done {
				return nil
			}
		case err := <-errChan:
			if err != nil {
				send("error", err.Error(), true)
				return err
			}
			return nil
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

func (s *server) Stop(ctx context.Context, req *pb.StopRequest) (*pb.StatusResponse, error) {
	// Parse config
	parser := config.NewParser()
	cfg, err := parser.Parse(req.ProjectPath)
	if err != nil {
		return &pb.StatusResponse{
			Status: "error",
			Error:  fmt.Sprintf("Failed to parse config: %v", err),
		}, nil
	}

	s.mu.Lock()
	rt := s.rt
	orch := s.orchestrator
	s.mu.Unlock()

	if rt == nil {
		return &pb.StatusResponse{
			Status: "error",
			Error:  "Docker not available: engine not started",
		}, nil
	}

	if orch == nil {
		return &pb.StatusResponse{
			Status: "error",
			Error:  "No active environment. Run 'devbox start' first.",
		}, nil
	}

	statusChan := make(chan string, 64)
	errChan := make(chan error, 1)
	gracePeriod := 30

	go func() {
		errChan <- orch.Stop(ctx, cfg, gracePeriod, statusChan)
	}()

	// Wait for completion
	select {
	case err := <-errChan:
		if err != nil {
			return &pb.StatusResponse{
				Status: "error",
				Error:  err.Error(),
			}, nil
		}
		return &pb.StatusResponse{
			Status: "stopped",
		}, nil
	case <-ctx.Done():
		return &pb.StatusResponse{
			Status: "error",
			Error:  "context cancelled",
		}, ctx.Err()
	}
}

func (s *server) Status(ctx context.Context, req *pb.StatusRequest) (*pb.StatusResponse, error) {
	// Parse config
	parser := config.NewParser()
	cfg, err := parser.Parse(req.ProjectPath)
	if err != nil {
		return &pb.StatusResponse{
			Status: "error",
			Error:  fmt.Sprintf("No devbox.yml found: %v", err),
		}, nil
	}

	if s.rt == nil {
		return &pb.StatusResponse{
			Status: "error",
			Error:  "Docker not available: engine not started",
		}, nil
	}

	// Use stored orchestrator
	s.mu.Lock()
	orch := s.orchestrator
	s.mu.Unlock()

	if orch == nil {
		return &pb.StatusResponse{
			Status: "stopped",
		}, nil
	}

	envStatus, err := orch.Status(ctx, cfg)
	if err != nil {
		return &pb.StatusResponse{
			Status: "error",
			Error:  err.Error(),
		}, nil
	}

	var services []*pb.ServiceStatus
	for _, svc := range envStatus.Services {
		services = append(services, &pb.ServiceStatus{
			Name:         svc.Name,
			Status:       svc.Status,
			Health:       svc.Health,
			Port:         int32(svc.Port),
			ContainerId:  svc.ContainerID,
			RestartCount: int32(svc.RestartCount),
		})
	}

	return &pb.StatusResponse{
		Status:   envStatus.Status,
		Services: services,
	}, nil
}

func (s *server) Logs(req *pb.LogsRequest, stream pb.EngineService_LogsServer) error {
	// Create Docker runtime
	rt := docker.NewDockerRuntime()
	if err := rt.Connect(stream.Context()); err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}
	defer rt.Close()

	// Find container
	containers, err := rt.ListContainers(stream.Context(), map[string]string{
		"devboxos.service": req.Service,
	})
	if err != nil {
		return fmt.Errorf("list containers: %w", err)
	}

	if len(containers) == 0 {
		return fmt.Errorf("no running container found for service: %s", req.Service)
	}

	containerID := containers[0].ID

	// Stream logs
	reader, err := rt.StreamLogs(stream.Context(), containerID, runtime.LogOptions{
		Follow: req.Follow,
		Tail:   int(req.Tail),
		Since:  req.Since,
	})
	if err != nil {
		return fmt.Errorf("stream logs: %w", err)
	}
	defer reader.Close()

	// Read and stream line by line
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if err := stream.Send(&pb.LogEntry{
			Service:   req.Service,
			Timestamp: time.Now().Format(time.RFC3339),
			Level:     "info",
			Message:   line,
		}); err != nil {
			return err
		}
	}

	return scanner.Err()
}

func (s *server) SnapshotSave(req *pb.SnapshotSaveRequest, stream pb.EngineService_SnapshotSaveServer) error {
	parser := config.NewParser()
	cfg, err := parser.Parse(req.ProjectPath)
	if err != nil {
		stream.Send(&pb.StreamResponse{Status: "error", Error: err.Error(), Done: true})
		return nil
	}

	rt := docker.NewDockerRuntime()
	if err := rt.Connect(stream.Context()); err != nil {
		stream.Send(&pb.StreamResponse{Status: "error", Error: "Docker not available", Done: true})
		return nil
	}
	defer rt.Close()

	statusChan := make(chan string, 64)
	done := make(chan error, 1)

	go func() {
		mgr := snapshot.NewManager(rt, req.ProjectPath)
		_, err := mgr.Save(stream.Context(), cfg, req.Name, req.IncludeLogs, statusChan)
		done <- err
	}()

	for {
		select {
		case msg := <-statusChan:
			stream.Send(&pb.StreamResponse{Status: "info", Message: msg})
		case err := <-done:
			if err != nil {
				stream.Send(&pb.StreamResponse{Status: "error", Error: err.Error(), Done: true})
				return nil
			}
			stream.Send(&pb.StreamResponse{Status: "done", Done: true})
			return nil
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

func (s *server) SnapshotLoad(req *pb.SnapshotLoadRequest, stream pb.EngineService_SnapshotLoadServer) error {
	rt := docker.NewDockerRuntime()
	if err := rt.Connect(stream.Context()); err != nil {
		stream.Send(&pb.StreamResponse{Status: "error", Error: "Docker not available", Done: true})
		return nil
	}
	defer rt.Close()

	statusChan := make(chan string, 64)
	done := make(chan error, 1)

	go func() {
		mgr := snapshot.NewManager(rt, req.ProjectPath)
		err := mgr.Load(stream.Context(), req.SnapshotId, req.Force, statusChan)
		done <- err
	}()

	for {
		select {
		case msg := <-statusChan:
			stream.Send(&pb.StreamResponse{Status: "info", Message: msg})
		case err := <-done:
			if err != nil {
				stream.Send(&pb.StreamResponse{Status: "error", Error: err.Error(), Done: true})
				return nil
			}
			stream.Send(&pb.StreamResponse{Status: "done", Done: true})
			return nil
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

func (s *server) SnapshotList(ctx context.Context, req *pb.SnapshotListRequest) (*pb.SnapshotListResponse, error) {
	mgr := snapshot.NewManager(docker.NewDockerRuntime(), req.ProjectPath)
	infos, err := mgr.List()
	if err != nil {
		return &pb.SnapshotListResponse{}, nil
	}

	var snapshots []*pb.Snapshot
	for _, info := range infos {
		snapshots = append(snapshots, &pb.Snapshot{
			Id:         info.ID,
			Name:       info.Name,
			SizeBytes:  info.SizeBytes,
			CreatedAt:  info.CreatedAt.Format(time.RFC3339),
		})
	}

	return &pb.SnapshotListResponse{Snapshots: snapshots}, nil
}

func (s *server) SnapshotDelete(ctx context.Context, req *pb.SnapshotDeleteRequest) (*pb.StatusResponse, error) {
	mgr := snapshot.NewManager(docker.NewDockerRuntime(), req.ProjectPath)
	if err := mgr.Delete(req.SnapshotId); err != nil {
		return &pb.StatusResponse{Status: "error", Error: err.Error()}, nil
	}
	return &pb.StatusResponse{Status: "ok"}, nil
}

func (s *server) Doctor(ctx context.Context, req *pb.DoctorRequest) (*pb.DoctorResponse, error) {
	var issues []*pb.DiagnosticIssue

	// Check Docker
	rt := docker.NewDockerRuntime()
	if err := rt.Connect(ctx); err != nil {
		issues = append(issues, &pb.DiagnosticIssue{
			Severity: "error",
			Message:  "Docker daemon is not running",
			Details:  err.Error(),
		})
		issues = append(issues, &pb.DiagnosticIssue{
			Severity: "info",
			Message:  "Start Docker Desktop or run: systemctl start docker",
		})
	} else {
		rt.Close()
		issues = append(issues, &pb.DiagnosticIssue{
			Severity: "info",
			Message:  "Docker daemon is running",
		})
	}

	// Check config
	parser := config.NewParser()
	cfg, err := parser.Parse(req.ProjectPath)
	if err != nil {
		issues = append(issues, &pb.DiagnosticIssue{
			Severity: "error",
			Message:  "No devbox.yml found",
			Details:  err.Error(),
		})
		issues = append(issues, &pb.DiagnosticIssue{
			Severity: "info",
			Message:  "Run 'devbox init' to create a configuration",
		})
	} else {
		issues = append(issues, &pb.DiagnosticIssue{
			Severity: "info",
			Message:  fmt.Sprintf("Configuration valid: %s (%d services)", cfg.Name, len(cfg.Services)),
		})
	}

	return &pb.DoctorResponse{Issues: issues}, nil
}

func (s *server) Reset(req *pb.ResetRequest, stream pb.EngineService_ResetServer) error {
	parser := config.NewParser()
	cfg, err := parser.Parse(req.ProjectPath)
	if err != nil {
		stream.Send(&pb.StreamResponse{Status: "error", Error: err.Error(), Done: true})
		return nil
	}

	rt := docker.NewDockerRuntime()
	if err := rt.Connect(stream.Context()); err != nil {
		stream.Send(&pb.StreamResponse{Status: "error", Error: "Docker not available", Done: true})
		return nil
	}
	defer rt.Close()

	orch, err := orchestrator.NewOrchestrator(rt, req.ProjectPath, cfg)
	if err != nil {
		stream.Send(&pb.StreamResponse{Status: "error", Error: err.Error(), Done: true})
		return nil
	}

	statusChan := make(chan string, 64)
	done := make(chan error, 1)

	go func() {
		done <- orch.Stop(stream.Context(), cfg, 30, statusChan)
	}()

	for {
		select {
		case msg := <-statusChan:
			stream.Send(&pb.StreamResponse{Status: "info", Message: msg})
		case err := <-done:
			if err != nil {
				stream.Send(&pb.StreamResponse{Status: "error", Error: err.Error(), Done: true})
				return nil
			}
			stream.Send(&pb.StreamResponse{Status: "info", Message: "Cleaning up..."})
			err = orch.Start(stream.Context(), cfg, req.ProjectPath, statusChan)
			if err != nil {
				stream.Send(&pb.StreamResponse{Status: "error", Error: err.Error(), Done: true})
				return nil
			}
			stream.Send(&pb.StreamResponse{Status: "done", Done: true})
			return nil
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

func main() {
	// Initialize platform-specific directories
	configDir := platform.ConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Fatalf("Failed to create config directory: %v", err)
	}

	devboxDir := platform.DevBoxDir(".")
	if err := os.MkdirAll(devboxDir, 0755); err != nil {
		log.Fatalf("Failed to create devbox directory: %v", err)
	}

	// Initialize state manager
	stateMgr, err := state.NewManager()
	if err != nil {
		log.Fatalf("Failed to initialize state: %v", err)
	}
	defer stateMgr.Close()

	// Platform-specific listener
	var lis net.Listener
	socketPath := platform.EngineSocketPath()

	if platform.IsWindows() {
		lis, err = net.Listen("tcp", "127.0.0.1:"+platform.DefaultEnginePort())
		if err != nil {
			log.Fatalf("Failed to listen on TCP: %v", err)
		}
		fmt.Printf("Engine listening on TCP 127.0.0.1:%s (Windows)\n", platform.DefaultEnginePort())
	} else {
		os.Remove(socketPath)
		lis, err = net.Listen("unix", socketPath)
		if err != nil {
			log.Fatalf("Failed to listen on %s: %v", socketPath, err)
		}
		defer os.Remove(socketPath)
		fmt.Printf("Engine listening on %s (%s)\n", socketPath, platform.Detect())
	}

	s := grpc.NewServer()
	svc := &server{
		startedAt: time.Now(),
		stateMgr:  stateMgr,
	}
	pb.RegisterEngineServiceServer(s, svc)

	go func() {
		sigCh := make(chan os.Signal, 1)
		if platform.IsWindows() {
			signal.Notify(sigCh, os.Interrupt)
		} else {
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		}
		<-sigCh
		fmt.Println("\nShutting down engine daemon...")
		s.GracefulStop()
	}()

	fmt.Printf("DevBoxOS Engine v%s started\n", version)
	fmt.Println("Press Ctrl+C to stop")

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
