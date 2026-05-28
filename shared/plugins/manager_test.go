package plugins

import (
	"context"
	"testing"
	"time"

	"github.com/devboxos/devboxos/shared/types"
)

func TestNewManager_Empty(t *testing.T) {
	m := NewManager("/tmp/test", nil)
	if m == nil {
		t.Fatal("NewManager() returned nil")
	}
	if len(m.List()) != 0 {
		t.Errorf("expected no plugins, got %d", len(m.List()))
	}
}

func TestNewManager_EmptySlice(t *testing.T) {
	m := NewManager("/tmp/test", []types.Plugin{})
	if len(m.List()) != 0 {
		t.Errorf("expected no plugins, got %d", len(m.List()))
	}
}

func TestNewManager_WithPlugins(t *testing.T) {
	plugins := []types.Plugin{
		{
			Name:    "notify",
			Command: "echo started",
			On:      []string{"post-start"},
		},
		{
			Name:    "backup",
			Command: "backup.sh",
			On:      []string{"pre-stop", "post-stop"},
		},
	}

	m := NewManager("/tmp/test", plugins)
	if len(m.List()) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(m.List()))
	}
}

func TestManager_HasHook(t *testing.T) {
	plugins := []types.Plugin{
		{
			Name:    "notify",
			Command: "echo started",
			On:      []string{"post-start"},
		},
	}

	m := NewManager("/tmp/test", plugins)

	if !m.HasHook(HookPostStart) {
		t.Error("expected HasHook(HookPostStart) to be true")
	}
	if m.HasHook(HookPreStart) {
		t.Error("expected HasHook(HookPreStart) to be false")
	}
	if m.HasHook(HookPreStop) {
		t.Error("expected HasHook(HookPreStop) to be false")
	}
}

func TestManager_HasHook_NoPlugins(t *testing.T) {
	m := NewManager("/tmp/test", nil)

	if m.HasHook(HookPreStart) {
		t.Error("expected HasHook to be false with no plugins")
	}
}

func TestManager_List(t *testing.T) {
	plugins := []types.Plugin{
		{
			Name:    "test-plugin",
			Command: "echo hello",
			On:      []string{"post-start"},
			Timeout: 10,
		},
	}

	m := NewManager("/tmp/test", plugins)
	list := m.List()

	if len(list) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(list))
	}
	if list[0].Name != "test-plugin" {
		t.Errorf("expected name 'test-plugin', got %s", list[0].Name)
	}
	if list[0].Command != "echo hello" {
		t.Errorf("expected command 'echo hello', got %s", list[0].Command)
	}
	if len(list[0].On) != 1 || list[0].On[0] != HookPostStart {
		t.Errorf("unexpected hooks: %v", list[0].On)
	}
	if list[0].Timeout != 10*time.Second {
		t.Errorf("expected timeout 10s, got %v", list[0].Timeout)
	}
}

func TestManager_List_DefaultTimeout(t *testing.T) {
	plugins := []types.Plugin{
		{
			Name:    "test",
			Command: "echo test",
			On:      []string{"pre-start"},
		},
	}

	m := NewManager("/tmp/test", plugins)
	list := m.List()

	if list[0].Timeout != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", list[0].Timeout)
	}
}

func TestManager_ExecuteHook_HookNotFound(t *testing.T) {
	plugins := []types.Plugin{
		{
			Name:    "notify",
			Command: "echo started",
			On:      []string{"post-start"},
		},
	}

	m := NewManager("/tmp/test", plugins)
	err := m.ExecuteHook(context.Background(), HookPreStart, nil)
	if err != nil {
		t.Fatalf("ExecuteHook() should succeed when no plugins match: %v", err)
	}
}

func TestManager_HooksInto(t *testing.T) {
	p := Plugin{
		Name: "test",
		On:   []Hook{HookPreStart, HookPostStop},
	}

	if !p.hooksInto(HookPreStart) {
		t.Error("expected hooksInto(HookPreStart) to be true")
	}
	if !p.hooksInto(HookPostStop) {
		t.Error("expected hooksInto(HookPostStop) to be true")
	}
	if p.hooksInto(HookPostStart) {
		t.Error("expected hooksInto(HookPostStart) to be false")
	}
	if p.hooksInto(HookOnFailure) {
		t.Error("expected hooksInto(HookOnFailure) to be false")
	}
}

func TestManager_HooksInto_Empty(t *testing.T) {
	p := Plugin{
		Name: "test",
		On:   nil,
	}

	if p.hooksInto(HookPreStart) {
		t.Error("expected hooksInto to be false for nil hooks")
	}
}
