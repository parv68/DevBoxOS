package orchestrator

import (
	"fmt"
)

// Graph resolves service dependency ordering using topological sort.
type Graph struct {
	nodes    map[string][]string // node -> dependencies
	inDegree map[string]int
}

// NewGraph creates a new dependency graph.
func NewGraph() *Graph {
	return &Graph{
		nodes:    make(map[string][]string),
		inDegree: make(map[string]int),
	}
}

// AddNode adds a service with its dependencies.
func (g *Graph) AddNode(name string, dependsOn []string) {
	g.nodes[name] = dependsOn
	if _, exists := g.inDegree[name]; !exists {
		g.inDegree[name] = 0
	}

	for _, dep := range dependsOn {
		g.inDegree[dep]++
		if _, exists := g.nodes[dep]; !exists {
			g.nodes[dep] = nil
		}
	}
}

// Resolve returns services in startup order (dependencies first).
func (g *Graph) Resolve() ([]string, error) {
	// Kahn's algorithm for topological sort
	queue := make([]string, 0)
	for node, degree := range g.inDegree {
		if degree == 0 {
			queue = append(queue, node)
		}
	}

	var result []string
	for len(queue) > 0 {
		// Dequeue
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		// Reduce in-degree for dependents
		for _, dep := range g.nodes[node] {
			g.inDegree[dep]--
			if g.inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	if len(result) != len(g.inDegree) {
		return nil, fmt.Errorf("circular dependency detected: not all services could be resolved")
	}

	return result, nil
}

// Reverse returns services in shutdown order (dependents first).
func (g *Graph) Reverse() ([]string, error) {
	startOrder, err := g.Resolve()
	if err != nil {
		return nil, err
	}

	// Reverse the startup order
	reversed := make([]string, len(startOrder))
	for i, v := range startOrder {
		reversed[len(startOrder)-1-i] = v
	}

	return reversed, nil
}
