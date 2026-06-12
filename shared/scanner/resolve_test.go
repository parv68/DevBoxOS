package scanner

import (
	"testing"
)

func TestResolveConflictsNoConflicts(t *testing.T) {
	results := []ScanResult{
		{
			ServiceName: "api",
			Language:    "go",
			Ports:       []DetectedPort{{Port: 8080, Priority: 10}},
		},
		{
			ServiceName: "frontend",
			Language:    "node",
			Ports:       []DetectedPort{{Port: 3000, Priority: 10}},
		},
		{
			ServiceName: "worker",
			Language:    "python",
			Ports:       []DetectedPort{{Port: 8000, Priority: 10}},
		},
	}

	resolved, warnings := ResolveConflicts(results)
	if len(warnings.Conflicts) != 0 {
		t.Fatalf("expected no conflicts, got: %v", warnings.Conflicts)
	}
	if resolved["api"].ResolvedPort != "8080" {
		t.Fatalf("expected api port 8080, got %s", resolved["api"].ResolvedPort)
	}
	if resolved["frontend"].ResolvedPort != "3000" {
		t.Fatalf("expected frontend port 3000, got %s", resolved["frontend"].ResolvedPort)
	}
	if resolved["worker"].ResolvedPort != "8000" {
		t.Fatalf("expected worker port 8000, got %s", resolved["worker"].ResolvedPort)
	}
}

func TestResolveConflictsTwoServicesSamePort(t *testing.T) {
	results := []ScanResult{
		{
			ServiceName: "api",
			Language:    "go",
			Ports:       []DetectedPort{{Port: 8080, Priority: 10}},
		},
		{
			ServiceName: "admin",
			Language:    "node",
			Ports:       []DetectedPort{{Port: 8080, Priority: 10}},
		},
	}

	resolved, warnings := ResolveConflicts(results)
	if len(warnings.Conflicts) != 1 {
		t.Fatalf("expected 1 conflict warning, got %d: %v", len(warnings.Conflicts), warnings.Conflicts)
	}
	// Alphabetically first service keeps original port
	if resolved["admin"].ResolvedPort != "8080" {
		t.Fatalf("expected admin (alphabetically first) to keep port 8080, got %s", resolved["admin"].ResolvedPort)
	}
	if resolved["api"].ResolvedPort == "8080" {
		t.Fatalf("expected api to get different port, still got 8080")
	}
}

func TestResolveConflictsFiveServicesSamePort(t *testing.T) {
	results := make([]ScanResult, 5)
	names := []string{"svc-a", "svc-b", "svc-c", "svc-d", "svc-e"}
	for i := 0; i < 5; i++ {
		results[i] = ScanResult{
			ServiceName: names[i],
			Language:    "go",
			Ports:       []DetectedPort{{Port: 8080, Priority: 10}},
		}
	}

	resolved, warnings := ResolveConflicts(results)
	if len(warnings.Conflicts) != 4 {
		t.Fatalf("expected 4 conflict warnings for 5 services on same port, got %d", len(warnings.Conflicts))
	}
	usedPorts := make(map[string]bool)
	for _, svc := range resolved {
		if usedPorts[svc.ResolvedPort] {
			t.Fatalf("duplicate resolved port %s for service %s", svc.ResolvedPort, svc.Name)
		}
		usedPorts[svc.ResolvedPort] = true
	}
}

func TestResolveConflictsMixed(t *testing.T) {
	results := []ScanResult{
		{
			ServiceName: "backend",
			Language:    "go",
			Ports:       []DetectedPort{{Port: 8080, Priority: 10}},
		},
		{
			ServiceName: "frontend",
			Language:    "node",
			Ports:       []DetectedPort{{Port: 3000, Priority: 10}},
		},
		{
			ServiceName: "api",
			Language:    "go",
			Ports:       []DetectedPort{{Port: 8080, Priority: 10}},
		},
		{
			ServiceName: "admin",
			Language:    "node",
			Ports:       []DetectedPort{{Port: 8080, Priority: 10}},
		},
		{
			ServiceName: "worker",
			Language:    "python",
			Ports:       []DetectedPort{},
		},
	}

	resolved, warnings := ResolveConflicts(results)
	if len(warnings.Conflicts) != 2 {
		t.Fatalf("expected 2 conflict warnings, got %d: %v", len(warnings.Conflicts), warnings.Conflicts)
	}
	// Alphabetical order: admin, api, backend, frontend, worker
	// admin gets 8080 first, api gets 8081, backend gets 8082
	// frontend keeps 3000, worker has no port
	if resolved["admin"].ResolvedPort != "8080" {
		t.Fatalf("expected admin (alphabetically first) to keep 8080, got %s", resolved["admin"].ResolvedPort)
	}
	if resolved["frontend"].ResolvedPort != "3000" {
		t.Fatalf("expected frontend to keep 3000, got %s", resolved["frontend"].ResolvedPort)
	}
	if resolved["api"].ResolvedPort == "8080" || resolved["api"].ResolvedPort == "" {
		t.Fatalf("expected api to get reassigned, got %s", resolved["api"].ResolvedPort)
	}
	if resolved["backend"].ResolvedPort == "8080" || resolved["backend"].ResolvedPort == "" {
		t.Fatalf("expected backend to get reassigned, got %s", resolved["backend"].ResolvedPort)
	}
	// Worker has no detected ports but language=python → gets default 8000
	if resolved["worker"].ResolvedPort != "8000" {
		t.Fatalf("expected worker to get default python port 8000, got %s", resolved["worker"].ResolvedPort)
	}
}

func TestResolveConflictsSingleService(t *testing.T) {
	results := []ScanResult{
		{
			ServiceName: "api",
			Language:    "go",
			Ports:       []DetectedPort{{Port: 8080, Priority: 10}},
		},
	}

	resolved, warnings := ResolveConflicts(results)
	if len(warnings.Conflicts) != 0 {
		t.Fatalf("expected no conflicts, got: %v", warnings.Conflicts)
	}
	if resolved["api"].ResolvedPort != "8080" {
		t.Fatalf("expected 8080, got %s", resolved["api"].ResolvedPort)
	}
}

func TestResolveConflictsEmptyResults(t *testing.T) {
	resolved, warnings := ResolveConflicts(nil)
	if len(resolved) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(resolved))
	}
	if len(warnings.Conflicts) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings.Conflicts)
	}
}

func TestFormatWarnings(t *testing.T) {
	warnings := ResolutionWarnings{
		Conflicts: []string{
			"api → 8081 (was 8080, conflicted with backend)",
		},
	}
	msg := FormatWarnings(warnings)
	if msg == "" {
		t.Fatal("expected non-empty warning message")
	}
	if len(warnings.Conflicts) == 0 {
		t.Fatal("expected no format for empty warnings")
	}
}

func TestFormatWarningsEmpty(t *testing.T) {
	msg := FormatWarnings(ResolutionWarnings{})
	if msg != "" {
		t.Fatalf("expected empty string, got %q", msg)
	}
}

func TestFindResult(t *testing.T) {
	results := []ScanResult{
		{ServiceName: "api"},
		{ServiceName: "frontend"},
	}
	r := findResult("api", results)
	if r == nil {
		t.Fatal("expected to find api")
	}
	r = findResult("nonexistent", results)
	if r != nil {
		t.Fatal("expected nil for nonexistent")
	}
}
