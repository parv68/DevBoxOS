package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/devboxos/devboxos/engine/internal/orchestrator"
	"github.com/devboxos/devboxos/engine/internal/state"
	pb "github.com/devboxos/devboxos/engine/proto"
	"github.com/devboxos/devboxos/shared/config"
	"github.com/devboxos/devboxos/shared/platform"
	"github.com/devboxos/devboxos/shared/runtime"
	"github.com/devboxos/devboxos/shared/runtime/docker"
	"github.com/devboxos/devboxos/shared/runtime/host"
	"github.com/devboxos/devboxos/shared/secrets"
	"github.com/devboxos/devboxos/shared/snapshot"
	"github.com/devboxos/devboxos/shared/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"google.golang.org/grpc"
)

var version = "0.1.0-dev"

type server struct {
	pb.UnimplementedEngineServiceServer
	startedAt    time.Time
	stateMgr     *state.Manager
	orchestrator *orchestrator.Orchestrator
	rt           runtime.Runtime // backward compat, points to hostRt
	hostRt       runtime.Runtime
	dockerRt     runtime.Runtime
	grpcServer   *grpc.Server
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
		resp := &pb.StreamResponse{
			Status:  status,
			Message: msg,
			Done:    done,
		}
		if done && status == "error" {
			resp.Error = msg
		}
		stream.Send(resp)
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

	// Choose runtime: Docker only if project has Docker services
	needsDocker := config.NeedsDocker(cfg)
	s.mu.Lock()
	if s.hostRt == nil {
		s.mu.Unlock()

		// Always create host runtime
		hostRt := host.NewHostRuntime()
		hostRt.SetVolumeRoot(filepath.Join(req.ProjectPath, ".devbox", "volumes"))

		var dockerRt runtime.Runtime
		if needsDocker {
			dockerRt = docker.NewDockerRuntime()
			if err := dockerRt.Connect(stream.Context()); err != nil {
				send("error", fmt.Sprintf("Docker not available: %v", err), true)
				return err
			}
			send("info", "Connected to Docker daemon", false)
		}

		s.mu.Lock()
		s.hostRt = hostRt
		s.dockerRt = dockerRt
		s.rt = hostRt
		s.mu.Unlock()

		if !needsDocker {
			send("info", "Using host process runtime (no Docker needed)", false)
		}
	} else {
		s.mu.Unlock()
		send("info", "Runtime already available", false)
	}

	// Create orchestrator with both runtimes
	orch, err := orchestrator.NewOrchestrator(s.dockerRt, s.hostRt, req.ProjectPath, cfg)
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
	s.mu.Lock()
	rt := s.rt
	s.mu.Unlock()

	if rt == nil {
		return fmt.Errorf("no active environment. Run 'devbox start' first.")
	}

	// Find container/process
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

	// Use stored runtime if available, otherwise create host runtime
	s.mu.Lock()
	rt := s.rt
	s.mu.Unlock()
	var hr *host.HostRuntime
	if rt == nil {
		hr = host.NewHostRuntime()
		hr.SetVolumeRoot(filepath.Join(req.ProjectPath, ".devbox", "volumes"))
		rt = hr
	} else if h, ok := rt.(*host.HostRuntime); ok {
		h.SetVolumeRoot(filepath.Join(req.ProjectPath, ".devbox", "volumes"))
		hr = h
	}

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
	// Use stored runtime if available, otherwise create host runtime
	s.mu.Lock()
	rt := s.rt
	s.mu.Unlock()
	if rt == nil {
		hr := host.NewHostRuntime()
		hr.SetVolumeRoot(filepath.Join(req.ProjectPath, ".devbox", "volumes"))
		rt = hr
	} else if hr, ok := rt.(*host.HostRuntime); ok {
		hr.SetVolumeRoot(filepath.Join(req.ProjectPath, ".devbox", "volumes"))
	}

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
	mgr := snapshot.NewManager(host.NewHostRuntime(), req.ProjectPath)
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
	mgr := snapshot.NewManager(host.NewHostRuntime(), req.ProjectPath)
	id, err := resolveSnapshotID(mgr, req.SnapshotId)
	if err != nil {
		return &pb.StatusResponse{Status: "error", Error: err.Error()}, nil
	}
	if err := mgr.Delete(id); err != nil {
		return &pb.StatusResponse{Status: "error", Error: err.Error()}, nil
	}
	return &pb.StatusResponse{Status: "ok"}, nil
}

func (s *server) SnapshotExport(req *pb.SnapshotExportRequest, stream pb.EngineService_SnapshotExportServer) error {
	mgr := snapshot.NewManager(host.NewHostRuntime(), req.ProjectPath)

	snapshotID := req.SnapshotId
	if snapshotID == "" {
		infos, err := mgr.List()
		if err != nil || len(infos) == 0 {
			stream.Send(&pb.StreamResponse{Status: "error", Error: "No snapshots found", Done: true})
			return nil
		}
		snapshotID = infos[len(infos)-1].ID
	} else {
		id, err := resolveSnapshotID(mgr, snapshotID)
		if err != nil {
			stream.Send(&pb.StreamResponse{Status: "error", Error: err.Error(), Done: true})
			return nil
		}
		snapshotID = id
	}

	statusChan := make(chan string, 64)
	done := make(chan error, 1)

	go func() {
		done <- mgr.Export(snapshotID, req.ExportPath, statusChan)
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

func (s *server) SnapshotImport(req *pb.SnapshotImportRequest, stream pb.EngineService_SnapshotImportServer) error {
	mgr := snapshot.NewManager(host.NewHostRuntime(), req.ProjectPath)

	statusChan := make(chan string, 64)
	done := make(chan error, 1)

	go func() {
		done <- mgr.Import(req.ImportPath, statusChan)
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

func (s *server) Build(req *pb.BuildRequest, stream pb.EngineService_BuildServer) error {
	send := func(status, msg string, done bool) {
		resp := &pb.StreamResponse{
			Status:  status,
			Message: msg,
			Done:    done,
		}
		if done && status == "error" {
			resp.Error = msg
		}
		stream.Send(resp)
	}

	parser := config.NewParser()
	cfg, err := parser.Parse(req.ProjectPath)
	if err != nil {
		send("error", fmt.Sprintf("Failed to parse config: %v", err), true)
		return nil
	}

	rt := docker.NewDockerRuntime()
	if err := rt.Connect(stream.Context()); err != nil {
		send("error", "Docker not available", true)
		return nil
	}
	defer rt.Close()

	allServices := cfg.Services
	filtered := make(map[string]types.Service)
	if req.Service != "" {
		svc, ok := allServices[req.Service]
		if !ok {
			send("error", fmt.Sprintf("Service not found: %s", req.Service), true)
			return nil
		}
		filtered[req.Service] = svc
	} else {
		filtered = allServices
	}

	for svcName, svc := range filtered {
		if svc.Build == nil || svc.Build.Context == "" {
			if req.Service != "" {
				send("error", fmt.Sprintf("Service %s has no build configuration", svcName), true)
				return nil
			}
			continue
		}

		buildCtx := svc.Build.Context
		if !filepath.IsAbs(buildCtx) {
			buildCtx = filepath.Join(req.ProjectPath, buildCtx)
		}

		tags := []string{fmt.Sprintf("devboxos-%s:latest", svcName)}
		if svc.Image != "" {
			tags = []string{svc.Image}
		}

		statusChan := make(chan string, 64)
		done := make(chan error, 1)

		send("info", fmt.Sprintf("Building service: %s", svcName), false)

		go func() {
			_, err := rt.BuildImage(stream.Context(), runtime.BuildConfig{
				ContextDir: buildCtx,
				Dockerfile: svc.Build.Dockerfile,
				BuildArgs:  svc.Build.Args,
				Tags:       tags,
				NoCache:    req.NoCache,
				Pull:       req.Pull,
			}, statusChan)
			done <- err
		}()

		buildDone := false
		for !buildDone {
			select {
			case msg := <-statusChan:
				send("info", msg, false)
			case err := <-done:
				if err != nil {
					send("error", fmt.Sprintf("Build failed for %s: %v", svcName, err), true)
					return nil
				}
				send("info", fmt.Sprintf("Service %s built", svcName), false)
				buildDone = true
			case <-stream.Context().Done():
				return stream.Context().Err()
			}
		}
	}

	send("done", "All services built", true)
	return nil
}

func (s *server) Exec(ctx context.Context, req *pb.ExecRequest) (*pb.ExecResponse, error) {
	s.mu.Lock()
	rt := s.rt
	s.mu.Unlock()

	// If we have a stored runtime, determine if it's Docker or Host
	if rt != nil {
		// Try to find the service's container/process via the runtime
		containers, err := rt.ListContainers(ctx, map[string]string{
			"devboxos.service": req.Service,
		})
		if err == nil && len(containers) > 0 {
			id := containers[0].ID
			info, err := rt.GetContainerInfo(ctx, id)
			if err == nil {
				// For host processes, spawn command directly
				if info.Image == "" {
					cmd := exec.CommandContext(ctx, req.Command, req.Args...)
					var stdoutBuf, stderrBuf bytes.Buffer
					cmd.Stdout = &stdoutBuf
					cmd.Stderr = &stderrBuf
					exitCode := 0
					if err := cmd.Run(); err != nil {
						if exitErr, ok := err.(*exec.ExitError); ok {
							exitCode = exitErr.ExitCode()
							// stderr already captured
						} else {
							exitCode = -1
						}
					}
					return &pb.ExecResponse{
						Stdout:   stdoutBuf.String(),
						Stderr:   stderrBuf.String(),
						ExitCode: int32(exitCode),
					}, nil
				}
			}
		}
	}

	// Fall back to Docker SDK for container-based services
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return &pb.ExecResponse{Stderr: fmt.Sprintf("Docker not available: %v\n\nService does not use Docker containers. Try running the command directly.", err), ExitCode: -1}, nil
	}

	containers, err := dockerClient.ContainerList(ctx, container.ListOptions{
		Filters: filters.NewArgs(
			filters.Arg("label", "devboxos.service="+req.Service),
		),
	})
	if err != nil {
		return &pb.ExecResponse{Stderr: fmt.Sprintf("List containers: %v", err), ExitCode: -1}, nil
	}

	if len(containers) == 0 {
		return &pb.ExecResponse{Stderr: fmt.Sprintf("No running container found for service: %s", req.Service), ExitCode: -1}, nil
	}

	containerID := containers[0].ID

	execConfig := container.ExecOptions{
		Cmd:          append([]string{req.Command}, req.Args...),
		AttachStdout: true,
		AttachStderr: true,
	}
	execResp, err := dockerClient.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return &pb.ExecResponse{Stderr: fmt.Sprintf("Create exec: %v", err), ExitCode: -1}, nil
	}

	attachResp, err := dockerClient.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{})
	if err != nil {
		return &pb.ExecResponse{Stderr: fmt.Sprintf("Attach exec: %v", err), ExitCode: -1}, nil
	}
	defer attachResp.Close()

	var stdoutBuf, stderrBuf strings.Builder
	_, err = stdcopy.StdCopy(&stdoutBuf, &stderrBuf, attachResp.Reader)
	if err != nil {
		return &pb.ExecResponse{Stderr: fmt.Sprintf("Read exec output: %v", err), ExitCode: -1}, nil
	}

	inspectResp, err := dockerClient.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return &pb.ExecResponse{
			Stdout:   stdoutBuf.String(),
			Stderr:   stderrBuf.String(),
			ExitCode: -1,
		}, nil
	}

	return &pb.ExecResponse{
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
		ExitCode: int32(inspectResp.ExitCode),
	}, nil
}

func (s *server) Doctor(ctx context.Context, req *pb.DoctorRequest) (*pb.DoctorResponse, error) {
	var issues []*pb.DiagnosticIssue

	// Check config
	parser := config.NewParser()
	cfg, parseErr := parser.Parse(req.ProjectPath)
	if parseErr != nil {
		issues = append(issues, &pb.DiagnosticIssue{
			Severity: "error",
			Message:  "No devbox.yml found",
			Details:  parseErr.Error(),
		})
		issues = append(issues, &pb.DiagnosticIssue{
			Severity: "info",
			Message:  "Run 'devbox init' to create a configuration",
		})
	} else {
		needsDocker := config.NeedsDocker(cfg)
		issues = append(issues, &pb.DiagnosticIssue{
			Severity: "info",
			Message:  fmt.Sprintf("Configuration valid: %s (%d services)", cfg.Name, len(cfg.Services)),
		})

		if needsDocker {
			// Check Docker only if project needs it
			rt := docker.NewDockerRuntime()
			if err := rt.Connect(ctx); err != nil {
				issues = append(issues, &pb.DiagnosticIssue{
					Severity: "error",
					Message:  "Docker daemon is not running (required by this project)",
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
		} else {
			issues = append(issues, &pb.DiagnosticIssue{
				Severity: "info",
				Message:  "Project does not require Docker (using host process runtime)",
			})
		}
	}

	var suggestions []string

	for _, issue := range issues {
		switch {
		case strings.Contains(issue.Message, "Docker"):
			suggestions = append(suggestions, "Start Docker Desktop and ensure it is running")
			suggestions = append(suggestions, "On Windows, ensure Docker Desktop uses WSL 2 backend")
		case strings.Contains(issue.Message, "devbox.yml") || strings.Contains(issue.Message, "Configuration"):
			suggestions = append(suggestions, "Create a configuration: devbox init")
			suggestions = append(suggestions, "Import from Docker Compose: devbox init compose-import ./docker-compose.yml")
		}
	}

	if len(suggestions) == 0 {
		if req.ProjectPath != "" {
			if _, err := os.Stat(filepath.Join(req.ProjectPath, "devbox.yml")); os.IsNotExist(err) {
				suggestions = append(suggestions, "Run 'devbox init' to create a configuration")
			}
		}
		suggestions = append(suggestions, "Run 'devbox start' to start services")
	}

	return &pb.DoctorResponse{Issues: issues, Suggestions: suggestions}, nil
}

func (s *server) Reset(req *pb.ResetRequest, stream pb.EngineService_ResetServer) error {
	parser := config.NewParser()
	cfg, err := parser.Parse(req.ProjectPath)
	if err != nil {
		stream.Send(&pb.StreamResponse{Status: "error", Error: err.Error(), Done: true})
		return nil
	}

	s.mu.Lock()
	hostRt := s.hostRt
	dockerRt := s.dockerRt
	s.mu.Unlock()
	if hostRt == nil {
		stream.Send(&pb.StreamResponse{Status: "error", Error: "No active environment. Run 'devbox start' first.", Done: true})
		return nil
	}

	orch, err := orchestrator.NewOrchestrator(dockerRt, hostRt, req.ProjectPath, cfg)
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

func getSecretStore(projectPath string) (*secrets.Store, *secrets.AgeCrypto, error) {
	keyPath := filepath.Join(projectPath, ".devbox", "secrets.key")
	storePath := filepath.Join(projectPath, ".devbox", "secrets.enc")

	crypto, err := secrets.LoadOrCreateKey(keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("load age key: %w", err)
	}

	store := secrets.NewStore(crypto, storePath)
	if err := store.Load(); err != nil {
		return nil, nil, fmt.Errorf("load secret store: %w", err)
	}

	return store, crypto, nil
}

func (s *server) SecretSet(ctx context.Context, req *pb.SecretSetRequest) (*pb.StatusResponse, error) {
	store, _, err := getSecretStore(req.ProjectPath)
	if err != nil {
		return &pb.StatusResponse{Status: "error", Error: err.Error()}, nil
	}

	if err := store.Set(req.Name, req.Value, "manual"); err != nil {
		return &pb.StatusResponse{Status: "error", Error: err.Error()}, nil
	}

	return &pb.StatusResponse{Status: "ok"}, nil
}

func (s *server) SecretGet(ctx context.Context, req *pb.SecretGetRequest) (*pb.SecretGetResponse, error) {
	store, _, err := getSecretStore(req.ProjectPath)
	if err != nil {
		return &pb.SecretGetResponse{Error: err.Error()}, nil
	}

	entry, err := store.Get(req.Name)
	if err != nil {
		return &pb.SecretGetResponse{Error: err.Error()}, nil
	}

	return &pb.SecretGetResponse{
		Name:  entry.Name,
		Value: entry.Value,
	}, nil
}

func (s *server) SecretList(ctx context.Context, req *pb.SecretListRequest) (*pb.SecretListResponse, error) {
	store, _, err := getSecretStore(req.ProjectPath)
	if err != nil {
		return &pb.SecretListResponse{Error: err.Error()}, nil
	}

	entries := store.List()
	var pbEntries []*pb.SecretEntry
	for _, e := range entries {
		pbEntries = append(pbEntries, &pb.SecretEntry{
			Name:      e.Name,
			Source:    e.Source,
			CreatedAt: e.CreatedAt.Format(time.RFC3339),
			UpdatedAt: e.UpdatedAt.Format(time.RFC3339),
		})
	}

	return &pb.SecretListResponse{Secrets: pbEntries}, nil
}

func (s *server) SecretDelete(ctx context.Context, req *pb.SecretDeleteRequest) (*pb.StatusResponse, error) {
	store, _, err := getSecretStore(req.ProjectPath)
	if err != nil {
		return &pb.StatusResponse{Status: "error", Error: err.Error()}, nil
	}

	if err := store.Delete(req.Name); err != nil {
		return &pb.StatusResponse{Status: "error", Error: err.Error()}, nil
	}

	return &pb.StatusResponse{Status: "ok"}, nil
}

func (s *server) SecretRotate(ctx context.Context, req *pb.SecretRotateRequest) (*pb.StatusResponse, error) {
	projectPath := req.ProjectPath
	keyPath := filepath.Join(projectPath, ".devbox", "secrets.key")
	storePath := filepath.Join(projectPath, ".devbox", "secrets.enc")

	resolver, err := secrets.NewResolver(projectPath, keyPath, storePath)
	if err != nil {
		return &pb.StatusResponse{Status: "error", Error: err.Error()}, nil
	}

	if err := resolver.Rotate(req.Name); err != nil {
		return &pb.StatusResponse{Status: "error", Error: err.Error()}, nil
	}

	return &pb.StatusResponse{Status: "ok"}, nil
}

// Shutdown gracefully stops the engine daemon.
func (s *server) Shutdown(ctx context.Context, req *pb.ShutdownRequest) (*pb.ShutdownResponse, error) {
	go s.grpcServer.GracefulStop()
	return &pb.ShutdownResponse{}, nil
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
		startedAt:  time.Now(),
		stateMgr:   stateMgr,
		grpcServer: s,
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

// resolveSnapshotID resolves a snapshot name or ID to a snapshot directory ID.
func resolveSnapshotID(mgr *snapshot.Manager, input string) (string, error) {
	// Try as ID first (fast path — directory exists)
	infos, err := mgr.List()
	if err != nil {
		return "", fmt.Errorf("list snapshots: %w", err)
	}

	for _, info := range infos {
		if info.ID == input || strings.HasPrefix(info.ID, input) {
			return info.ID, nil
		}
	}

	// Try as name
	for _, info := range infos {
		if info.Name == input {
			return info.ID, nil
		}
	}

	return "", fmt.Errorf("snapshot %q not found", input)
}
