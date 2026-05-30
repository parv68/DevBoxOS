package client

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	pb "github.com/devboxos/devboxos/engine/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type mockSnapshot struct {
	ID, Name   string
	SizeBytes  int64
	CreatedAt  string
	IncludeLogs bool
}

type mockServer struct {
	pb.UnimplementedEngineServiceServer
	mu          sync.Mutex
	secrets     map[string]string
	secretSrc   map[string]string
	config      map[string]string
	snapshots   map[string]mockSnapshot
	snapCounter int
}

func newMockServer() *mockServer {
	return &mockServer{
		secrets:   make(map[string]string),
		secretSrc: make(map[string]string),
		config:    map[string]string{"telemetry": "true", "engine": "auto"},
		snapshots: make(map[string]mockSnapshot),
	}
}

func (m *mockServer) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{
		Version: "0.1.0-test",
		Uptime:  42,
	}, nil
}

func (m *mockServer) Start(req *pb.StartRequest, stream pb.EngineService_StartServer) error {
	stream.Send(&pb.StreamResponse{Status: "info", Message: "Loading config..."})
	stream.Send(&pb.StreamResponse{Status: "info", Message: "Starting services..."})
	stream.Send(&pb.StreamResponse{Status: "info", Message: "All services started", Done: true})
	return nil
}

func (m *mockServer) Stop(ctx context.Context, req *pb.StopRequest) (*pb.StatusResponse, error) {
	return &pb.StatusResponse{Status: "stopped"}, nil
}

func (m *mockServer) Status(ctx context.Context, req *pb.StatusRequest) (*pb.StatusResponse, error) {
	return &pb.StatusResponse{
		Status: "running",
		Services: []*pb.ServiceStatus{
			{Name: "web", Status: "running", Health: "healthy", Port: 8080, ContainerId: "abc123"},
			{Name: "db", Status: "running", Health: "healthy", Port: 5432, ContainerId: "def456"},
		},
	}, nil
}

func (m *mockServer) Logs(req *pb.LogsRequest, stream pb.EngineService_LogsServer) error {
	entries := []string{"log line 1", "log line 2", "log line 3"}
	for _, entry := range entries {
		stream.Send(&pb.LogEntry{
			Service:   req.Service,
			Timestamp: time.Now().Format(time.RFC3339),
			Level:     "info",
			Message:   entry,
		})
	}
	return nil
}

func (m *mockServer) Doctor(ctx context.Context, req *pb.DoctorRequest) (*pb.DoctorResponse, error) {
	return &pb.DoctorResponse{
		Issues: []*pb.DiagnosticIssue{
			{Severity: "info", Message: "Docker is running"},
			{Severity: "info", Message: fmt.Sprintf("Config found at %s", req.ProjectPath)},
		},
	}, nil
}

func (m *mockServer) Reset(req *pb.ResetRequest, stream pb.EngineService_ResetServer) error {
	stream.Send(&pb.StreamResponse{Status: "info", Message: "Stopping services..."})
	stream.Send(&pb.StreamResponse{Status: "info", Message: "Starting services..."})
	stream.Send(&pb.StreamResponse{Status: "done", Done: true})
	return nil
}

func (m *mockServer) SecretSet(ctx context.Context, req *pb.SecretSetRequest) (*pb.StatusResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.secrets[req.Name] = req.Value
	m.secretSrc[req.Name] = "manual"
	return &pb.StatusResponse{Status: "ok"}, nil
}

func (m *mockServer) SecretGet(ctx context.Context, req *pb.SecretGetRequest) (*pb.SecretGetResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	val, ok := m.secrets[req.Name]
	if !ok {
		return &pb.SecretGetResponse{Error: fmt.Sprintf("secret %s not found", req.Name)}, nil
	}
	return &pb.SecretGetResponse{Name: req.Name, Value: val}, nil
}

func (m *mockServer) SecretList(ctx context.Context, req *pb.SecretListRequest) (*pb.SecretListResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var entries []*pb.SecretEntry
	for name := range m.secrets {
		entries = append(entries, &pb.SecretEntry{
			Name:    name,
			Source:  m.secretSrc[name],
			CreatedAt: time.Now().Format(time.RFC3339),
			UpdatedAt: time.Now().Format(time.RFC3339),
		})
	}
	return &pb.SecretListResponse{Secrets: entries}, nil
}

func (m *mockServer) SecretDelete(ctx context.Context, req *pb.SecretDeleteRequest) (*pb.StatusResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.secrets, req.Name)
	delete(m.secretSrc, req.Name)
	return &pb.StatusResponse{Status: "ok"}, nil
}

func (m *mockServer) SecretRotate(ctx context.Context, req *pb.SecretRotateRequest) (*pb.StatusResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	src, ok := m.secretSrc[req.Name]
	if !ok {
		return &pb.StatusResponse{Status: "error", Error: "secret not found"}, nil
	}
	if src == "manual" {
		return &pb.StatusResponse{Status: "error", Error: "cannot rotate manual secret"}, nil
	}
	m.secrets[req.Name] = "rotated-new-value"
	return &pb.StatusResponse{Status: "ok"}, nil
}

func (m *mockServer) SnapshotSave(req *pb.SnapshotSaveRequest, stream pb.EngineService_SnapshotSaveServer) error {
	m.mu.Lock()
	id := fmt.Sprintf("snap-%d", m.snapCounter)
	m.snapCounter++
	m.snapshots[id] = mockSnapshot{
		ID:           id,
		Name:         req.Name,
		SizeBytes:    4096,
		CreatedAt:    time.Now().Format(time.RFC3339),
		IncludeLogs:  req.IncludeLogs,
	}
	m.mu.Unlock()

	stream.Send(&pb.StreamResponse{Status: "info", Message: "Committing containers..."})
	stream.Send(&pb.StreamResponse{Status: "info", Message: "Compressing layers..."})
	stream.Send(&pb.StreamResponse{Status: "done", Done: true})
	return nil
}

func (m *mockServer) SnapshotLoad(req *pb.SnapshotLoadRequest, stream pb.EngineService_SnapshotLoadServer) error {
	m.mu.Lock()
	_, ok := m.snapshots[req.SnapshotId]
	m.mu.Unlock()

	if !ok {
		stream.Send(&pb.StreamResponse{Status: "error", Error: "snapshot not found", Done: true})
		return nil
	}

	stream.Send(&pb.StreamResponse{Status: "info", Message: "Loading snapshot..."})
	stream.Send(&pb.StreamResponse{Status: "info", Message: "Restoring containers..."})
	stream.Send(&pb.StreamResponse{Status: "done", Done: true})
	return nil
}

func (m *mockServer) SnapshotList(ctx context.Context, req *pb.SnapshotListRequest) (*pb.SnapshotListResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var snapshots []*pb.Snapshot
	for _, s := range m.snapshots {
		snapshots = append(snapshots, &pb.Snapshot{
			Id:        s.ID,
			Name:      s.Name,
			SizeBytes: s.SizeBytes,
			CreatedAt: s.CreatedAt,
		})
	}
	return &pb.SnapshotListResponse{Snapshots: snapshots}, nil
}

func (m *mockServer) SnapshotDelete(ctx context.Context, req *pb.SnapshotDeleteRequest) (*pb.StatusResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.snapshots[req.SnapshotId]; !ok {
		return &pb.StatusResponse{Status: "error", Error: "snapshot not found"}, nil
	}
	delete(m.snapshots, req.SnapshotId)
	return &pb.StatusResponse{Status: "ok"}, nil
}

func startMockServer() (*mockServer, pb.EngineServiceClient, func(), error) {
	ms := newMockServer()

	lis := bufconn.Listen(1024 * 1024)
	gs := grpc.NewServer()
	pb.RegisterEngineServiceServer(gs, ms)

	go gs.Serve(lis)

	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		gs.GracefulStop()
		return nil, nil, nil, fmt.Errorf("grpc.Dial: %w", err)
	}

	client := pb.NewEngineServiceClient(conn)

	cleanup := func() {
		conn.Close()
		gs.GracefulStop()
	}

	return ms, client, cleanup, nil
}
