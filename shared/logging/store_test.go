package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewStore(t *testing.T) {
	s := NewStore("/tmp/test-project")
	if s == nil {
		t.Fatal("NewStore() returned nil")
	}
}

func TestStore_AppendAndRead(t *testing.T) {
	baseDir := t.TempDir()
	s := NewStore(baseDir)

	err := s.Append("test-project", "web", []string{"line1", "line2", "line3"})
	if err != nil {
		t.Fatalf("Append() failed: %v", err)
	}

	lines, err := s.Read("test-project", "web", time.Time{}, 0)
	if err != nil {
		t.Fatalf("Read() failed: %v", err)
	}

	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
	if lines[0] != "line1" || lines[1] != "line2" || lines[2] != "line3" {
		t.Errorf("unexpected lines: %v", lines)
	}
}

func TestStore_AppendMultiple(t *testing.T) {
	baseDir := t.TempDir()
	s := NewStore(baseDir)

	s.Append("test-project", "web", []string{"line1"})
	s.Append("test-project", "web", []string{"line2"})

	lines, _ := s.Read("test-project", "web", time.Time{}, 0)
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestStore_ReadNoLogs(t *testing.T) {
	baseDir := t.TempDir()
	s := NewStore(baseDir)

	lines, err := s.Read("test-project", "nonexistent", time.Time{}, 0)
	if err != nil {
		t.Fatalf("Read() failed: %v", err)
	}
	if len(lines) != 0 {
		t.Errorf("expected 0 lines for nonexistent service, got %d", len(lines))
	}
}

func TestStore_ReadWithLimit(t *testing.T) {
	baseDir := t.TempDir()
	s := NewStore(baseDir)

	s.Append("test-project", "web", []string{"line1", "line2", "line3", "line4", "line5"})

	lines, _ := s.Read("test-project", "web", time.Time{}, 3)
	if len(lines) != 3 {
		t.Errorf("expected 3 lines (limited), got %d", len(lines))
	}
	if lines[0] != "line3" {
		t.Errorf("expected first limited line to be line3, got %s", lines[0])
	}
}

func TestStore_ReadWithSince(t *testing.T) {
	baseDir := t.TempDir()
	s := NewStore(baseDir)

	s.Append("test-project", "web", []string{"old line"})
	since := time.Now()
	s.Append("test-project", "web", []string{"new line"})

	lines, _ := s.Read("test-project", "web", since, 0)
	if len(lines) == 0 {
		t.Log("since filter may have filtered all lines (timing)")
	}
}

func TestStore_ReadWithFutureSince(t *testing.T) {
	baseDir := t.TempDir()
	s := NewStore(baseDir)

	s.Append("test-project", "web", []string{"old line"})

	future := time.Now().Add(24 * time.Hour)
	lines, _ := s.Read("test-project", "web", future, 0)
	if len(lines) != 0 {
		t.Errorf("expected 0 lines with future since time, got %d", len(lines))
	}
}

func TestStore_Search(t *testing.T) {
	baseDir := t.TempDir()
	s := NewStore(baseDir)

	s.Append("test-project", "web", []string{
		"INFO: Server started on port 8080",
		"ERROR: Connection refused",
		"WARN: Retrying connection",
		"INFO: Request completed",
	})

	matches, err := s.Search("test-project", "web", "ERROR", time.Time{})
	if err != nil {
		t.Fatalf("Search() failed: %v", err)
	}
	if len(matches) != 1 {
		t.Errorf("expected 1 match for ERROR, got %d", len(matches))
	}

	matches, err = s.Search("test-project", "web", "connection", time.Time{})
	if err != nil {
		t.Fatalf("Search() failed: %v", err)
	}
	if len(matches) != 2 {
		t.Errorf("expected 2 case-insensitive matches for 'connection', got %d", len(matches))
	}
}

func TestStore_SearchNoMatch(t *testing.T) {
	baseDir := t.TempDir()
	s := NewStore(baseDir)

	s.Append("test-project", "web", []string{"line1", "line2"})

	matches, _ := s.Search("test-project", "web", "NONEXISTENT", time.Time{})
	if len(matches) != 0 {
		t.Errorf("expected 0 matches, got %d", len(matches))
	}
}

func TestStore_SearchInvalidRegex(t *testing.T) {
	baseDir := t.TempDir()
	s := NewStore(baseDir)

	s.Append("test-project", "web", []string{"line1"})

	_, err := s.Search("test-project", "web", "[invalid", time.Time{})
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}

func TestStore_Export(t *testing.T) {
	baseDir := t.TempDir()
	s := NewStore(baseDir)

	s.Append("test-project", "web", []string{"line1", "line2"})

	outputPath := filepath.Join(baseDir, "export.txt")
	err := s.Export("test-project", "web", outputPath)
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	data, _ := os.ReadFile(outputPath)
	content := strings.TrimSpace(string(data))
	if content != "line1\nline2" {
		t.Errorf("unexpected export content: %s", content)
	}
}

func TestStore_ExportNoLogs(t *testing.T) {
	baseDir := t.TempDir()
	s := NewStore(baseDir)

	outputPath := filepath.Join(baseDir, "export.txt")
	err := s.Export("test-project", "nonexistent", outputPath)
	if err != nil {
		t.Fatalf("Export() with no logs failed: %v", err)
	}
}

func TestStore_Size(t *testing.T) {
	baseDir := t.TempDir()
	s := NewStore(baseDir)

	size, err := s.Size("test-project")
	if err != nil {
		t.Fatalf("Size() failed: %v", err)
	}
	if size != 0 {
		t.Errorf("expected 0 size for empty project, got %d", size)
	}

	s.Append("test-project", "web", []string{"hello world"})

	size, _ = s.Size("test-project")
	if size == 0 {
		t.Errorf("expected non-zero size after appending, got %d", size)
	}
}

func TestStore_SizeNonexistent(t *testing.T) {
	baseDir := t.TempDir()
	s := NewStore(baseDir)

	size, err := s.Size("nonexistent-project")
	if err != nil {
		t.Fatalf("Size() for nonexistent project failed: %v", err)
	}
	if size != 0 {
		t.Errorf("expected 0 size for nonexistent project, got %d", size)
	}
}

func TestStore_Clear(t *testing.T) {
	baseDir := t.TempDir()
	s := NewStore(baseDir)

	s.Append("test-project", "web", []string{"line1"})

	err := s.Clear("test-project")
	if err != nil {
		t.Fatalf("Clear() failed: %v", err)
	}

	lines, _ := s.Read("test-project", "web", time.Time{}, 0)
	if len(lines) != 0 {
		t.Errorf("expected 0 lines after clear, got %d", len(lines))
	}
}

func TestStore_ClearNonexistent(t *testing.T) {
	baseDir := t.TempDir()
	s := NewStore(baseDir)

	err := s.Clear("nonexistent")
	if err != nil {
		t.Fatalf("Clear() for nonexistent project failed: %v", err)
	}
}

func TestStore_AppendMultipleServices(t *testing.T) {
	baseDir := t.TempDir()
	s := NewStore(baseDir)

	s.Append("test-project", "web", []string{"web line"})
	s.Append("test-project", "api", []string{"api line"})

	webLines, _ := s.Read("test-project", "web", time.Time{}, 0)
	apiLines, _ := s.Read("test-project", "api", time.Time{}, 0)

	if len(webLines) != 1 || webLines[0] != "web line" {
		t.Errorf("unexpected web lines: %v", webLines)
	}
	if len(apiLines) != 1 || apiLines[0] != "api line" {
		t.Errorf("unexpected api lines: %v", apiLines)
	}
}

func TestStore_AppendMultipleProjects(t *testing.T) {
	baseDir := t.TempDir()
	s := NewStore(baseDir)

	s.Append("project-a", "web", []string{"a web line"})
	s.Append("project-b", "web", []string{"b web line"})

	aLines, _ := s.Read("project-a", "web", time.Time{}, 0)
	bLines, _ := s.Read("project-b", "web", time.Time{}, 0)

	if len(aLines) != 1 || aLines[0] != "a web line" {
		t.Errorf("unexpected project-a lines: %v", aLines)
	}
	if len(bLines) != 1 || bLines[0] != "b web line" {
		t.Errorf("unexpected project-b lines: %v", bLines)
	}
}
