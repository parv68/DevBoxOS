package orchestrator

import (
	"reflect"
	"testing"
)

func TestGraph_AddNode_Resolve(t *testing.T) {
	t.Run("no dependencies", func(t *testing.T) {
		g := NewGraph()
		g.AddNode("web", nil)
		g.AddNode("db", nil)

		result, err := g.Resolve()
		if err != nil {
			t.Fatalf("Resolve failed: %v", err)
		}
		if len(result) != 2 {
			t.Fatalf("got %d nodes, want 2", len(result))
		}
	})

	t.Run("simple dependency order", func(t *testing.T) {
		g := NewGraph()
		g.AddNode("web", []string{"db"})
		g.AddNode("db", nil)

		result, err := g.Resolve()
		if err != nil {
			t.Fatalf("Resolve failed: %v", err)
		}

		// db should come before web
		if len(result) != 2 {
			t.Fatalf("got %d nodes, want 2: %v", len(result), result)
		}
		if result[0] != "db" {
			t.Errorf("first node = %q, want %q (db must start before web)", result[0], "db")
		}
		if result[1] != "web" {
			t.Errorf("second node = %q, want %q", result[1], "web")
		}
	})

	t.Run("chained dependencies", func(t *testing.T) {
		g := NewGraph()
		g.AddNode("web", []string{"api"})
		g.AddNode("api", []string{"db"})
		g.AddNode("db", nil)

		result, err := g.Resolve()
		if err != nil {
			t.Fatalf("Resolve failed: %v", err)
		}

		if len(result) != 3 {
			t.Fatalf("got %d nodes, want 3", len(result))
		}

		// Verify order: db -> api -> web
		positions := make(map[string]int)
		for i, n := range result {
			positions[n] = i
		}
		if positions["db"] > positions["api"] {
			t.Errorf("db (%d) should come before api (%d)", positions["db"], positions["api"])
		}
		if positions["api"] > positions["web"] {
			t.Errorf("api (%d) should come before web (%d)", positions["api"], positions["web"])
		}
	})

	t.Run("multiple dependencies", func(t *testing.T) {
		g := NewGraph()
		g.AddNode("app", []string{"db", "cache"})
		g.AddNode("db", nil)
		g.AddNode("cache", []string{"db"})

		result, err := g.Resolve()
		if err != nil {
			t.Fatalf("Resolve failed: %v", err)
		}

		if len(result) != 3 {
			t.Fatalf("got %d nodes, want 3", len(result))
		}

		// db must be first (no deps, and both cache and app depend on it)
		if result[0] != "db" {
			t.Errorf("first node = %q, want %q", result[0], "db")
		}
	})

	t.Run("circular dependency", func(t *testing.T) {
		g := NewGraph()
		g.AddNode("a", []string{"b"})
		g.AddNode("b", []string{"a"})

		_, err := g.Resolve()
		if err == nil {
			t.Fatal("expected circular dependency error")
		}
	})

	t.Run("self dependency", func(t *testing.T) {
		g := NewGraph()
		g.AddNode("a", []string{"a"})

		_, err := g.Resolve()
		if err == nil {
			t.Fatal("expected circular dependency error")
		}
	})
}

func TestGraph_Reverse(t *testing.T) {
	g := NewGraph()
	g.AddNode("web", []string{"api"})
	g.AddNode("api", []string{"db"})
	g.AddNode("db", nil)

	order, err := g.Reverse()
	if err != nil {
		t.Fatalf("Reverse failed: %v", err)
	}

	if len(order) != 3 {
		t.Fatalf("got %d nodes, want 3", len(order))
	}

	// Reverse order shoud be web first, then api, then db
	if order[0] != "web" {
		t.Errorf("first node in reverse = %q, want %q", order[0], "web")
	}
	if order[2] != "db" {
		t.Errorf("last node in reverse = %q, want %q", order[2], "db")
	}
}

func TestGraph_Empty(t *testing.T) {
	g := NewGraph()
	result, err := g.Resolve()
	if err != nil {
		t.Fatalf("Resolve failed on empty graph: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("got %d nodes, want 0", len(result))
	}
}

func TestGraph_Deterministic(t *testing.T) {
	// Two runs with same chained nodes should produce same order
	g1 := NewGraph()
	g1.AddNode("web", []string{"api"})
	g1.AddNode("api", []string{"db"})
	g1.AddNode("db", nil)

	g2 := NewGraph()
	g2.AddNode("web", []string{"api"})
	g2.AddNode("api", []string{"db"})
	g2.AddNode("db", nil)

	r1, _ := g1.Resolve()
	r2, _ := g2.Resolve()

	if !reflect.DeepEqual(r1, r2) {
		t.Errorf("results differ between runs: %v vs %v", r1, r2)
	}
	if len(r1) != 3 || r1[0] != "db" || r1[2] != "web" {
		t.Errorf("unexpected order: %v", r1)
	}
}
