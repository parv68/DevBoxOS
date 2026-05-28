package networking

import "testing"

func TestDNSResolver_AddRecord(t *testing.T) {
	d := NewDNSResolver()
	d.AddRecord("web.local", "172.20.0.2")

	ip, ok := d.Resolve("web.local")
	if !ok {
		t.Fatal("expected web.local to resolve")
	}
	if ip != "172.20.0.2" {
		t.Errorf("expected 172.20.0.2, got %s", ip)
	}
}

func TestDNSResolver_RemoveRecord(t *testing.T) {
	d := NewDNSResolver()
	d.AddRecord("web.local", "172.20.0.2")
	d.RemoveRecord("web.local")

	_, ok := d.Resolve("web.local")
	if ok {
		t.Error("expected web.local to be removed")
	}
}

func TestDNSResolver_ResolveNotFound(t *testing.T) {
	d := NewDNSResolver()

	_, ok := d.Resolve("nonexistent.local")
	if ok {
		t.Error("expected nonexistent.local to not resolve")
	}
}

func TestDNSResolver_RegisterService(t *testing.T) {
	d := NewDNSResolver()

	d.RegisterService("web", "172.20.0.2", "testproject.local")

	shortIP, ok := d.Resolve("web.local")
	if !ok {
		t.Error("expected short name web.local to resolve")
	}
	if shortIP != "172.20.0.2" {
		t.Errorf("expected 172.20.0.2 for short name, got %s", shortIP)
	}

	fqdnIP, ok := d.Resolve("web.testproject.local")
	if !ok {
		t.Error("expected FQDN web.testproject.local to resolve")
	}
	if fqdnIP != "172.20.0.2" {
		t.Errorf("expected 172.20.0.2 for FQDN, got %s", fqdnIP)
	}
}

func TestDNSResolver_UnregisterService(t *testing.T) {
	d := NewDNSResolver()

	d.RegisterService("web", "172.20.0.2", "testproject.local")
	d.UnregisterService("web", "testproject.local")

	if _, ok := d.Resolve("web.local"); ok {
		t.Error("expected web.local to be removed after unregister")
	}
	if _, ok := d.Resolve("web.testproject.local"); ok {
		t.Error("expected web.testproject.local to be removed after unregister")
	}
}

func TestDNSResolver_ListRecords(t *testing.T) {
	d := NewDNSResolver()

	d.AddRecord("web.local", "172.20.0.2")
	d.AddRecord("api.local", "172.20.0.3")

	records := d.ListRecords()
	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}
	if records["web.local"] != "172.20.0.2" {
		t.Errorf("expected 172.20.0.2 for web.local, got %s", records["web.local"])
	}

	// Verify it's a copy
	records["web.local"] = "modified"
	original, _ := d.Resolve("web.local")
	if original == "modified" {
		t.Error("ListRecords should return a copy, not a reference")
	}
}

func TestDNSResolver_ConcurrentAccess(t *testing.T) {
	d := NewDNSResolver()
	done := make(chan bool)

	go func() {
		for i := 0; i < 100; i++ {
			d.AddRecord("web.local", "172.20.0.2")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			d.Resolve("web.local")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			d.ListRecords()
		}
		done <- true
	}()

	<-done
	<-done
	<-done
}
