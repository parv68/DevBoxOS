package networking

import "testing"

func TestEgressPolicy_DefaultDeny(t *testing.T) {
	e := NewEgressPolicy("")

	if mode := e.GetMode(); mode != "default-deny" {
		t.Errorf("expected default-deny mode, got %s", mode)
	}

	if e.IsAllowed("web", "google.com") {
		t.Error("expected default-deny to block, got allowed")
	}
}

func TestEgressPolicy_AllowAll(t *testing.T) {
	e := NewEgressPolicy("allow-all")

	if e.IsAllowed("web", "google.com") != true {
		t.Error("expected allow-all to allow, got denied")
	}
	if e.IsAllowed("api", "10.0.0.1") != true {
		t.Error("expected allow-all to allow any destination")
	}
}

func TestEgressPolicy_AddRule(t *testing.T) {
	e := NewEgressPolicy("")

	e.AddRule("web", "api.local")
	e.AddRule("web", "db.local")

	if !e.IsAllowed("web", "api.local") {
		t.Error("expected web to be allowed to api.local")
	}
	if !e.IsAllowed("web", "db.local") {
		t.Error("expected web to be allowed to db.local")
	}
	if e.IsAllowed("web", "google.com") {
		t.Error("expected web to be denied to google.com")
	}
}

func TestEgressPolicy_WildcardRule(t *testing.T) {
	e := NewEgressPolicy("")

	e.AddRule("web", "*")

	if !e.IsAllowed("web", "anything.local") {
		t.Error("expected wildcard to allow all destinations")
	}
}

func TestEgressPolicy_ServiceNotListed(t *testing.T) {
	e := NewEgressPolicy("")

	e.AddRule("web", "db.local")

	if e.IsAllowed("api", "db.local") {
		t.Error("expected api to be denied (no rules)")
	}
}

func TestEgressPolicy_MultipleServices(t *testing.T) {
	e := NewEgressPolicy("")

	e.AddRule("web", "api.local")
	e.AddRule("api", "db.local")

	if !e.IsAllowed("web", "api.local") {
		t.Error("expected web to be allowed to api.local")
	}
	if !e.IsAllowed("api", "db.local") {
		t.Error("expected api to be allowed to db.local")
	}
	if e.IsAllowed("web", "db.local") {
		t.Error("expected web to be denied to db.local")
	}
}

func TestCheckPortAvailability(t *testing.T) {
	err := CheckPortAvailability("0")
	if err == nil {
		t.Log("port 0 should be available (no real check)")
	}
}
