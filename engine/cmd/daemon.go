package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/devboxos/devboxos/shared/config"
	"github.com/devboxos/devboxos/engine/internal/orchestrator"
	engineruntime "github.com/devboxos/devboxos/shared/runtime"
	dockerruntime "github.com/devboxos/devboxos/shared/runtime/docker"
	"github.com/devboxos/devboxos/engine/internal/state"
	pb "github.com/devboxos/devboxos/engine/proto"
	"google.golang.org/grpc"
)

const (
	version = "0.1.0-dev"
)

// server implements the EngineService gRPC server.
type server struct {
	pb.UnimplementedEngineServiceServer
	startedAt    time.Time
	stateMgr     *state.Manager
	orchestrator *orchestrator.Orchestrator
	rt           engineruntime.Runtime
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
	if s.rt == nil {
		rt := dockerruntime.NewDockerRuntime()
		if err := rt.Connect(stream.Context()); err != nil {
			send("error", fmt.Sprintf("Docker not available: %v", err), true)
			return err
		}
		s.rt = rt
	}

	send("info", "Connected to Docker daemon", false)

	// Create orchestrator
	orch, err := orchestrator.NewOrchestrator(s.rt, req.ProjectPath, cfg)
	if err != nil {
		send("error", fmt.Sprintf("Failed to create orchestrator: %v", err), true)
		return err
	}

	// Store orchestrator for log collection and status tracking
	s.orchestrator = orch

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

	if s.rt == nil {
		return &pb.StatusResponse{
			Status: "error",
			Error:  "Docker not available: engine not started",
		}, nil
	}

	// Use stored orchestrator
	if s.orchestrator == nil {
		return &pb.StatusResponse{
			Status: "error",
			Error:  "No active environment. Run 'devbox start' first.",
		}, nil
	}

	// Status channel
	statusChan := make(chan string, 64)
	errChan := make(chan error, 1)

	go func() {
		gracePeriod := 30
		if req.GracePeriodSeconds > 0 {
			gracePeriod = int(req.GracePeriodSeconds)
		}
		errChan <- s.orchestrator.Stop(ctx, cfg, gracePeriod, statusChan)
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
	if s.orchestrator == nil {
		return &pb.StatusResponse{
			Status: "stopped",
		}, nil
	}

	envStatus, err := s.orchestrator.Status(ctx, cfg)
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
	rt := dockerruntime.NewDockerRuntime()
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
	reader, err := rt.StreamLogs(stream.Context(), containerID, engineruntime.LogOptions{
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
	stream.Send(&pb.StreamResponse{
		Status:  "info",
		Message: "Snapshot engine coming in Sprint 9-10",
	})
	stream.Send(&pb.StreamResponse{
		Status: "done",
		Done:   true,
	})
	return nil
}

func (s *server) SnapshotLoad(req *pb.SnapshotLoadRequest, stream pb.EngineService_SnapshotLoadServer) error {
	stream.Send(&pb.StreamResponse{
		Status:  "info",
		Message: "Snapshot engine coming in Sprint 9-10",
	})
	stream.Send(&pb.StreamResponse{
		Status: "done",
		Done:   true,
	})
	return nil
}

func (s *server) SnapshotList(ctx context.Context, req *pb.SnapshotListRequest) (*pb.SnapshotListResponse, error) {
	return &pb.SnapshotListResponse{}, nil
}

func (s *server) SnapshotDelete(ctx context.Context, req *pb.SnapshotDeleteRequest) (*pb.StatusResponse, error) {
	return &pb.StatusResponse{Status: "ok"}, nil
}

func (s *server) Doctor(ctx context.Context, req *pb.DoctorRequest) (*pb.DoctorResponse, error) {
	var issues []*pb.DiagnosticIssue

	// Check Docker
	rt := dockerruntime.NewDockerRuntime()
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
	stream.Send(&pb.StreamResponse{
		Status:  "info",
		Message: "Reset coming in Sprint 13-14",
	})
	stream.Send(&pb.StreamResponse{
		Status: "done",
		Done:   true,
	})
	return nil
}

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %v", err)
	}

	devboxDir := filepath.Join(homeDir, ".devbox")
	if err := os.MkdirAll(devboxDir, 0755); err != nil {
		log.Fatalf("Failed to create %s: %v", devboxDir, err)
	}

	// Initialize state manager
	stateMgr, err := state.NewManager()
	if err != nil {
		log.Fatalf("Failed to initialize state: %v", err)
	}
	defer stateMgr.Close()

	// Determine socket path
	socketPath := filepath.Join(devboxDir, "engine.sock")

	var lis net.Listener
	if os.PathSeparator == '\\' {
		lis, err = net.Listen("tcp", "127.0.0.1:51000")
		if err != nil {
			log.Fatalf("Failed to listen on TCP: %v", err)
		}
		fmt.Printf("Engine listening on TCP 127.0.0.1:51000 (Windows mode)\n")
	} else {
		os.Remove(socketPath)
		lis, err = net.Listen("unix", socketPath)
		if err != nil {
			log.Fatalf("Failed to listen on %s: %v", socketPath, err)
		}
		fmt.Printf("Engine listening on %s\n", socketPath)
	}

	s := grpc.NewServer()
	svc := &server{
		startedAt: time.Now(),
		stateMgr:  stateMgr,
	}
	pb.RegisterEngineServiceServer(s, svc)

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		fmt.Println("\nShutting down engine daemon...")
		s.GracefulStop()
		os.Remove(socketPath)
		os.Exit(0)
	}()

	fmt.Printf("DevBoxOS Engine v%s started\n", version)
	fmt.Println("Press Ctrl+C to stop")

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
