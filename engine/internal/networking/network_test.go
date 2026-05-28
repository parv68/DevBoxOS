package networking

import (
	"testing"
)

func TestProjectNetwork_GetHostname(t *testing.T) {
	nw := &ProjectNetwork{
		Name:   "devbox-testproject",
		Domain: "testproject.local",
	}

	hostname := nw.GetHostname("web")
	expected := "web.testproject.local"
	if hostname != expected {
		t.Errorf("GetHostname() = %s, want %s", hostname, expected)
	}
}

func TestProjectNetwork_GetHostname_OtherService(t *testing.T) {
	nw := &ProjectNetwork{
		Name:   "devbox-myapp",
		Domain: "myapp.local",
	}

	hostname := nw.GetHostname("api")
	expected := "api.myapp.local"
	if hostname != expected {
		t.Errorf("GetHostname() = %s, want %s", hostname, expected)
	}
}

func TestProjectNetwork_RegisterContainer(t *testing.T) {
	nw := &ProjectNetwork{
		Name:       "devbox-test",
		Domain:     "test.local",
		Containers: make(map[string]string),
	}

	nw.RegisterContainer("web", "abc123")
	nw.RegisterContainer("api", "def456")

	if nw.Containers["web"] != "abc123" {
		t.Errorf("expected container web=abc123, got %s", nw.Containers["web"])
	}
	if nw.Containers["api"] != "def456" {
		t.Errorf("expected container api=def456, got %s", nw.Containers["api"])
	}
}

func TestProjectNetwork_RegisterContainerOverwrite(t *testing.T) {
	nw := &ProjectNetwork{
		Name:       "devbox-test",
		Domain:     "test.local",
		Containers: make(map[string]string),
	}

	nw.RegisterContainer("web", "abc123")
	nw.RegisterContainer("web", "xyz789")

	if nw.Containers["web"] != "xyz789" {
		t.Errorf("expected container web=xyz789, got %s", nw.Containers["web"])
	}
}
