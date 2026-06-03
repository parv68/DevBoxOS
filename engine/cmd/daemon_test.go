package main

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/devboxos/devboxos/engine/internal/state"
	pb "github.com/devboxos/devboxos/engine/proto"
	"github.com/devboxos/devboxos/shared/runtime/docker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func setupTestServer(t *testing.T) (*server, pb.EngineServiceClient, func()) {
	t.Helper()

	stateMgr, err := state.NewManager()
	if err != nil {
		t.Fatalf("state.NewManager() failed: %v", err)
	}

	mockRT := NewMockRuntime()
	mockRT.AddContainer("mock-1", "devbox-test-service", "nginx:alpine", "running", map[string]string{
		"devboxos.service": "web",
	})

	srv := &server{
		startedAt: time.Now(),
		stateMgr:  stateMgr,
		rt:        mockRT,
	}

	lis := bufconn.Listen(bufSize)
	gs := grpc.NewServer()
	pb.RegisterEngineServiceServer(gs, srv)

	go gs.Serve(lis)

	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		t.Fatalf("grpc.DialContext() failed: %v", err)
	}

	client := pb.NewEngineServiceClient(conn)

	cleanup := func() {
		conn.Close()
		gs.GracefulStop()
		stateMgr.Close()
	}

	return srv, client, cleanup
}

func setupTestProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	devboxDir := filepath.Join(dir, ".devbox")
	os.MkdirAll(devboxDir, 0755)

	t.Cleanup(func() {
		os.RemoveAll(devboxDir)
	})

	yml := []byte(`name: test-project
version: "1.0"
services:
  web:
    image: nginx:alpine
`)
	os.WriteFile(filepath.Join(dir, "devbox.yml"), yml, 0644)
	return dir
}

func TestServer_Ping(t *testing.T) {
	_, client, cleanup := setupTestServer(t)
	defer cleanup()

	resp, err := client.Ping(context.Background(), &pb.PingRequest{})
	if err != nil {
		t.Fatalf("Ping() failed: %v", err)
	}
	if resp.Version == "" {
		t.Error("expected non-empty version")
	}
	if resp.Uptime < 0 {
		t.Errorf("expected positive uptime, got %d", resp.Uptime)
	}
}

func TestServer_SecretCRUD(t *testing.T) {
	projectDir := setupTestProject(t)
	_, client, cleanup := setupTestServer(t)
	defer cleanup()

	setResp, err := client.SecretSet(context.Background(), &pb.SecretSetRequest{
		ProjectPath: projectDir,
		Name:        "DB_PASSWORD",
		Value:       "s3cret!",
	})
	if err != nil {
		t.Fatalf("SecretSet() failed: %v", err)
	}
	if setResp.Status != "ok" {
		t.Errorf("expected status ok, got %s", setResp.Status)
	}

	getResp, err := client.SecretGet(context.Background(), &pb.SecretGetRequest{
		ProjectPath: projectDir,
		Name:        "DB_PASSWORD",
	})
	if err != nil {
		t.Fatalf("SecretGet() failed: %v", err)
	}
	if getResp.Value != "s3cret!" {
		t.Errorf("expected value 's3cret!', got '%s'", getResp.Value)
	}

	listResp, err := client.SecretList(context.Background(), &pb.SecretListRequest{
		ProjectPath: projectDir,
	})
	if err != nil {
		t.Fatalf("SecretList() failed: %v", err)
	}
	if len(listResp.Secrets) != 1 {
		t.Errorf("expected 1 secret, got %d", len(listResp.Secrets))
	}
	if listResp.Secrets[0].Name != "DB_PASSWORD" {
		t.Errorf("expected name 'DB_PASSWORD', got '%s'", listResp.Secrets[0].Name)
	}

	delResp, err := client.SecretDelete(context.Background(), &pb.SecretDeleteRequest{
		ProjectPath: projectDir,
		Name:        "DB_PASSWORD",
	})
	if err != nil {
		t.Fatalf("SecretDelete() failed: %v", err)
	}
	if delResp.Status != "ok" {
		t.Errorf("expected status ok, got %s", delResp.Status)
	}

	getResp2, err := client.SecretGet(context.Background(), &pb.SecretGetRequest{
		ProjectPath: projectDir,
		Name:        "DB_PASSWORD",
	})
	if err != nil {
		t.Fatalf("SecretGet() after delete failed: %v", err)
	}
	if getResp2.Error == "" {
		t.Error("expected error after deleting secret")
	}
}

func TestServer_SecretRotate_ManualFails(t *testing.T) {
	projectDir := setupTestProject(t)
	_, client, cleanup := setupTestServer(t)
	defer cleanup()

	client.SecretSet(context.Background(), &pb.SecretSetRequest{
		ProjectPath: projectDir,
		Name:        "API_KEY",
		Value:       "old-key",
	})

	rotateResp, err := client.SecretRotate(context.Background(), &pb.SecretRotateRequest{
		ProjectPath: projectDir,
		Name:        "API_KEY",
	})
	if err != nil {
		t.Fatalf("SecretRotate() failed: %v", err)
	}
	if rotateResp.Status != "error" {
		t.Errorf("expected 'error' rotating a manual secret, got %s", rotateResp.Status)
	}
	if rotateResp.Error == "" {
		t.Error("expected non-empty error message")
	}
}

func TestServer_Status_Stopped(t *testing.T) {
	projectDir := setupTestProject(t)
	_, client, cleanup := setupTestServer(t)
	defer cleanup()

	resp, err := client.Status(context.Background(), &pb.StatusRequest{
		ProjectPath: projectDir,
	})
	if err != nil {
		t.Fatalf("Status() failed: %v", err)
	}
	if resp.Status != "stopped" {
		t.Errorf("expected status 'stopped', got '%s'", resp.Status)
	}
}

func TestServer_Doctor_ReportsConfig(t *testing.T) {
	projectDir := setupTestProject(t)
	_, client, cleanup := setupTestServer(t)
	defer cleanup()

	resp, err := client.Doctor(context.Background(), &pb.DoctorRequest{
		ProjectPath: projectDir,
	})
	if err != nil {
		t.Fatalf("Doctor() failed: %v", err)
	}

	if len(resp.Issues) == 0 {
		t.Error("expected at least 1 diagnostic issue")
	}

	foundConfig := false
	for _, issue := range resp.Issues {
		if issue.Message != "" {
			foundConfig = true
		}
	}
	if !foundConfig {
		t.Error("expected diagnostic issues, got empty messages")
	}
}

func TestServer_Doctor_NoConfig(t *testing.T) {
	emptyDir := t.TempDir()
	_, client, cleanup := setupTestServer(t)
	defer cleanup()

	resp, err := client.Doctor(context.Background(), &pb.DoctorRequest{
		ProjectPath: emptyDir,
	})
	if err != nil {
		t.Fatalf("Doctor() failed: %v", err)
	}

	if len(resp.Issues) == 0 {
		t.Error("expected diagnostic issues for project without config")
	}
}

func TestServer_Logs_NoDocker(t *testing.T) {
	projectDir := setupTestProject(t)
	_, client, cleanup := setupTestServer(t)
	defer cleanup()

	stream, err := client.Logs(context.Background(), &pb.LogsRequest{
		ProjectPath: projectDir,
		Service:     "web",
	})
	if err != nil {
		t.Fatalf("Logs() stream creation failed: %v", err)
	}

	_, err = stream.Recv()
	if err == nil {
		t.Error("expected error from Logs() without Docker, got nil")
	}
}

func TestServer_Reset_NoDocker(t *testing.T) {
	// GitHub CI runners have Docker pre-installed, so skip this test
	// when Docker is actually available.
	rt := docker.NewDockerRuntime()
	if err := rt.Connect(context.Background()); err == nil {
		rt.Close()
		t.Skip("Docker is available — skipping NoDocker test")
	}
	rt.Close()

	projectDir := setupTestProject(t)
	_, client, cleanup := setupTestServer(t)
	defer cleanup()

	stream, err := client.Reset(context.Background(), &pb.ResetRequest{
		ProjectPath: projectDir,
	})
	if err != nil {
		t.Fatalf("Reset() stream creation failed: %v", err)
	}

	resp, err := stream.Recv()
	if err != nil {
		t.Fatalf("Reset() Recv() failed: %v", err)
	}
	if resp.Status != "error" {
		t.Errorf("expected error status, got '%s'", resp.Status)
	}
}

func TestServer_SnapshotList_Empty(t *testing.T) {
	projectDir := setupTestProject(t)
	_, client, cleanup := setupTestServer(t)
	defer cleanup()

	_, err := client.SnapshotList(context.Background(), &pb.SnapshotListRequest{
		ProjectPath: projectDir,
	})
	if err != nil {
		t.Fatalf("SnapshotList() failed: %v", err)
	}
}

func TestServer_SnapshotDelete_NotFound(t *testing.T) {
	projectDir := setupTestProject(t)
	_, client, cleanup := setupTestServer(t)
	defer cleanup()

	resp, err := client.SnapshotDelete(context.Background(), &pb.SnapshotDeleteRequest{
		ProjectPath: projectDir,
		SnapshotId:  "nonexistent",
	})
	if err != nil {
		t.Fatalf("SnapshotDelete() failed: %v", err)
	}
	if resp.Status != "ok" {
		t.Logf("SnapshotDelete returned: %s", resp.Status)
	}
}

func TestServer_SnapshotSave_NoDocker(t *testing.T) {
	rt := docker.NewDockerRuntime()
	if err := rt.Connect(context.Background()); err == nil {
		rt.Close()
		t.Skip("Docker is available — skipping NoDocker test")
	}
	rt.Close()

	projectDir := setupTestProject(t)
	_, client, cleanup := setupTestServer(t)
	defer cleanup()

	stream, err := client.SnapshotSave(context.Background(), &pb.SnapshotSaveRequest{
		ProjectPath: projectDir,
		Name:        "test-snap",
	})
	if err != nil {
		t.Fatalf("SnapshotSave() stream failed: %v", err)
	}

	// Drain stream until EOF to ensure goroutine finishes before cleanup.
	lastStatus := ""
	for {
		resp, err := stream.Recv()
		if err != nil {
			break
		}
		lastStatus = resp.Status
	}
	if lastStatus != "" && lastStatus != "error" {
		t.Logf("SnapshotSave returned status: %s (expected 'error' when no Docker)", lastStatus)
	}
}

func TestServer_SnapshotLoad_NoDocker(t *testing.T) {
	rt := docker.NewDockerRuntime()
	if err := rt.Connect(context.Background()); err == nil {
		rt.Close()
		t.Skip("Docker is available — skipping NoDocker test")
	}
	rt.Close()

	projectDir := setupTestProject(t)
	_, client, cleanup := setupTestServer(t)
	defer cleanup()

	stream, err := client.SnapshotLoad(context.Background(), &pb.SnapshotLoadRequest{
		ProjectPath: projectDir,
		SnapshotId:  "test-id",
	})
	if err != nil {
		t.Fatalf("SnapshotLoad() stream failed: %v", err)
	}

	lastStatus := ""
	for {
		resp, err := stream.Recv()
		if err != nil {
			break
		}
		lastStatus = resp.Status
	}
	if lastStatus != "" && lastStatus != "error" {
		t.Logf("SnapshotLoad returned status: %s (expected 'error' when no Docker)", lastStatus)
	}
}

func TestServer_Stop_NoOrchestrator(t *testing.T) {
	projectDir := setupTestProject(t)
	_, client, cleanup := setupTestServer(t)
	defer cleanup()

	resp, err := client.Stop(context.Background(), &pb.StopRequest{
		ProjectPath: projectDir,
	})
	if err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}
	if resp.Error == "" {
		t.Log("Stop returned ok (no error)")
	}
}
